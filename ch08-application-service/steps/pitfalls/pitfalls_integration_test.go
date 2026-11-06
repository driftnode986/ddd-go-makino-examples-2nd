//go:build integration

package pitfalls_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"example.com/trainingbooking/steps/txrepos"
	"example.com/trainingbooking/steps/updatefn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// recordingOutbox は渡されたイベントを覚える。
type recordingOutbox struct {
	mu     sync.Mutex
	events []domain.DomainEvent
}

func (o *recordingOutbox) Append(
	_ context.Context, events ...domain.DomainEvent,
) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, events...)
	return nil
}

func (o *recordingOutbox) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.events)
}

// addConfirmed は確定済みの予約を保存して ID を返す。
func addConfirmed(
	t *testing.T,
	r domain.BookingRepository,
	tx application.Transactor,
) ids.BookingID {
	t.Helper()
	b := newBooking(t)
	if err := b.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		return r.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return b.ID()
}

// #region outer-read

func TestOuterContextReadsOutsideTheTx(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	b := newBooking(t)

	outer := t.Context()
	err := tx.Do(outer, func(inner context.Context) error {
		if err := repo.Add(inner, b); err != nil {
			return err
		}
		_, err := repo.FindByID(outer, b.ID())
		t.Logf("外側の ctx で読む: %v", err)
		_, err = repo.FindByID(inner, b.ID())
		t.Logf("渡された ctx で読む: %v", err)
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
}

// #endregion

// #region bound-read

func TestBoundStoresReadInsideTheTx(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	run := txrepos.PgRunner{
		Bookings: repo, Outbox: &recordingOutbox{},
		Tx: pgtx.NewTransactor(pool),
	}
	b := newBooking(t)

	outer := t.Context()
	err := run.Do(outer, func(s txrepos.Stores) error {
		if err := s.Bookings.Add(outer, b); err != nil {
			return err
		}
		_, err := s.Bookings.FindByID(outer, b.ID())
		t.Logf("外側の ctx で読む: %v", err)
		return err
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
}

// #endregion

// 同じ条件で、outbox が失敗したら3つの形のどれも予約を戻す。
func TestAlternativesRollBackAlike(t *testing.T) {
	cancelWith := map[string]func(
		context.Context, domain.BookingRepository,
		application.OutboxWriter, application.Transactor,
		ids.BookingID,
	) error{
		"context": func(
			ctx context.Context, r domain.BookingRepository,
			o application.OutboxWriter, tx application.Transactor,
			id ids.BookingID,
		) error {
			_, err := application.NewCancelBooking(r, o, tx, clock).
				Execute(ctx, application.CancelBookingInput{
					BookingID: id.String(),
				})
			return err
		},
		"txrepos": func(
			ctx context.Context, r domain.BookingRepository,
			o application.OutboxWriter, tx application.Transactor,
			id ids.BookingID,
		) error {
			return txrepos.Cancel(ctx, txrepos.PgRunner{
				Bookings: r, Outbox: o, Tx: tx,
			}, id, clock)
		},
		"updatefn": func(
			ctx context.Context, r domain.BookingRepository,
			o application.OutboxWriter, tx application.Transactor,
			id ids.BookingID,
		) error {
			return updatefn.Cancel(ctx, updatefn.Repository{
				Bookings: r, Outbox: o, Tx: tx,
			}, id, clock)
		},
	}
	for name, cancel := range cancelWith {
		t.Run(name, func(t *testing.T) {
			pool := dbtest.New(t, postgres.Migrations)
			repo := postgres.NewBookingRepository(pool)
			tx := pgtx.NewTransactor(pool)
			id := addConfirmed(t, repo, tx)

			out := &flakyOutbox{next: &recordingOutbox{}, fails: 1}
			err := cancel(t.Context(), repo, out, tx, id)
			if !errors.Is(err, errOutbox) {
				t.Fatalf("err = %v", err)
			}
			got, err := repo.FindByID(t.Context(), id)
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			if got.Status() != domain.StatusConfirmed {
				t.Fatalf("Status = %s", got.Status())
			}
			// 成功の経路も同じ結果になる
			err = cancel(t.Context(), repo, out, tx, id)
			if err != nil {
				t.Fatalf("2回目: %v", err)
			}
			got, err = repo.FindByID(t.Context(), id)
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			if got.Status() != domain.StatusCancelled {
				t.Fatalf("Status = %s", got.Status())
			}
		})
	}
}

// interleaveRepo は読み出しの直後に1度だけ before を実行する。
type interleaveRepo struct {
	domain.BookingRepository
	before *func()
}

func (r interleaveRepo) FindByID(
	ctx context.Context, id ids.BookingID,
) (*domain.Booking, error) {
	b, err := r.BookingRepository.FindByID(ctx, id)
	if f := *r.before; f != nil {
		*r.before = nil
		f()
	}
	return b, err
}

func newRepo(
	t *testing.T,
) (*postgres.BookingRepository, *pgtx.Transactor) {
	t.Helper()
	pool := dbtest.New(t, postgres.Migrations)
	return postgres.NewBookingRepository(pool),
		pgtx.NewTransactor(pool)
}

// #region reread

func TestConflictThenRereadJudgesAgain(t *testing.T) {
	repo, tx := newRepo(t)
	outbox := &recordingOutbox{}
	id := addConfirmed(t, repo, tx)
	in := application.CancelBookingInput{BookingID: id.String()}
	other := application.NewCancelBooking(repo, outbox, tx, clock)
	// 読み出しと保存の間に、別のキャンセルが先にコミットする
	before := func() {
		if _, err := other.Execute(t.Context(), in); err != nil {
			t.Fatalf("別のキャンセル: %v", err)
		}
	}
	mine := application.NewCancelBooking(
		interleaveRepo{repo, &before}, outbox, tx, clock)

	for i := range 2 {
		_, err := mine.Execute(t.Context(), in)
		got, ok := errors.AsType[*application.ApplicationError](err)
		if !ok {
			t.Fatalf("err = %v", err)
		}
		t.Logf("%d回目: %s: %s", i+1, got.Kind, got.Message)
	}
	if outbox.count() != 1 {
		t.Fatalf("events = %d", outbox.count())
	}
	t.Logf("イベント: %d 件", outbox.count())
}

// #endregion

// countingTracer は SQL の実行回数を数える。
type countingTracer struct{ n atomic.Int64 }

func (c *countingTracer) TraceQueryStart(
	ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData,
) context.Context {
	c.n.Add(1)
	return ctx
}

func (*countingTracer) TraceQueryEnd(
	context.Context, *pgx.Conn, pgx.TraceQueryEndData,
) {
}

// #region n-plus-one

func TestListByFindByIDRunsManyQueries(t *testing.T) {
	const bookings = 20
	base := dbtest.New(t, postgres.Migrations)
	cfg := base.Config()
	counter := &countingTracer{}
	cfg.ConnConfig.Tracer = counter
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	t.Cleanup(pool.Close)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	queries := postgres.NewBookingQueries(pool)
	member := addMany(t, repo, tx, bookings)

	ctx := t.Context()
	counter.n.Store(0)
	const listIDs = "SELECT id FROM bookings WHERE member_id = $1"
	rows, err := pool.Query(ctx, listIDs, member.String())
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	found, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("CollectRows: %v", err)
	}
	for _, s := range found {
		id, err := ids.ParseID[ids.BookingMarker](s)
		if err != nil {
			t.Fatalf("ParseID: %v", err)
		}
		if _, err := repo.FindByID(ctx, id); err != nil {
			t.Fatalf("FindByID: %v", err)
		}
	}
	t.Logf("集約を1件ずつ読む: SQL %d 回", counter.n.Load())

	counter.n.Store(0)
	views, err := queries.ListByMember(ctx, member)
	if err != nil || len(views) != bookings {
		t.Fatalf("ListByMember: %d, %v", len(views), err)
	}
	t.Logf("ListByMember: SQL %d 回", counter.n.Load())
}

// #endregion

func addMany(
	t *testing.T, r domain.BookingRepository,
	tx application.Transactor, n int,
) ids.MemberID {
	t.Helper()
	member := ids.NewID[ids.MemberMarker]()
	for i := range n {
		at := start.Add(time.Duration(i) * 24 * time.Hour)
		b := bookingFor(t, member, at)
		err := tx.Do(t.Context(), func(ctx context.Context) error {
			return r.Add(ctx, b)
		})
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	return member
}

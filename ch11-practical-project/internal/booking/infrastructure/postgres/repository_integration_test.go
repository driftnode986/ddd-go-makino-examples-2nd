//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/domain/repositorytest"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.New(t, postgres.Migrations)
}

func TestBookingRepository_Contract(t *testing.T) {
	repositorytest.Run(t,
		func(t *testing.T) (
			domain.BookingRepository, repositorytest.Transactor,
		) {
			pool := newPool(t)
			return postgres.NewBookingRepository(pool),
				pgtx.NewTransactor(pool)
		})
}

// btree_gist で uuid と時間帯の排他制約が張れていることを、
// リポジトリを通さず SQL で確かめる。
func TestExclusionConstraint_UUIDWithBtreeGist(t *testing.T) {
	pool := newPool(t)
	ctx := t.Context()
	const insert = `INSERT INTO bookings
		(id, member_id, lesson_id, starts_at, ends_at,
		 fee_yen, discount_yen, status, version)
		VALUES ($1, $2, $3, $4, $5, 3000, 0, $6, 0)`
	member := ids.NewID[ids.MemberMarker]().String()
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, time.UTC)
	exec := func(
		who string, from time.Time, status string,
	) error {
		_, err := pool.Exec(ctx, insert,
			ids.NewID[ids.BookingMarker]().String(), who,
			ids.NewID[ids.LessonMarker]().String(),
			from, from.Add(time.Hour), status)
		return err
	}

	if err := exec(member, start, "pending"); err != nil {
		t.Fatalf("1件目: %v", err)
	}
	err := exec(member, start.Add(30*time.Minute), "confirmed")
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok || pgErr.Code != "23P01" {
		t.Fatalf("重なる予約: err = %v", err)
	}
	if pgErr.ConstraintName != "bookings_no_overlap" {
		t.Fatalf("制約 = %s", pgErr.ConstraintName)
	}
	// 接する時間帯、別の会員、終わった状態は制約にかからない
	err = exec(member, start.Add(time.Hour), "pending")
	if err != nil {
		t.Errorf("接する時間帯: %v", err)
	}
	other := ids.NewID[ids.MemberMarker]().String()
	if err := exec(other, start, "pending"); err != nil {
		t.Errorf("別の会員: %v", err)
	}
	if err := exec(member, start, "cancelled"); err != nil {
		t.Errorf("cancelled: %v", err)
	}
	if err := exec(member, start, "rejected"); err != nil {
		t.Errorf("rejected: %v", err)
	}
}

func TestTransactor_RollbackUndoesAddAndSave(t *testing.T) {
	pool := newPool(t)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	ctx := t.Context()
	saved, added := newPair(t)
	err := tx.Do(ctx, func(ctx context.Context) error {
		return repo.Add(ctx, saved)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	boom := errors.New("boom")
	err = tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.Add(ctx, added); err != nil {
			return err
		}
		loaded, err := repo.FindByID(ctx, saved.ID())
		if err != nil {
			return err
		}
		if err := loaded.Confirm(); err != nil {
			return err
		}
		if err := repo.Save(ctx, loaded); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Do: %v", err)
	}

	_, err = repo.FindByID(ctx, added.ID())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Add が残った: %v", err)
	}
	got, err := repo.FindByID(ctx, saved.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status() != domain.StatusPending || got.Version() != 0 {
		t.Errorf("Save が残った: %s v%d", got.Status(), got.Version())
	}
}

func TestTransactor_NestedJoinsOuter(t *testing.T) {
	pool := newPool(t)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	ctx := t.Context()
	inner, _ := newPair(t)

	boom := errors.New("boom")
	err := tx.Do(ctx, func(ctx context.Context) error {
		err := tx.Do(ctx, func(ctx context.Context) error {
			return repo.Add(ctx, inner)
		})
		if err != nil {
			return err
		}
		// 内側の Do の後でも確定していない
		if _, err := repo.FindByID(
			context.WithoutCancel(ctx), inner.ID()); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Do: %v", err)
	}
	_, err = repo.FindByID(ctx, inner.ID())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("外側の失敗で内側も戻らなかった: %v", err)
	}
}

// 重複で失敗した Add は、外側の Tx を失敗させて全体を戻す。
func TestTransactor_OverlapAbortsOuter(t *testing.T) {
	pool := newPool(t)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	ctx := t.Context()
	first, _ := newPair(t)

	err := tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.Add(ctx, first); err != nil {
			return err
		}
		return repo.Add(ctx, overlapping(t, first))
	})
	if !errors.Is(err, domain.ErrOverlap) {
		t.Fatalf("Do: %v", err)
	}
	_, err = repo.FindByID(ctx, first.ID())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("1件目が残った: %v", err)
	}
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC),
}

func newBooking(
	t *testing.T, member ids.MemberID, day int,
) *domain.Booking {
	t.Helper()
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, day, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](), member,
		ids.NewID[ids.LessonMarker](), slot, fee, clock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

// newPair は時間帯の重ならない予約を2件作る。
func newPair(t *testing.T) (a, b *domain.Booking) {
	t.Helper()
	member := ids.NewID[ids.MemberMarker]()
	return newBooking(t, member, 1), newBooking(t, member, 2)
}

func overlapping(t *testing.T, b *domain.Booking) *domain.Booking {
	t.Helper()
	return newBooking(t, b.MemberID(), 1)
}

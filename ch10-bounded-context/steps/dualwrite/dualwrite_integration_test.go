//go:build integration

// Package dualwrite は保存とイベントの送信を別々に行うと片方だけが
// 成功することを、本物の PostgreSQL で確かめる。
package dualwrite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC),
}

// broker はメッセージブローカーの代わりで、届いた数を数える。
type broker struct {
	down     bool
	received int
}

func (b *broker) publish() error {
	if b.down {
		return errors.New("broker is down")
	}
	b.received++
	return nil
}

type fixture struct {
	pool *pgxpool.Pool
	tx   *pgtx.Transactor
	repo *postgres.BookingRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.New(t, postgres.Migrations, outbox.Migrations)
	return fixture{
		pool: pool,
		tx:   pgtx.NewTransactor(pool),
		repo: postgres.NewBookingRepository(pool),
	}
}

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](), slot, fee, clock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

func (f fixture) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	err := f.pool.QueryRow(t.Context(),
		"SELECT count(*) FROM "+table).Scan(&n)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

var errStopped = errors.New("ここでプロセスが止まる")

// #region after-commit

// コミットしてから送る形。送れないと、予約だけが残る。
func TestPublishAfterCommit(t *testing.T) {
	f := newFixture(t)
	b := broker{down: true}
	save := func(ctx context.Context) error {
		return f.repo.Add(ctx, newBooking(t))
	}
	if err := f.tx.Do(t.Context(), save); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if err := b.publish(); err == nil {
		t.Fatal("送信は失敗するはず")
	}
	t.Logf("コミットの後に送る: 予約 %d 件, 届いたイベント %d 件",
		f.count(t, "bookings"), b.received)
}

// #endregion

// #region before-commit

// 送ってからコミットする形。コミットの前に止まると、イベントだけが
// 届く。席は、存在しない予約のために確保される。
func TestPublishBeforeCommit(t *testing.T) {
	f := newFixture(t)
	var b broker
	err := f.tx.Do(t.Context(), func(ctx context.Context) error {
		if err := f.repo.Add(ctx, newBooking(t)); err != nil {
			return err
		}
		if err := b.publish(); err != nil {
			return err
		}
		return errStopped
	})
	if !errors.Is(err, errStopped) {
		t.Fatalf("Do: %v", err)
	}
	t.Logf("コミットの前に送る: 予約 %d 件, 届いたイベント %d 件",
		f.count(t, "bookings"), b.received)
}

// #endregion

// outbox に書く形。どこで止まっても、予約とイベントの行は一致する。
func TestOutboxKeepsBoth(t *testing.T) {
	f := newFixture(t)
	writer := postgres.NewOutboxWriter(outbox.NewStore(f.pool))
	add := func(ctx context.Context, b *domain.Booking) error {
		if err := f.repo.Add(ctx, b); err != nil {
			return err
		}
		return writer.Append(ctx, b.PullEvents()...)
	}
	err := f.tx.Do(t.Context(), func(ctx context.Context) error {
		if err := add(ctx, newBooking(t)); err != nil {
			return err
		}
		return errStopped
	})
	if !errors.Is(err, errStopped) {
		t.Fatalf("Do: %v", err)
	}
	bookings, events := f.count(t, "bookings"), f.count(t, "outbox")
	if bookings != 0 || events != 0 {
		t.Errorf("bookings = %d, outbox = %d", bookings, events)
	}
	t.Logf("コミットの前に止まる: 予約 %d 件, outbox %d 件",
		bookings, events)

	err = f.tx.Do(t.Context(), func(ctx context.Context) error {
		return add(ctx, newBooking(t))
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	bookings, events = f.count(t, "bookings"), f.count(t, "outbox")
	if bookings != 1 || events != 1 {
		t.Errorf("bookings = %d, outbox = %d", bookings, events)
	}
	t.Logf("コミットの後に止まる: 予約 %d 件, outbox %d 件",
		bookings, events)
}

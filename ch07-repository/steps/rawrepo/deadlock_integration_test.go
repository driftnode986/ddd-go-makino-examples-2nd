//go:build integration

package rawrepo_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"example.com/trainingbooking/steps/rawrepo"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC),
}

func newBooking(
	t *testing.T, member ids.MemberID,
) *domain.Booking {
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
		ids.NewID[ids.BookingMarker](), member,
		ids.NewID[ids.LessonMarker](), slot, fee, clock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

func code(err error) string {
	if err == nil {
		return "成功"
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code
	}
	return err.Error()
}

// addBoth は同じ会員の重なる予約を2つの Tx で同時に保存する。
func addBoth(
	ctx context.Context, tx *pgtx.Transactor,
	r rawrepo.Repo, a, b *domain.Booking,
) (error, error) {
	errs := make(chan error, 2)
	start := make(chan struct{})
	for _, target := range []*domain.Booking{a, b} {
		go func() {
			<-start
			errs <- tx.Do(ctx, func(ctx context.Context) error {
				return r.Add(ctx, target)
			})
		}()
	}
	close(start)
	return <-errs, <-errs
}

// #region tally

func tally(
	t *testing.T, tx *pgtx.Transactor, r rawrepo.Repo, rounds int,
) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for range rounds {
		member := ids.NewID[ids.MemberMarker]()
		e1, e2 := addBoth(t.Context(), tx, r,
			newBooking(t, member), newBooking(t, member))
		counts[code(e1)]++
		counts[code(e2)]++
	}
	return counts
}

// #endregion

// #region compare

func TestConcurrentAddsWithAndWithoutLock(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	tx := pgtx.NewTransactor(pool)
	const rounds = 300

	begin := time.Now()
	got := tally(t, tx, rawrepo.Repo{Lock: false}, rounds)
	t.Logf("ロックなし %d 回: %v (%s)", rounds, got,
		time.Since(begin).Round(100*time.Millisecond))

	begin = time.Now()
	got = tally(t, tx, rawrepo.Repo{Lock: true}, rounds)
	t.Logf("ロックあり %d 回: %v (%s)", rounds, got,
		time.Since(begin).Round(100*time.Millisecond))
	if got["40P01"] != 0 {
		t.Errorf("ロックありでデッドロック: %d 件", got["40P01"])
	}
}

// #endregion

// デッドロックのエラーの中身を見せる。起きなければ試行を続ける。
func TestDeadlockMessage(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	tx := pgtx.NewTransactor(pool)
	for range 2000 {
		err := confirmWhileAdding(t, pool, tx, false)
		pgErr, ok := errors.AsType[*pgconn.PgError](err)
		if ok && pgErr.Code == "40P01" {
			t.Logf("%s (SQLSTATE %s)", pgErr.Message, pgErr.Code)
			t.Logf("%s", pgErr.Detail)
			return
		}
	}
	t.Log("2000 回の試行でデッドロックは起きなかった")
}

// #region save-and-add

// 確定（Save）と、それに重なる予約（Add）を同時に実行する。
// 確定は失敗する理由が無いので、成功してほしい。
func TestConfirmAndOverlappingAdd(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	tx := pgtx.NewTransactor(pool)
	for _, lock := range []bool{false, true} {
		saves := map[string]int{}
		for range 300 {
			saves[code(confirmWhileAdding(t, pool, tx, lock))]++
		}
		t.Logf("ロック %v の確定の結果: %v", lock, saves)
	}
}

// #endregion

// confirmWhileAdding は確定した Save の結果を返す。
func confirmWhileAdding(
	t *testing.T, pool *pgxpool.Pool, tx *pgtx.Transactor, lock bool,
) error {
	t.Helper()
	r := rawrepo.Repo{Lock: lock}
	repo := postgres.NewBookingRepository(pool)
	member := ids.NewID[ids.MemberMarker]()
	first := newBooking(t, member)
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		return repo.Add(ctx, first)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := first.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	second := newBooking(t, member)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		<-start
		_ = tx.Do(t.Context(), func(ctx context.Context) error {
			return r.Add(ctx, second)
		})
	})
	var saveErr error
	wg.Go(func() {
		<-start
		saveErr = tx.Do(t.Context(),
			func(ctx context.Context) error {
				return r.Save(ctx, first)
			})
	})
	close(start)
	wg.Wait()
	return saveErr
}

// #region other-member

// 同じ会員は待ち、別の会員は待たないことを確かめる。
func TestLockIsPerMember(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	tx := pgtx.NewTransactor(pool)
	repo := postgres.NewBookingRepository(pool)
	member := ids.NewID[ids.MemberMarker]()

	err := tx.Do(t.Context(), func(ctx context.Context) error {
		if err := repo.Add(ctx, newBooking(t, member)); err != nil {
			return err
		}
		t.Logf("同じ会員: %v", addWithin(t, tx, repo, member))
		other := ids.NewID[ids.MemberMarker]()
		t.Logf("別の会員: %v", addWithin(t, tx, repo, other))
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
}

// #endregion

// addWithin は別の Tx で1秒以内に予約を保存する。
func addWithin(
	t *testing.T, tx *pgtx.Transactor,
	repo *postgres.BookingRepository, member ids.MemberID,
) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	err := tx.Do(ctx, func(ctx context.Context) error {
		return repo.Add(ctx, newBooking(t, member))
	})
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return err
}

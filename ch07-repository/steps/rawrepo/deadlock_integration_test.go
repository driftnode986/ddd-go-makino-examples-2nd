//go:build integration

package rawrepo_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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

	got := tally(t, tx, rawrepo.Repo{Lock: false}, rounds)
	t.Logf("ロックなし %d 回: %v", rounds, got)

	got = tally(t, tx, rawrepo.Repo{Lock: true}, rounds)
	t.Logf("ロックあり %d 回: %v", rounds, got)
	if got["40P01"] != 0 {
		t.Errorf("ロックありでデッドロック: %d 件", got["40P01"])
	}
}

// #endregion

// デッドロックのエラーの中身を見せる。起きなければ試行を続ける。
func TestDeadlockMessage(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	tx := pgtx.NewTransactor(pool)
	r := oneTx{rawrepo.Repo{Lock: false}, tx}
	for range 2000 {
		addErr, saveErr := confirmWhileAdding(t, pool, tx, r)
		for _, err := range []error{addErr, saveErr} {
			pgErr, ok := errors.AsType[*pgconn.PgError](err)
			if ok && pgErr.Code == "40P01" {
				t.Logf("%s (SQLSTATE %s)",
					pgErr.Message, pgErr.Code)
				t.Logf("%s", pgErr.Detail)
				return
			}
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
		r := oneTx{rawrepo.Repo{Lock: lock}, tx}
		saves, adds := map[string]int{}, map[string]int{}
		for range 300 {
			addErr, saveErr := confirmWhileAdding(t, pool, tx, r)
			saves[code(saveErr)]++
			adds[code(addErr)]++
		}
		t.Logf("ロック %v 確定: %v 重なる予約: %v", lock, saves, adds)
	}
}

// #endregion

// #region retry

// ロックを取らず、40P01 を1回だけ再試行する形を動かす。
func TestRetryWithoutLock(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	tx := pgtx.NewTransactor(pool)
	r := rawrepo.Repo{Lock: false}
	saves, adds := map[string]int{}, map[string]int{}
	var retries atomic.Int64
	var slowest atomic.Int64
	for range 100 {
		addErr, saveErr := confirmWhileAdding(t, pool, tx,
			retryOnce{r, tx, &retries, &slowest})
		saves[code(saveErr)]++
		adds[code(addErr)]++
	}
	t.Logf("確定: %v 重なる予約: %v", saves, adds)
	t.Logf("再試行: %d 回 1回の最長: 約 %.0f 秒",
		retries.Load(), time.Duration(slowest.Load()).Seconds())
}

// #endregion

// retryOnce は 40P01 のときに1回だけ呼び直す。
type retryOnce struct {
	r       rawrepo.Repo
	tx      *pgtx.Transactor
	retries *atomic.Int64
	slowest *atomic.Int64
}

func (o retryOnce) run(
	ctx context.Context, fn func(context.Context) error,
) error {
	begin := time.Now()
	defer func() {
		d := int64(time.Since(begin))
		for {
			cur := o.slowest.Load()
			if d <= cur || o.slowest.CompareAndSwap(cur, d) {
				return
			}
		}
	}()
	err := o.tx.Do(ctx, fn)
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok || pgErr.Code != "40P01" {
		return err
	}
	o.retries.Add(1)
	return o.tx.Do(ctx, fn)
}

func (o retryOnce) add(
	ctx context.Context, b *domain.Booking,
) error {
	return o.run(ctx, func(ctx context.Context) error {
		return o.r.Add(ctx, b)
	})
}

func (o retryOnce) save(
	ctx context.Context, b *domain.Booking,
) error {
	return o.run(ctx, func(ctx context.Context) error {
		return o.r.Save(ctx, b)
	})
}

// saver は Add と Save を Tx の中で実行する。
type saver interface {
	add(context.Context, *domain.Booking) error
	save(context.Context, *domain.Booking) error
}

// oneTx は Add と Save をそれぞれ1つの Tx で実行する。
type oneTx struct {
	r  rawrepo.Repo
	tx *pgtx.Transactor
}

func (o oneTx) add(
	ctx context.Context, b *domain.Booking,
) error {
	return o.tx.Do(ctx, func(ctx context.Context) error {
		return o.r.Add(ctx, b)
	})
}

func (o oneTx) save(
	ctx context.Context, b *domain.Booking,
) error {
	return o.tx.Do(ctx, func(ctx context.Context) error {
		return o.r.Save(ctx, b)
	})
}

// confirmWhileAdding は重なる予約の Add と、確定の Save を同時に
// 実行して、それぞれの結果を返す。
func confirmWhileAdding(
	t *testing.T, pool *pgxpool.Pool, tx *pgtx.Transactor,
	target saver,
) (addErr, saveErr error) {
	t.Helper()
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
		addErr = target.add(t.Context(), second)
	})
	wg.Go(func() {
		<-start
		saveErr = target.save(t.Context(), first)
	})
	close(start)
	wg.Wait()
	return addErr, saveErr
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

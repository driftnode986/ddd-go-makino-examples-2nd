//go:build integration

// Package confirmdeadlock は確定の Save が 40P01 になったとき、配信処理
// の再配信で回復することを確かめる。第7章の再試行案の答え合わせである。
package confirmdeadlock_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres/sqlcgen"
	catalogcontract "example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC),
}

type fixture struct {
	pool  *pgxpool.Pool
	tx    *pgtx.Transactor
	repo  *postgres.BookingRepository
	store *outbox.Store
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.New(t, postgres.Migrations, outbox.Migrations)
	return fixture{
		pool:  pool,
		tx:    pgtx.NewTransactor(pool),
		repo:  postgres.NewBookingRepository(pool),
		store: outbox.NewStore(pool),
	}
}

func newBooking(
	t *testing.T, member ids.MemberID, option bool,
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
	if option {
		price, err := domain.NewMoney(300)
		if err != nil {
			t.Fatalf("NewMoney: %v", err)
		}
		o, err := domain.NewBookingOption("シューズ", price, 1)
		if err != nil {
			t.Fatalf("NewBookingOption: %v", err)
		}
		if err := b.AddOption(o, clock); err != nil {
			t.Fatalf("AddOption: %v", err)
		}
	}
	return b
}

// addWithSeatReserved は pending の予約と、それを確定する
// SeatReserved の未配信の行を作る。
func (f fixture) addWithSeatReserved(
	t *testing.T, b *domain.Booking,
) {
	t.Helper()
	body, err := json.Marshal(catalogcontract.SeatReserved{
		LessonID:   b.LessonID().String(),
		BookingID:  b.ID().String(),
		OccurredAt: clock.now,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	err = f.tx.Do(t.Context(), func(ctx context.Context) error {
		if err := f.repo.Add(ctx, b); err != nil {
			return err
		}
		return f.store.Append(ctx, outbox.Message{
			EventID:       outbox.NewEventID(),
			AggregateType: "lesson",
			AggregateID:   b.LessonID().String(),
			EventType:     catalogcontract.EventSeatReserved,
			Payload:       body,
			OccurredAt:    clock.now,
		})
	})
	if err != nil {
		t.Fatalf("addWithSeatReserved: %v", err)
	}
}

func (f fixture) relay(
	repo domain.BookingRepository, logs *bytes.Buffer,
) *outbox.Relay {
	results := application.NewSeatResultHandler(
		repo, outbox.NewProcessedEvents(f.pool), f.tx)
	relay := outbox.NewRelay(f.store,
		slog.New(slog.NewJSONHandler(logs, nil)),
		outbox.RelayOptions{})
	relay.Subscribe(catalogcontract.EventSeatReserved, "booking",
		outbox.On(results.OnSeatReserved))
	return relay
}

func status(
	t *testing.T, pool *pgxpool.Pool, id ids.BookingID,
) string {
	t.Helper()
	var s string
	err := pool.QueryRow(context.Background(),
		`SELECT status FROM bookings WHERE id = $1`,
		id.String()).Scan(&s)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	return s
}

func waitForLockWait(
	t *testing.T, pool *pgxpool.Pool, query string,
) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		err := pool.QueryRow(t.Context(),
			`SELECT count(*) FROM pg_stat_activity
			 WHERE datname = current_database()
			   AND wait_event_type = 'Lock' AND query LIKE $1`,
			"%"+query+"%").Scan(&n)
		if err != nil {
			t.Fatalf("pg_stat_activity: %v", err)
		}
		if n > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("%q が待ちに入らない", query)
}

// 順序を作って、本物のリポジトリの確定を必ず 40P01 にする。
//  1. 別の Tx が明細の行をロックする
//  2. 確定の Save が予約を更新し、明細の削除で 1 を待つ
//  3. 別の Tx が予約の行を更新しようとして 2 を待つ
//
// 別の Tx の deadlock_timeout を延ばしたので、確定の側が中止される。
func TestRedeliveryRecoversFromDeadlock(t *testing.T) {
	f := newFixture(t)
	b := newBooking(t, ids.NewID[ids.MemberMarker](), true)
	f.addWithSeatReserved(t, b)
	ctx := t.Context()

	other, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer func() { _ = other.Rollback(context.Background()) }()
	// この Tx の検査を遅らせ、中止される側を確定の Save に固定する。
	_, err = other.Exec(ctx, `SET LOCAL deadlock_timeout = '30s'`)
	if err != nil {
		t.Fatalf("SET LOCAL: %v", err)
	}
	_, err = other.Exec(ctx, `SELECT 1 FROM booking_options
		WHERE booking_id = $1 FOR UPDATE`, b.ID().String())
	if err != nil {
		t.Fatalf("FOR UPDATE: %v", err)
	}

	var logs bytes.Buffer
	relay := f.relay(f.repo, &logs)
	begin := time.Now()
	first := make(chan error, 1)
	go func() {
		_, err := relay.RunOnce(ctx)
		first <- err
	}()
	waitForLockWait(t, f.pool, "DELETE FROM booking_options")
	blocked := make(chan error, 1)
	go func() {
		_, err := other.Exec(ctx,
			`UPDATE bookings SET version = version WHERE id = $1`,
			b.ID().String())
		blocked <- err
	}()
	if err := <-first; err != nil {
		t.Fatalf("1回目の RunOnce: %v", err)
	}
	failed := time.Since(begin)
	if err := <-blocked; err != nil {
		t.Fatalf("別の Tx: %v", err)
	}
	if err := other.Rollback(ctx); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	var attempts int
	var lastError string
	err = f.pool.QueryRow(ctx,
		`SELECT attempts, last_error FROM outbox`).
		Scan(&attempts, &lastError)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if attempts != 1 || !strings.Contains(lastError, "40P01") {
		t.Fatalf("attempts = %d, last_error = %q",
			attempts, lastError)
	}
	if got := status(t, f.pool, b.ID()); got != "pending" {
		t.Fatalf("失敗の後の状態 = %s", got)
	}
	var processed int
	err = f.pool.QueryRow(ctx,
		`SELECT count(*) FROM processed_events`).Scan(&processed)
	if err != nil || processed != 0 {
		t.Fatalf("processed = %d, err = %v", processed, err)
	}
	t.Logf("1回目: 約 %.0f 秒で失敗, 状態 pending, 処理済みの記録 %d 件",
		failed.Seconds(), processed)
	t.Logf("原因: %s", lastError)

	n, err := relay.RunOnce(ctx)
	if err != nil || n != 1 {
		t.Fatalf("再配信: n = %d, err = %v", n, err)
	}
	if got := status(t, f.pool, b.ID()); got != "confirmed" {
		t.Fatalf("再配信の後の状態 = %s", got)
	}
	warns := strings.Count(logs.String(), `"level":"WARN"`)
	if warns != 1 {
		t.Fatalf("WARN = %d 行: %s", warns, logs.String())
	}
	t.Logf("再配信: 状態 confirmed, WARN のログ %d 行", warns)
}

// noLockRepo は Save で会員のロックを取らない。第7章の再試行案である。
// before は UPDATE の直前に呼ぶ。重なる予約と SQL の開始をそろえる。
type noLockRepo struct {
	*postgres.BookingRepository
	before func()
}

func (r *noLockRepo) Save(
	ctx context.Context, b *domain.Booking,
) error {
	r.before()
	s := b.State()
	tx, ok := pgtx.From(ctx)
	if !ok {
		return errors.New("no transaction")
	}
	_, err := sqlcgen.New(tx).UpdateBooking(ctx,
		sqlcgen.UpdateBookingParams{
			ID:          s.ID.String(),
			Version:     int64(s.Version),
			Status:      string(s.Status),
			DiscountYen: s.DiscountYen,
		})
	return err
}

// addNoLock は重なる予約をロックなしで保存する。
func addNoLock(
	ctx context.Context, tx *pgtx.Transactor, b *domain.Booking,
	barrier func(),
) error {
	s := b.State()
	return tx.Do(ctx, func(ctx context.Context) error {
		cur, _ := pgtx.From(ctx)
		barrier()
		return sqlcgen.New(cur).InsertBooking(ctx,
			sqlcgen.InsertBookingParams{
				ID:          s.ID.String(),
				MemberID:    s.MemberID.String(),
				LessonID:    s.LessonID.String(),
				StartsAt:    s.Start,
				EndsAt:      s.End,
				FeeYen:      s.FeeYen,
				DiscountYen: s.DiscountYen,
				Status:      string(s.Status),
				Version:     int64(s.Version),
			})
	})
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

// #region race

// ロックなしの確定と、重なる予約を同時に動かす。配信処理の Run は
// 本物の周期(200ms)で再配信する。デッドロックの回数は実行ごとに
// 変わるが、確定はすべて confirmed に進むはずである。
func TestConfirmWithoutLockRecovers(t *testing.T) {
	f := newFixture(t)
	const rounds = 300
	failed, slowest := 0, time.Duration(0)
	adds := map[string]int{}
	var logs bytes.Buffer
	repo := &noLockRepo{BookingRepository: f.repo}
	relay := f.relay(repo, &logs)
	for range rounds {
		attempts, took, addErr := f.round(t, relay, repo)
		adds[code(addErr)]++
		if attempts > 0 {
			failed++
			slowest = max(slowest, took)
		}
	}
	warns := strings.Count(logs.String(), `"level":"WARN"`)
	if warns != failed {
		t.Errorf("WARN = %d 行, 失敗 = %d 回", warns, failed)
	}
	t.Logf("確定 %d 回: すべて confirmed", rounds)
	t.Logf("40P01 で配信に失敗: %d 回, WARN のログ: %d 行",
		failed, warns)
	t.Logf("失敗した回の確定までの最長: 約 %.1f 秒", slowest.Seconds())
	t.Logf("重なる予約: %v", adds)
}

// #endregion

// round は1回分の確定と重なる予約を同時に動かす。
func (f fixture) round(
	t *testing.T, relay *outbox.Relay, repo *noLockRepo,
) (attempts int, took time.Duration, addErr error) {
	t.Helper()
	member := ids.NewID[ids.MemberMarker]()
	first := newBooking(t, member, false)
	f.addWithSeatReserved(t, first)
	second := newBooking(t, member, false)

	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	// 両方の Tx が SQL の直前まで進んでから、同時に進める
	var barrier sync.WaitGroup
	barrier.Add(2)
	wait := func() { barrier.Done(); barrier.Wait() }
	repo.before = sync.OnceFunc(wait)
	wg.Go(func() {
		addErr = addNoLock(t.Context(), f.tx, second, wait)
	})
	begin := time.Now()
	wg.Go(func() { _ = relay.Run(ctx) })
	deadline := time.Now().Add(30 * time.Second)
	for status(t, f.pool, first.ID()) != "confirmed" {
		if time.Now().After(deadline) {
			t.Fatal("確定に進まない")
		}
		time.Sleep(2 * time.Millisecond)
	}
	took = time.Since(begin)
	cancel()
	wg.Wait()
	err := f.pool.QueryRow(t.Context(),
		`SELECT attempts FROM outbox
		 WHERE aggregate_id = $1`, first.LessonID().String()).
		Scan(&attempts)
	if err != nil {
		t.Fatalf("attempts: %v", err)
	}
	return attempts, took, addErr
}

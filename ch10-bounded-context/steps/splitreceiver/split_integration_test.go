//go:build integration

// Package splitreceiver は受け手の処理を Tx で分けると、途中で止まった
// ときの再配信で結果が変わることを確かめる。
package splitreceiver_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	bookingapp "example.com/trainingbooking/internal/booking/application"
	bookingdomain "example.com/trainingbooking/internal/booking/domain"
	bookingpg "example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/catalog/application"
	catalogcontract "example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/catalog/domain"
	catalogpg "example.com/trainingbooking/internal/catalog/infrastructure/postgres"
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

var errStopped = errors.New("ここでプロセスが止まる")

type fixture struct {
	pool      *pgxpool.Pool
	tx        *pgtx.Transactor
	lessons   *catalogpg.LessonRepository
	writer    *catalogpg.OutboxWriter
	store     *outbox.Store
	processed *outbox.ProcessedEvents
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.New(t, bookingpg.Migrations,
		catalogpg.Migrations, outbox.Migrations)
	store := outbox.NewStore(pool)
	return fixture{
		pool:      pool,
		tx:        pgtx.NewTransactor(pool),
		lessons:   catalogpg.NewLessonRepository(pool),
		writer:    catalogpg.NewOutboxWriter(store),
		store:     store,
		processed: outbox.NewProcessedEvents(pool),
	}
}

func (f fixture) addLesson(
	t *testing.T, capacity int, seats ...ids.BookingID,
) ids.LessonID {
	t.Helper()
	l, err := domain.NewLesson(domain.LessonSpec{
		ID:       ids.NewID[ids.LessonMarker](),
		Title:    "少人数ピラティス",
		Start:    clock.now.AddDate(0, 1, 0),
		Duration: time.Hour,
		FeeYen:   3000,
		Capacity: capacity,
	})
	if err != nil {
		t.Fatalf("NewLesson: %v", err)
	}
	for _, b := range seats {
		if err := l.ReserveSeat(b, clock); err != nil {
			t.Fatalf("ReserveSeat: %v", err)
		}
		l.PullEvents()
	}
	err = f.tx.Do(t.Context(), func(ctx context.Context) error {
		return f.lessons.Add(ctx, l)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return l.ID()
}

func (f fixture) seats(t *testing.T, id ids.LessonID) int {
	t.Helper()
	l, err := f.lessons.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	return l.ReservedSeats()
}

func (f fixture) events(t *testing.T, eventType string) int {
	t.Helper()
	var n int
	err := f.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM outbox WHERE event_type = $1`,
		eventType).Scan(&n)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	return n
}

// reserveSeat は席の確保と結果のイベントの1つの Tx である。
func (f fixture) reserveSeat(
	ctx context.Context, lesson ids.LessonID, booking ids.BookingID,
) error {
	return f.tx.Do(ctx, func(ctx context.Context) error {
		l, err := f.lessons.FindByID(ctx, lesson)
		if err != nil {
			return err
		}
		err = l.ReserveSeat(booking, clock)
		if err != nil && !errors.Is(err, domain.ErrLessonFull) {
			return err
		}
		if err == nil {
			if err := f.lessons.Save(ctx, l); err != nil {
				return err
			}
		}
		return f.writer.Append(ctx, l.PullEvents()...)
	})
}

// seen は処理済みの記録があるかを読む。
func (f fixture) seen(ctx context.Context, eventID string) bool {
	var ok bool
	err := f.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM processed_events
		WHERE subscriber = 'catalog' AND event_id = $1)`,
		eventID).Scan(&ok)
	return err == nil && ok
}

func (f fixture) mark(ctx context.Context, eventID string) error {
	return f.tx.Do(ctx, func(ctx context.Context) error {
		_, err := f.processed.MarkProcessed(ctx, "catalog", eventID)
		return err
	})
}

// #region process-then-mark

// processThenMark は確かめる→処理→記録を別々の Tx で行う。stop が
// true なら、処理の後、記録の前に止まる。
func (f fixture) processThenMark(
	ctx context.Context, eventID string,
	lesson ids.LessonID, booking ids.BookingID, stop bool,
) error {
	if f.seen(ctx, eventID) {
		return nil
	}
	if err := f.reserveSeat(ctx, lesson, booking); err != nil {
		return err
	}
	if stop {
		return errStopped
	}
	return f.mark(ctx, eventID)
}

// #endregion

// 処理の後に止まると、満席の拒否が2回記録される。受け手の Booking は
// 2回目を拒否できないので、そのレッスンの配信が止まる。
func TestProcessThenMark_RejectionTwice(t *testing.T) {
	f := newFixture(t)
	lesson := f.addLesson(t, 1, ids.NewID[ids.BookingMarker]())
	bookings := bookingpg.NewBookingRepository(f.pool)
	target := addPending(t, f, bookings, lesson)
	eventID := outbox.NewEventID()
	ctx := t.Context()

	err := f.processThenMark(ctx, eventID, lesson, target, true)
	if !errors.Is(err, errStopped) {
		t.Fatalf("1回目: %v", err)
	}
	err = f.processThenMark(ctx, eventID, lesson, target, false)
	if err != nil {
		t.Fatalf("再配信: %v", err)
	}
	n := f.events(t, catalogcontract.EventSeatRejected)
	if n != 2 {
		t.Fatalf("SeatRejected = %d 件", n)
	}
	t.Logf("SeatRejected: %d 件", n)

	results := bookingapp.NewSeatResultHandler(
		bookings, f.processed, f.tx)
	relay := outbox.NewRelay(f.store, slog.New(slog.DiscardHandler),
		outbox.RelayOptions{})
	relay.Subscribe(catalogcontract.EventSeatRejected, "booking",
		outbox.On(results.OnSeatRejected))
	for range 3 {
		if _, err := relay.RunOnce(ctx); err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
	}
	head, err := f.store.Deliverable(ctx, 10)
	if err != nil || len(head) != 1 {
		t.Fatalf("head = %+v, err = %v", head, err)
	}
	var lastError string
	err = f.pool.QueryRow(ctx,
		`SELECT last_error FROM outbox WHERE id = $1`,
		head[0].ID).Scan(&lastError)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	t.Logf("未配信の先頭: 連番 %d, 失敗 %d 回", head[0].Sequence,
		head[0].Attempts)
	t.Logf("原因: %s", lastError)
}

func addPending(
	t *testing.T, f fixture, repo *bookingpg.BookingRepository,
	lesson ids.LessonID,
) ids.BookingID {
	t.Helper()
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := bookingdomain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	fee, err := bookingdomain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b, err := bookingdomain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](), lesson, slot, fee, clock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	err = f.tx.Do(t.Context(), func(ctx context.Context) error {
		return repo.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return b.ID()
}

// 記録を先にコミットすると、処理の前に止まったイベントは二度と処理
// されない。席は確保されず、予約は pending のまま残る。
func TestMarkThenProcess_EventIsLost(t *testing.T) {
	f := newFixture(t)
	lesson := f.addLesson(t, 2)
	booking := ids.NewID[ids.BookingMarker]()
	eventID := outbox.NewEventID()
	ctx := t.Context()

	if err := f.mark(ctx, eventID); err != nil {
		t.Fatalf("mark: %v", err)
	}
	// ここで止まる。再配信では記録があるので処理しない
	if !f.seen(ctx, eventID) {
		err := f.reserveSeat(ctx, lesson, booking)
		if err != nil {
			t.Fatalf("reserveSeat: %v", err)
		}
	}
	n := f.events(t, catalogcontract.EventSeatReserved)
	if f.seats(t, lesson) != 0 || n != 0 {
		t.Fatalf("seats = %d, events = %d", f.seats(t, lesson), n)
	}
	t.Logf("席: %d, SeatReserved: %d 件", f.seats(t, lesson), n)
}

// #region counted

// countedLesson は席を数だけで持つレッスンである。
type countedLesson struct{ reserved int }

func (l *countedLesson) reserveSeat() { l.reserved++ }

// 席を数で持つと、処理の後に止まった再配信で席が二重に確保される。
func TestCountedSeats_ReservedTwice(t *testing.T) {
	var l countedLesson
	processed := map[string]bool{}
	deliver := func(eventID string, stop bool) {
		if processed[eventID] {
			return
		}
		l.reserveSeat() // ここまでがコミットされる
		if stop {
			return
		}
		processed[eventID] = true
	}
	deliver("event-1", true)  // 処理の後、記録の前に止まる
	deliver("event-1", false) // 再配信
	if l.reserved != 2 {
		t.Fatalf("reserved = %d", l.reserved)
	}
	t.Logf("予約1件に対して、席の数: %d", l.reserved)
}

// #endregion

// failingOutbox は最初の Append を失敗させる。処理の途中で止まる
// 状況を作る。
type failingOutbox struct {
	application.OutboxWriter
	failed bool
}

func (o *failingOutbox) Append(
	ctx context.Context, events ...domain.DomainEvent,
) error {
	if !o.failed {
		o.failed = true
		return errStopped
	}
	return o.OutboxWriter.Append(ctx, events...)
}

// 本書の ReserveSeat は、記録と席とイベントを同じ Tx で書く。処理の
// 途中で止まっても、コミットの後で止まっても、結果は1回分である。
func TestSameTransaction(t *testing.T) {
	f := newFixture(t)
	lesson := f.addLesson(t, 2)
	reserve := application.NewReserveSeat(f.lessons,
		&failingOutbox{OutboxWriter: f.writer},
		f.processed, f.tx, clock)
	in := application.ReserveSeatInput{
		EventID:   outbox.NewEventID(),
		LessonID:  lesson,
		BookingID: ids.NewID[ids.BookingMarker](),
	}
	ctx := t.Context()

	err := reserve.Execute(ctx, in)
	if !errors.Is(err, errStopped) {
		t.Fatalf("途中で止まる: %v", err)
	}
	n := f.events(t, catalogcontract.EventSeatReserved)
	if f.seats(t, lesson) != 0 || n != 0 {
		t.Fatalf("seats = %d, events = %d", f.seats(t, lesson), n)
	}
	t.Logf("途中で止まった後: 席 %d, SeatReserved %d 件",
		f.seats(t, lesson), n)

	if err := reserve.Execute(ctx, in); err != nil {
		t.Fatalf("再配信: %v", err)
	}
	// コミットの後、配信済みの記録の前に止まった場合の再配信である
	if err := reserve.Execute(ctx, in); err != nil {
		t.Fatalf("2度目の再配信: %v", err)
	}
	n = f.events(t, catalogcontract.EventSeatReserved)
	l, err := f.lessons.FindByID(ctx, lesson)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if l.ReservedSeats() != 1 || n != 1 || l.Version() != 1 {
		t.Fatalf("seats = %d, events = %d, version = %d",
			l.ReservedSeats(), n, l.Version())
	}
	t.Logf("再配信を2回した後: 席 %d, SeatReserved %d 件, 版 %d",
		l.ReservedSeats(), n, l.Version())
}

// 同じイベントを2つの処理が同時に記録すると、後の処理は先の処理の
// コミットを待ち、処理済みと判定する。順序を作って毎回同じ経路を通す。
func TestConcurrentMarkWaitsForCommit(t *testing.T) {
	f := newFixture(t)
	eventID := outbox.NewEventID()
	ctx := t.Context()
	var second bool
	done := make(chan error, 1)
	err := f.tx.Do(ctx, func(txCtx context.Context) error {
		first, err := f.processed.MarkProcessed(
			txCtx, "catalog", eventID)
		if err != nil || !first {
			t.Fatalf("1つ目: first = %v, err = %v", first, err)
		}
		go func() {
			done <- f.tx.Do(ctx, func(ctx context.Context) error {
				var err error
				second, err = f.processed.MarkProcessed(
					ctx, "catalog", eventID)
				return err
			})
		}()
		f.waitForInsertLock(t)
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("2つ目: %v", err)
	}
	if second {
		t.Error("2つ目が初めてのイベントと判定された")
	}
}

// waitForInsertLock は processed_events の INSERT がロックを待つまで
// 待つ。
func (f fixture) waitForInsertLock(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		err := f.pool.QueryRow(t.Context(),
			`SELECT count(*) FROM pg_stat_activity
			 WHERE datname = current_database()
			   AND wait_event_type = 'Lock'
			   AND query LIKE '%processed_events%'`).Scan(&n)
		if err != nil {
			t.Fatalf("pg_stat_activity: %v", err)
		}
		if n > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("2つ目の INSERT が待ちに入らない")
}

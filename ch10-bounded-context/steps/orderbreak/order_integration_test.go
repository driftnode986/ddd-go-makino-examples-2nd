//go:build integration

// Package orderbreak は配信の順序と件数の枠が崩れる場面を確かめる。
package orderbreak_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC),
}

const eventType = "test.event"

func appendEvents(
	t *testing.T, pool *pgxpool.Pool, messages ...outbox.Message,
) {
	t.Helper()
	store := outbox.NewStore(pool)
	err := pgtx.NewTransactor(pool).Do(t.Context(),
		func(ctx context.Context) error {
			return store.Append(ctx, messages...)
		})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
}

func message(aggregate, name string) outbox.Message {
	return outbox.Message{
		EventID:       outbox.NewEventID(),
		AggregateType: "test",
		AggregateID:   aggregate,
		EventType:     eventType,
		Payload:       fmt.Appendf(nil, `{"name":%q}`, name),
		OccurredAt:    clock.now,
	}
}

// #region skip-locked

const claim = `SELECT event_type, sequence FROM outbox
	WHERE published_at IS NULL ORDER BY id LIMIT 1
	FOR UPDATE SKIP LOCKED`

// 2本の配信処理が SKIP LOCKED で1行ずつ取ると、同じ集約の連番2を
// 先に取れる。連番1の Tx が終わる前に配信すれば順序が逆になる。
func TestSkipLockedOvertakes(t *testing.T) {
	pool := dbtest.New(t, outbox.Migrations)
	booking := uuid.New().String()
	placed, cancelled := message(booking, "placed"),
		message(booking, "cancelled")
	placed.EventType = "booking.placed"
	cancelled.EventType = "booking.cancelled"
	appendEvents(t, pool, placed, cancelled)

	ctx := t.Context()
	first, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer func() { _ = first.Rollback(context.Background()) }()
	second, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer func() { _ = second.Rollback(context.Background()) }()
	name, seq := take(t, first)
	t.Logf("配信処理 A が取った行: %s (連番 %d)", name, seq)
	name, seq = take(t, second)
	t.Logf("配信処理 B が取った行: %s (連番 %d)", name, seq)
	if seq != 2 {
		t.Fatalf("B の連番 = %d", seq)
	}
}

// #endregion

func take(t *testing.T, tx pgx.Tx) (string, int64) {
	t.Helper()
	var name string
	var seq int64
	err := tx.QueryRow(t.Context(), claim).Scan(&name, &seq)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	return name, seq
}

// #region release-first

// キャンセルが確保を追い越すと、戻す席がまだ無いので何も起きず、
// その後の確保で席が埋まる。キャンセルした予約が席を持ち続ける。
func TestCancelBeforePlaced(t *testing.T) {
	l, err := domain.NewLesson(domain.LessonSpec{
		ID:       ids.NewID[ids.LessonMarker](),
		Title:    "少人数ピラティス",
		Start:    clock.now.AddDate(0, 1, 0),
		Duration: time.Hour,
		FeeYen:   5000,
		Capacity: 1,
	})
	if err != nil {
		t.Fatalf("NewLesson: %v", err)
	}
	booking := ids.NewID[ids.BookingMarker]()
	if err := l.ReleaseSeat(booking); err != nil {
		t.Fatalf("ReleaseSeat: %v", err)
	}
	if err := l.ReserveSeat(booking, clock); err != nil {
		t.Fatalf("ReserveSeat: %v", err)
	}
	if l.ReservedSeats() != 1 {
		t.Fatalf("席 = %d", l.ReservedSeats())
	}
	t.Logf("キャンセル済みの予約が持つ席: %d", l.ReservedSeats())
}

// #endregion

// idOrderStore は ListDeliverable の ORDER BY を h.id だけにした
// 読み方である。失敗の回数を見ない。
type idOrderStore struct {
	*outbox.Store
	pool *pgxpool.Pool
}

func (s idOrderStore) Deliverable(
	ctx context.Context, limit int,
) ([]outbox.Delivery, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT h.id, h.event_id, h.aggregate_id, h.payload FROM (
			SELECT DISTINCT ON (aggregate_type, aggregate_id)
				id, event_id, aggregate_type, aggregate_id,
				sequence, payload
			FROM outbox WHERE published_at IS NULL
			ORDER BY aggregate_type, aggregate_id, sequence
		) h ORDER BY h.id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(
		r pgx.CollectableRow,
	) (outbox.Delivery, error) {
		var d outbox.Delivery
		err := r.Scan(&d.ID, &d.EventID, &d.AggregateID, &d.Payload)
		d.EventType = eventType
		return d, err
	})
}

// #region batch

// 配信の枠(5件)と同じ数の集約が失敗し続ける。3周で正常な集約の
// イベントが何件配られるかを、並べ方で比べる。
func TestFailuresFillTheBatch(t *testing.T) {
	for _, c := range []struct {
		name  string
		store func(*pgxpool.Pool) outbox.RelayStore
		want  int
	}{
		{"ORDER BY h.id", func(p *pgxpool.Pool) outbox.RelayStore {
			return idOrderStore{outbox.NewStore(p), p}
		}, 0},
		{"ORDER BY h.attempts, h.id",
			func(p *pgxpool.Pool) outbox.RelayStore {
				return outbox.NewStore(p)
			}, 1},
	} {
		got := delivered(t, c.store)
		t.Logf("%s: 正常な集約への配信 %d 件", c.name, got)
		if got != c.want {
			t.Errorf("%s: %d 件, want %d", c.name, got, c.want)
		}
	}
}

// #endregion

func delivered(
	t *testing.T, store func(*pgxpool.Pool) outbox.RelayStore,
) int {
	t.Helper()
	pool := dbtest.New(t, outbox.Migrations)
	const batch = 5
	for range batch {
		appendEvents(t, pool, message(uuid.New().String(), "stuck"))
	}
	appendEvents(t, pool, message(uuid.New().String(), "ok"))
	relay := outbox.NewRelay(store(pool),
		slog.New(slog.DiscardHandler),
		outbox.RelayOptions{BatchSize: batch})
	n := 0
	relay.Subscribe(eventType, "test", outbox.On(func(
		_ context.Context, _ string, e struct {
			Name string `json:"name"`
		},
	) error {
		if e.Name == "stuck" {
			return errors.New("boom")
		}
		n++
		return nil
	}))
	for range 3 {
		if _, err := relay.RunOnce(t.Context()); err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
	}
	return n
}

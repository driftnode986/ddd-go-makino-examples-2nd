// Package outbox は集約と同じ Tx でイベントを書き、別の処理で配信
// する仕組みである。配信は少なくとも1回で、順序は集約ごとに保つ。
package outbox

import (
	"context"
	"embed"
	"time"
	"uuid"

	"example.com/trainingbooking/internal/platform/outbox/sqlcgen"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations はスキーマの SQL である。
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Message は outbox に書くイベントである。
type Message struct {
	EventID       string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	OccurredAt    time.Time
}

// Delivery は配信するイベントである。
type Delivery struct {
	Message
	ID        int64
	Sequence  int64
	Attempts  int
	CreatedAt time.Time
}

// NewEventID はイベントの ID を採番する。
func NewEventID() string { return uuid.NewV7().String() }

// Store は outbox テーブルの読み書きである。
type Store struct {
	pool *pgxpool.Pool
}

// NewStore は Store を返す。
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// #region append

// Append はイベントを ctx の Tx の中で書き、集約ごとの連番を振る。
// ctx に Tx が無ければ txn.ErrNoTransaction を返す。
func (s *Store) Append(
	ctx context.Context, messages ...Message,
) error {
	tx, ok := pgtx.From(ctx)
	if !ok {
		return txn.ErrNoTransaction
	}
	q := sqlcgen.New(tx)
	for _, m := range messages {
		key := m.AggregateType + ":" + m.AggregateID
		if err := q.LockAggregate(ctx, key); err != nil {
			return err
		}
		err := q.InsertEvent(ctx, sqlcgen.InsertEventParams{
			EventID:       m.EventID,
			AggregateType: m.AggregateType,
			AggregateID:   m.AggregateID,
			EventType:     m.EventType,
			Payload:       m.Payload,
			OccurredAt:    m.OccurredAt,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// #endregion

// Deliverable は集約ごとに未配信の先頭を、失敗の少ない順に limit
// 件返す。失敗の回数が同じなら古い順である。
func (s *Store) Deliverable(
	ctx context.Context, limit int,
) ([]Delivery, error) {
	rows, err := sqlcgen.New(s.pool).ListDeliverable(
		ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]Delivery, len(rows))
	for i, r := range rows {
		out[i] = Delivery{
			Message: Message{
				EventID:       r.EventID,
				AggregateType: r.AggregateType,
				AggregateID:   r.AggregateID,
				EventType:     r.EventType,
				Payload:       r.Payload,
				OccurredAt:    r.OccurredAt,
			},
			ID:        r.ID,
			Sequence:  r.Sequence,
			Attempts:  int(r.Attempts),
			CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

// MarkPublished は配信済みの記録を残す。
func (s *Store) MarkPublished(
	ctx context.Context, id int64,
) error {
	return sqlcgen.New(s.pool).MarkPublished(ctx, id)
}

// RecordFailure は失敗の回数と原因を残す。
func (s *Store) RecordFailure(
	ctx context.Context, id int64, cause string,
) error {
	return sqlcgen.New(s.pool).RecordFailure(ctx,
		sqlcgen.RecordFailureParams{ID: id, LastError: cause})
}

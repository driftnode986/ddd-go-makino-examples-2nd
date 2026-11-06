package outbox

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const (
	defaultInterval   = 200 * time.Millisecond
	defaultBatchSize  = 100
	defaultStallAfter = 5 * time.Minute
)

// Handler は配信されたイベントを処理する。処理済みの判定は受け手が
// 行う。同じイベントが2回以上届くことがある。
type Handler func(ctx context.Context, d Delivery) error

// On は本文を T に読み替える Handler を返す。知らない項目は無視する。
// イベントに項目を足しても、古い受け手が動き続けるためである。
func On[T any](
	fn func(ctx context.Context, eventID string, event T) error,
) Handler {
	return func(ctx context.Context, d Delivery) error {
		var event T
		if err := json.Unmarshal(d.Payload, &event); err != nil {
			return fmt.Errorf("decode %s: %w", d.EventType, err)
		}
		return fn(ctx, d.EventID, event)
	}
}

// RelayStore は配信処理が使う outbox の操作である。
type RelayStore interface {
	Deliverable(ctx context.Context, limit int) ([]Delivery, error)
	MarkPublished(ctx context.Context, id int64) error
	RecordFailure(ctx context.Context, id int64, cause string) error
}

// RelayOptions は配信処理の設定である。ゼロ値は既定値を使う。
type RelayOptions struct {
	// Interval は配るものが無いときの待ち時間である。
	Interval time.Duration
	// BatchSize は1回に配る上限である。
	BatchSize int
	// StallAfter は配信の遅れを障害として記録するまでの時間である。
	// 超えても再配信は止めない。
	StallAfter time.Duration
}

type subscription struct {
	name    string
	handler Handler
}

// Relay は outbox のイベントを購読者へ配信する。1本だけ動かす。
type Relay struct {
	store       RelayStore
	logger      *slog.Logger
	options     RelayOptions
	subscribers map[string][]subscription
}

// NewRelay は Relay を返す。Run の前に Subscribe する。
func NewRelay(
	store RelayStore, logger *slog.Logger, opts RelayOptions,
) *Relay {
	opts.Interval = cmp.Or(opts.Interval, defaultInterval)
	opts.BatchSize = cmp.Or(opts.BatchSize, defaultBatchSize)
	opts.StallAfter = cmp.Or(opts.StallAfter, defaultStallAfter)
	return &Relay{
		store:       store,
		logger:      logger,
		options:     opts,
		subscribers: make(map[string][]subscription),
	}
}

// Subscribe は eventType のイベントの購読者を登録する。
func (r *Relay) Subscribe(
	eventType, name string, h Handler,
) {
	r.subscribers[eventType] = append(r.subscribers[eventType],
		subscription{name: name, handler: h})
}

// Run は ctx が終わるまで配信を続け、ctx.Err() を返す。DB の障害で
// は止まらず、記録して次の周期にやり直す。
func (r *Relay) Run(ctx context.Context) error {
	for {
		n, err := r.RunOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			r.logger.ErrorContext(ctx, "relay failed", "error", err)
		}
		if n > 0 {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.options.Interval):
		}
	}
}

// RunOnce は集約ごとに未配信の先頭を1回ずつ配り、配信済みにした
// 件数を返す。受け手の失敗はエラーにせず、記録して次の周期に
// 再配信する。後の連番はそれまで候補に出ない。
func (r *Relay) RunOnce(ctx context.Context) (int, error) {
	deliveries, err := r.store.Deliverable(
		ctx, r.options.BatchSize)
	if err != nil {
		return 0, err
	}
	published := 0
	var errs []error
	for _, d := range deliveries {
		if err := r.deliver(ctx, d); err != nil {
			if ctx.Err() != nil {
				return published, ctx.Err()
			}
			r.recordFailure(ctx, d, err)
			continue
		}
		// ここで止まると再配信になる。受け手の処理済みの記録で
		// 二重の処理を防ぐ
		err := r.store.MarkPublished(ctx, d.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"mark published %s: %w", d.EventID, err))
			continue
		}
		published++
	}
	return published, errors.Join(errs...)
}

func (r *Relay) deliver(ctx context.Context, d Delivery) error {
	for _, s := range r.subscribers[d.EventType] {
		if err := s.handler(ctx, d); err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
	}
	return nil
}

// recordFailure は失敗を残す。許容時間を超えた遅れはエラーの水準に
// 上げる。予約は pending のまま見え、再配信は続く。
func (r *Relay) recordFailure(
	ctx context.Context, d Delivery, cause error,
) {
	level := slog.LevelWarn
	if time.Since(d.CreatedAt) > r.options.StallAfter {
		level = slog.LevelError
	}
	r.logger.Log(ctx, level, "delivery failed",
		"event_id", d.EventID,
		"event_type", d.EventType,
		"aggregate_id", d.AggregateID,
		"attempts", d.Attempts+1,
		"error", cause,
	)
	err := r.store.RecordFailure(ctx, d.ID, cause.Error())
	if err != nil {
		r.logger.ErrorContext(ctx, "record failure failed",
			"event_id", d.EventID, "error", err)
	}
}

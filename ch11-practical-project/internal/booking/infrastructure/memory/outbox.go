package memory

import (
	"context"
	"slices"
	"sync"

	"example.com/trainingbooking/internal/booking/domain"
)

// OutboxWriter は OutboxWriter のテストダブルである。
// 書いたイベントをメモリに持ち、Tx が失敗すれば戻す。
type OutboxWriter struct {
	mu     sync.Mutex
	events []domain.DomainEvent
}

// NewOutboxWriter は空の OutboxWriter を返す。
func NewOutboxWriter() *OutboxWriter { return &OutboxWriter{} }

// Append はイベントを ctx の Tx の中で書く。
func (o *OutboxWriter) Append(
	ctx context.Context, events ...domain.DomainEvent,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cur, ok := txFrom(ctx)
	if !ok {
		return domain.ErrNoTransaction
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	mark := len(o.events)
	o.events = append(o.events, events...)
	cur.undo = append(cur.undo, func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		o.events = o.events[:mark]
	})
	return nil
}

// Events は書かれたイベントのコピーを返す。
func (o *OutboxWriter) Events() []domain.DomainEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.events)
}

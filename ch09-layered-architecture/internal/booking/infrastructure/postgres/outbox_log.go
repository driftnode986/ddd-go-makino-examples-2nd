package postgres

import (
	"context"
	"log/slog"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

// LoggingOutbox はイベントをログに書くだけの OutboxWriter である。
// 配信処理が入るまでのもので、配信はしない。
type LoggingOutbox struct {
	logger *slog.Logger
}

// NewLoggingOutbox は LoggingOutbox を返す。
func NewLoggingOutbox(logger *slog.Logger) *LoggingOutbox {
	return &LoggingOutbox{logger: logger}
}

// Append はイベントをログに書く。ctx に Tx が無ければ
// txn.ErrNoTransaction を返す。
func (o *LoggingOutbox) Append(
	ctx context.Context, events ...domain.DomainEvent,
) error {
	if _, ok := pgtx.From(ctx); !ok {
		return txn.ErrNoTransaction
	}
	for _, e := range events {
		o.logger.InfoContext(ctx, "event recorded",
			"event", e.EventName(),
			"aggregate_id", e.AggregateID(),
		)
	}
	return nil
}

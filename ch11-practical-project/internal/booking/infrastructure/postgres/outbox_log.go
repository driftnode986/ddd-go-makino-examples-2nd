package postgres

import (
	"context"
	"log/slog"

	"example.com/trainingbooking/internal/booking/domain"
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
// domain.ErrNoTransaction を返す。
func (o *LoggingOutbox) Append(
	ctx context.Context, events ...domain.DomainEvent,
) error {
	if _, ok := txFrom(ctx); !ok {
		return domain.ErrNoTransaction
	}
	for _, e := range events {
		o.logger.InfoContext(ctx, "event recorded",
			"event", e.EventName(),
			"aggregate_id", e.AggregateID(),
		)
	}
	return nil
}

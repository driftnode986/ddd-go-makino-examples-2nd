package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"example.com/trainingbooking/internal/booking/contract"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/platform/outbox"
)

const aggregateType = "booking"

// OutboxWriter はドメインイベントを統合イベントに変えて outbox に
// 書く。
type OutboxWriter struct {
	store *outbox.Store
}

// NewOutboxWriter は OutboxWriter を返す。
func NewOutboxWriter(store *outbox.Store) *OutboxWriter {
	return &OutboxWriter{store: store}
}

// Append はイベントを ctx の Tx の中で書く。
func (w *OutboxWriter) Append(
	ctx context.Context, events ...domain.DomainEvent,
) error {
	messages := make([]outbox.Message, len(events))
	for i, e := range events {
		m, err := toMessage(e)
		if err != nil {
			return err
		}
		messages[i] = m
	}
	return w.store.Append(ctx, messages...)
}

func toMessage(e domain.DomainEvent) (outbox.Message, error) {
	var eventType string
	var payload any
	switch e := e.(type) {
	case domain.BookingPlaced:
		eventType = contract.EventBookingPlaced
		payload = contract.BookingPlaced{
			BookingID:  e.BookingID.String(),
			MemberID:   e.MemberID.String(),
			LessonID:   e.LessonID.String(),
			OccurredAt: e.OccurredAt,
		}
	case domain.BookingCancelled:
		eventType = contract.EventBookingCancelled
		payload = contract.BookingCancelled{
			BookingID:  e.BookingID.String(),
			LessonID:   e.LessonID.String(),
			OccurredAt: e.OccurredAt,
		}
	default:
		return outbox.Message{}, fmt.Errorf(
			"unknown event: %s", e.EventName())
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return outbox.Message{}, err
	}
	return outbox.Message{
		EventID:       outbox.NewEventID(),
		AggregateType: aggregateType,
		AggregateID:   e.AggregateID(),
		EventType:     eventType,
		Payload:       body,
		OccurredAt:    e.When(),
	}, nil
}

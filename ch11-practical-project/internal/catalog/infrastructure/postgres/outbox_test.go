package postgres

import (
	"encoding/json"
	"testing"
	"time"

	"example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
)

func TestToMessage_ConvertsToContract(t *testing.T) {
	lesson := ids.NewID[ids.LessonMarker]()
	booking := ids.NewID[ids.BookingMarker]()
	at := time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC)

	m, err := toMessage(domain.SeatRejected{
		LessonID:   lesson,
		BookingID:  booking,
		Reason:     domain.RejectLessonNotFound,
		OccurredAt: at,
	})
	if err != nil {
		t.Fatalf("toMessage: %v", err)
	}
	if m.EventType != contract.EventSeatRejected ||
		m.AggregateType != "lesson" ||
		m.AggregateID != lesson.String() || m.EventID == "" ||
		!m.OccurredAt.Equal(at) {
		t.Errorf("message = %+v", m)
	}
	var got contract.SeatRejected
	if err := json.Unmarshal(m.Payload, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.BookingID != booking.String() ||
		got.LessonID != lesson.String() ||
		got.Reason != contract.ReasonLessonNotFound {
		t.Errorf("payload = %+v", got)
	}

	m, err = toMessage(domain.SeatReserved{
		LessonID: lesson, BookingID: booking, OccurredAt: at,
	})
	if err != nil {
		t.Fatalf("toMessage: %v", err)
	}
	if m.EventType != contract.EventSeatReserved {
		t.Errorf("message = %+v", m)
	}
}

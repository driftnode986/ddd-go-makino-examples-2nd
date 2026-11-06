package postgres

import (
	"encoding/json"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/contract"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

func TestToMessage_ConvertsToContract(t *testing.T) {
	booking := ids.NewID[ids.BookingMarker]()
	member := ids.NewID[ids.MemberMarker]()
	lesson := ids.NewID[ids.LessonMarker]()
	at := time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC)

	m, err := toMessage(domain.BookingPlaced{
		BookingID:  booking,
		MemberID:   member,
		LessonID:   lesson,
		OccurredAt: at,
	})
	if err != nil {
		t.Fatalf("toMessage: %v", err)
	}
	if m.EventType != contract.EventBookingPlaced ||
		m.AggregateType != "booking" ||
		m.AggregateID != booking.String() || m.EventID == "" ||
		!m.OccurredAt.Equal(at) {
		t.Errorf("message = %+v", m)
	}
	var got contract.BookingPlaced
	if err := json.Unmarshal(m.Payload, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.BookingID != booking.String() ||
		got.MemberID != member.String() ||
		got.LessonID != lesson.String() {
		t.Errorf("payload = %+v", got)
	}

	m, err = toMessage(domain.BookingCancelled{
		BookingID: booking, LessonID: lesson, OccurredAt: at,
	})
	if err != nil {
		t.Fatalf("toMessage: %v", err)
	}
	if m.EventType != contract.EventBookingCancelled {
		t.Errorf("message = %+v", m)
	}
}

package application_test

import (
	"errors"
	"testing"
	"uuid"

	bookingcontract "example.com/trainingbooking/internal/booking/contract"
	"example.com/trainingbooking/internal/catalog/application"
	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
)

func newBookingEventHandler(
	d memoryDeps,
) *application.BookingEventHandler {
	return application.NewBookingEventHandler(
		d.reserve(), d.release())
}

func placedEvent(lesson ids.LessonID) bookingcontract.BookingPlaced {
	return bookingcontract.BookingPlaced{
		BookingID: ids.NewID[ids.BookingMarker]().String(),
		LessonID:  lesson.String(),
	}
}

func TestBookingPlaced_ReservesSeat(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 1)
	err := newBookingEventHandler(d).OnBookingPlaced(t.Context(),
		uuid.New().String(), placedEvent(lesson))
	if err != nil {
		t.Fatalf("OnBookingPlaced: %v", err)
	}
	if n := seats(t, d.repo, lesson); n != 1 {
		t.Errorf("seats = %d", n)
	}
}

// 満席は配信の失敗ではない。拒否を記録して配信済みにする。
func TestBookingPlaced_FullIsDelivered(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 1)
	h := newBookingEventHandler(d)
	for range 2 {
		err := h.OnBookingPlaced(t.Context(),
			uuid.New().String(), placedEvent(lesson))
		if err != nil {
			t.Fatalf("OnBookingPlaced: %v", err)
		}
	}
	events := d.outbox.Events()
	if len(events) != 2 {
		t.Fatalf("events = %v", events)
	}
	if _, ok := events[1].(domain.SeatRejected); !ok {
		t.Errorf("event = %#v", events[1])
	}
}

// レッスンが無いのも配信の失敗ではない。止めずに拒否を伝える。
func TestBookingPlaced_MissingLessonIsDelivered(t *testing.T) {
	d := newDeps()
	err := newBookingEventHandler(d).OnBookingPlaced(t.Context(),
		uuid.New().String(),
		placedEvent(ids.NewID[ids.LessonMarker]()))
	if err != nil {
		t.Fatalf("OnBookingPlaced: %v", err)
	}
	if len(d.outbox.Events()) != 1 {
		t.Errorf("events = %v", d.outbox.Events())
	}
}

func TestBookingPlaced_RedeliveryIsNoOp(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 2)
	h := newBookingEventHandler(d)
	event := placedEvent(lesson)
	eventID := uuid.New().String()
	for range 2 {
		err := h.OnBookingPlaced(t.Context(), eventID, event)
		if err != nil {
			t.Fatalf("OnBookingPlaced: %v", err)
		}
	}
	if n := seats(t, d.repo, lesson); n != 1 {
		t.Errorf("seats = %d", n)
	}
	if len(d.outbox.Events()) != 1 {
		t.Errorf("結果のイベントが増えた: %v", d.outbox.Events())
	}
}

func TestBookingPlaced_BrokenIDIsError(t *testing.T) {
	d := newDeps()
	err := newBookingEventHandler(d).OnBookingPlaced(t.Context(),
		uuid.New().String(), bookingcontract.BookingPlaced{
			BookingID: "x",
			LessonID:  ids.NewID[ids.LessonMarker]().String(),
		})
	if !errors.Is(err, ids.ErrInvalidID) {
		t.Fatalf("err = %v", err)
	}
}

func TestBookingCancelled_ReleasesSeatOnce(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 2)
	h := newBookingEventHandler(d)
	kept := placedEvent(lesson)
	err := h.OnBookingPlaced(t.Context(), uuid.New().String(), kept)
	if err != nil {
		t.Fatalf("OnBookingPlaced: %v", err)
	}
	placed := placedEvent(lesson)
	err = h.OnBookingPlaced(t.Context(), uuid.New().String(), placed)
	if err != nil {
		t.Fatalf("OnBookingPlaced: %v", err)
	}
	cancelled := bookingcontract.BookingCancelled{
		BookingID: placed.BookingID,
		LessonID:  placed.LessonID,
	}
	eventID := uuid.New().String()
	for range 2 {
		err := h.OnBookingCancelled(t.Context(), eventID, cancelled)
		if err != nil {
			t.Fatalf("OnBookingCancelled: %v", err)
		}
		if n := seats(t, d.repo, lesson); n != 1 {
			t.Fatalf("seats = %d", n)
		}
	}
}

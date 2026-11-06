package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

func newCancel(d memoryDeps) *application.CancelBooking {
	return application.NewCancelBooking(
		d.repo, d.outbox, d.tx, fixedClock{now})
}

func cancelInput(id ids.BookingID) application.CancelBookingInput {
	return application.CancelBookingInput{BookingID: id.String()}
}

func TestCancelBooking_ConfirmedIsCancelled(t *testing.T) {
	d := newMemoryDeps()
	id := addConfirmed(t, d)
	out, err := newCancel(d).Execute(t.Context(), cancelInput(id))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.BookingID != id.String() || out.Status != "cancelled" {
		t.Errorf("out = %+v", out)
	}
	got, err := d.repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status() != domain.StatusCancelled {
		t.Errorf("保存された Status = %s", got.Status())
	}
	events := d.outbox.Events()
	if len(events) != 1 {
		t.Fatalf("events = %v", events)
	}
	if _, ok := events[0].(domain.BookingCancelled); !ok {
		t.Errorf("event = %#v", events[0])
	}
}

func TestCancelBooking_PendingIsKindConflict(t *testing.T) {
	d := newMemoryDeps()
	out, err := newPlace(t, d).Execute(t.Context(), placeInput())
	if err != nil {
		t.Fatalf("PlaceBooking: %v", err)
	}
	in := application.CancelBookingInput{BookingID: out.BookingID}
	_, err = newCancel(d).Execute(t.Context(), in)
	wantKind(t, err, application.KindConflict)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("err = %v", err)
	}
}

func TestCancelBooking_InvalidIDIsKindInvalid(t *testing.T) {
	in := application.CancelBookingInput{BookingID: "x"}
	_, err := newCancel(newMemoryDeps()).Execute(t.Context(), in)
	wantKind(t, err, application.KindInvalid)
}

func TestCancelBooking_MissingIsKindNotFound(t *testing.T) {
	in := cancelInput(ids.NewID[ids.BookingMarker]())
	_, err := newCancel(newMemoryDeps()).Execute(t.Context(), in)
	wantKind(t, err, application.KindNotFound)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestCancelBooking_TooLateIsKindConflict(t *testing.T) {
	d := newMemoryDeps()
	id := addConfirmed(t, d)
	// 開始は6月1日10時。その23時間前である
	late := time.Date(2030, time.May, 31, 11, 0, 0, 0, jst)
	cancel := application.NewCancelBooking(
		d.repo, d.outbox, d.tx, fixedClock{late})
	_, err := cancel.Execute(t.Context(), cancelInput(id))
	wantKind(t, err, application.KindConflict)
	if !errors.Is(err, domain.ErrCancelTooLate) {
		t.Errorf("err = %v", err)
	}
}

func TestCancelBooking_VersionConflictIsKindConflict(t *testing.T) {
	d := newMemoryDeps()
	id := addConfirmed(t, d)
	cancel := application.NewCancelBooking(
		failingRepository{d.repo, domain.ErrVersionConflict},
		d.outbox, d.tx, fixedClock{now})
	_, err := cancel.Execute(t.Context(), cancelInput(id))
	wantKind(t, err, application.KindConflict)
}

func TestCancelBooking_OutboxFailureRollsBack(t *testing.T) {
	d := newMemoryDeps()
	id := addConfirmed(t, d)
	boom := errors.New("outbox")
	cancel := application.NewCancelBooking(
		d.repo, failingOutbox{boom}, d.tx, fixedClock{now})
	_, err := cancel.Execute(t.Context(), cancelInput(id))
	wantKind(t, err, application.KindInternal)
	got, err := d.repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status() != domain.StatusConfirmed {
		t.Errorf("Status = %s", got.Status())
	}
}

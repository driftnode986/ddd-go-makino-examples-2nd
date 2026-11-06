package application_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

func newPlace(
	t *testing.T, d memoryDeps,
) *application.PlaceBooking {
	t.Helper()
	return application.NewPlaceBooking(
		d.repo, d.outbox, d.tx, fixedClock{now}, terms(t))
}

func TestPlaceBooking_SavesPendingAndRecordsEvent(t *testing.T) {
	d := newMemoryDeps()
	in := placeInput()
	out, err := newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != "pending" {
		t.Errorf("Status = %s", out.Status)
	}
	id, err := ids.ParseID[ids.BookingMarker](out.BookingID)
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	got, err := d.repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status() != domain.StatusPending {
		t.Errorf("保存された Status = %s", got.Status())
	}
	if got.Total().Amount() != 3000 {
		t.Errorf("Total = %d", got.Total().Amount())
	}
	if !got.Slot().Start().Equal(in.Start) {
		t.Errorf("Start = %v", got.Slot().Start())
	}
	events := d.outbox.Events()
	if len(events) != 1 {
		t.Fatalf("events = %v", events)
	}
	placed, ok := events[0].(domain.BookingPlaced)
	if !ok || !placed.BookingID.Equal(id) {
		t.Errorf("event = %#v", events[0])
	}
}

func TestPlaceBooking_InvalidMemberIDIsKindInvalid(t *testing.T) {
	in := placeInput()
	in.MemberID = "not-a-uuid"
	_, err := newPlace(t, newMemoryDeps()).Execute(t.Context(), in)
	got := wantKind(t, err, application.KindInvalid)
	if !errors.Is(err, ids.ErrInvalidID) {
		t.Errorf("原因をたどれない: %v", got)
	}
}

func TestPlaceBooking_InvalidInputIsKindInvalid(t *testing.T) {
	cases := []struct {
		name   string
		change func(*application.PlaceBookingInput)
		cause  error
	}{
		{"レッスンの ID", func(in *application.PlaceBookingInput) {
			in.LessonID = ""
		}, ids.ErrInvalidID},
		{"開始が無い", func(in *application.PlaceBookingInput) {
			in.Start = time.Time{}
		}, domain.ErrInvalidTimeSlot},
		{"長さが0", func(in *application.PlaceBookingInput) {
			in.Duration = 0
		}, domain.ErrInvalidDuration},
		{"正時でない", func(in *application.PlaceBookingInput) {
			in.Start = in.Start.Add(time.Minute)
		}, domain.ErrNotOnTheHour},
		{"営業時間の外", func(in *application.PlaceBookingInput) {
			in.Start = time.Date(2030, time.June, 1, 8, 0, 0, 0, jst)
		}, domain.ErrOutOfBusinessHours},
		{"過去の開始", func(in *application.PlaceBookingInput) {
			in.Start = time.Date(
				2030, time.April, 1, 10, 0, 0, 0, jst)
		}, domain.ErrStartInPast},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := placeInput()
			c.change(&in)
			d := newMemoryDeps()
			_, err := newPlace(t, d).Execute(t.Context(), in)
			wantKind(t, err, application.KindInvalid)
			if !errors.Is(err, c.cause) {
				t.Errorf("err = %v, want %v", err, c.cause)
			}
			if len(d.outbox.Events()) != 0 {
				t.Errorf("events = %v", d.outbox.Events())
			}
		})
	}
}

// 営業時間の判定に使うのは terms の Location である。
func TestPlaceBooking_UsesLocationFromTerms(t *testing.T) {
	in := placeInput()
	// UTC の10時は JST の19時で営業時間内、
	// UTC の13時は JST の22時で営業時間外になる。
	in.Start = time.Date(2030, time.June, 1, 10, 0, 0, 0, time.UTC)
	place := newPlace(t, newMemoryDeps())
	if _, err := place.Execute(t.Context(), in); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	in = placeInput()
	in.Start = time.Date(2030, time.June, 1, 13, 0, 0, 0, time.UTC)
	_, err := place.Execute(t.Context(), in)
	if !errors.Is(err, domain.ErrOutOfBusinessHours) {
		t.Fatalf("err = %v", err)
	}
}

func TestPlaceBooking_OverlapIsKindConflict(t *testing.T) {
	d := newMemoryDeps()
	place := newPlace(t, d)
	in := placeInput()
	if _, err := place.Execute(t.Context(), in); err != nil {
		t.Fatalf("1件目: %v", err)
	}
	in.LessonID = ids.NewID[ids.LessonMarker]().String()
	_, err := place.Execute(t.Context(), in)
	wantKind(t, err, application.KindConflict)
	if !errors.Is(err, domain.ErrOverlap) {
		t.Errorf("err = %v", err)
	}
	if len(d.outbox.Events()) != 1 {
		t.Errorf("events = %v", d.outbox.Events())
	}
}

func TestPlaceBooking_RepositoryFailureKeepsCause(t *testing.T) {
	boom := errors.New("connection refused: secret-host:5432")
	d := newMemoryDeps()
	place := application.NewPlaceBooking(
		failingRepository{d.repo, fmt.Errorf("add: %w", boom)},
		d.outbox, d.tx, fixedClock{now}, terms(t))

	_, err := place.Execute(t.Context(), placeInput())
	got := wantKind(t, err, application.KindInternal)
	if !errors.Is(err, boom) {
		t.Errorf("原因をたどれない: %v", err)
	}
	// Message は外に出す文なので、原因の文字列を含めない
	if strings.Contains(got.Message, "secret-host") {
		t.Errorf("Message に原因が出た: %s", got.Message)
	}
}

// Tx の無い書き込みの誤りは、利用者の入力の誤りとして見せない。
func TestPlaceBooking_NoTransactionIsKindInternal(t *testing.T) {
	d := newMemoryDeps()
	place := application.NewPlaceBooking(
		failingRepository{d.repo, domain.ErrNoTransaction},
		d.outbox, d.tx, fixedClock{now}, terms(t))
	_, err := place.Execute(t.Context(), placeInput())
	wantKind(t, err, application.KindInternal)
}

func TestPlaceBooking_CanceledContextKeepsCause(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	place := newPlace(t, newMemoryDeps())
	_, err := place.Execute(ctx, placeInput())
	wantKind(t, err, application.KindInternal)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
}

func TestPlaceBooking_OutboxFailureRollsBack(t *testing.T) {
	d := newMemoryDeps()
	boom := errors.New("outbox")
	place := application.NewPlaceBooking(
		d.repo, failingOutbox{boom}, d.tx, fixedClock{now}, terms(t))
	in := placeInput()
	_, err := place.Execute(t.Context(), in)
	wantKind(t, err, application.KindInternal)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
	// 予約が残っていれば、同じ時間帯は重複になる
	_, err = newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Errorf("予約が残った: %v", err)
	}
}

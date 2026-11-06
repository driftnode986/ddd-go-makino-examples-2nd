package pitfalls_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

var (
	jst   = time.FixedZone("JST", 9*60*60)
	start = time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	clock = fixedClock{start.AddDate(0, -1, 0)}
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	return bookingFor(t, ids.NewID[ids.MemberMarker](), start)
}

func bookingFor(
	t *testing.T, member ids.MemberID, at time.Time,
) *domain.Booking {
	t.Helper()
	slot, err := domain.NewTimeSlot(at, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](), member,
		ids.NewID[ids.LessonMarker](), slot, fee, clock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

// flakyOutbox は最初の fails 回だけ失敗し、その後は次に渡す。
type flakyOutbox struct {
	next interface {
		Append(context.Context, ...domain.DomainEvent) error
	}
	fails int
}

var errOutbox = errors.New("outbox に書けない")

func (o *flakyOutbox) Append(
	ctx context.Context, events ...domain.DomainEvent,
) error {
	if o.fails > 0 {
		o.fails--
		return errOutbox
	}
	return o.next.Append(ctx, events...)
}

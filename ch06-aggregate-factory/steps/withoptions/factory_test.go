package withoptions_test

import (
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/steps/withoptions"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var (
	jst     = time.FixedZone("JST", 9*60*60)
	start   = time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	earlier = fixedClock{now: start.AddDate(0, -1, 0)}
)

func yen(t *testing.T, amount int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(amount)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func newWith(
	t *testing.T, opts ...withoptions.Option,
) (*domain.Booking, error) {
	t.Helper()
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	return withoptions.New(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](),
		slot, yen(t, 3000), earlier, opts...,
	)
}

// #region order

func TestOrderChangesResult(t *testing.T) {
	shoes, err := domain.NewBookingOption("シューズ", yen(t, 500), 2)
	if err != nil {
		t.Fatal(err)
	}
	opt := withoptions.WithOption(shoes)
	discount := withoptions.WithDiscount(yen(t, 3500))

	b, err := newWith(t, opt, discount)
	t.Logf("明細→割引: %d円 %v", b.Total().Amount(), err)

	_, err = newWith(t, discount, opt)
	t.Logf("割引→明細: %v", err)
}

// #endregion

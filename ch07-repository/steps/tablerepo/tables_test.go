package tablerepo_test

import (
	"fmt"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/steps/tablerepo"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](),
		slot, fee,
		fixedClock{now: start.AddDate(0, -1, 0)},
	)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

// #region sixth-row

func TestSixthRowIsSaved(t *testing.T) {
	b := newBooking(t)
	var options tablerepo.OptionTable
	for i := range 6 {
		options.Insert(b.ID(), domain.OptionState{
			Name:         fmt.Sprintf("opt-%d", i),
			UnitPriceYen: 100,
			Quantity:     1,
		})
	}
	t.Logf("保存した明細: %d 件", len(options.Rows(b.ID())))

	s := b.State()
	s.Options = options.Rows(b.ID())
	_, err := domain.ReconstructBooking(s)
	t.Logf("読み戻し: %v", err)
}

// #endregion

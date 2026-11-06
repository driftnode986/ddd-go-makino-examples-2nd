// Package encodeaggregate は集約をそのまま JSON にしたときの結果を見せる。
package encodeaggregate_test

import (
	"encoding/json"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Date(2030, time.May, 1, 0, 0, 0, 0, jst)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b, err := domain.NewBooking(ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](), ids.NewID[ids.LessonMarker](),
		slot, fee, fixedClock{now})
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

// #region encode

func TestEncodeBooking(t *testing.T) {
	b := newBooking(t)
	body, err := json.Marshal(b)
	t.Logf("Booking: %s (err = %v)", body, err)
	if string(body) != "{}" {
		t.Fatalf("body = %s", body)
	}

	state, err := json.Marshal(b.State())
	t.Logf("State: %s (err = %v)", state, err)
}

// #endregion

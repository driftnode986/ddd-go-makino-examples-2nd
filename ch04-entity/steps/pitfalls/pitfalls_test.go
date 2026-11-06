// Package pitfalls_test は、第4章で失敗や落とし穴を動かすテストである。
package pitfalls_test

import (
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var (
	jst     = time.FixedZone("JST", 9*60*60)
	start   = time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	earlier = fixedClock{now: start.AddDate(0, -1, 0)}
)

func parts(t *testing.T) (domain.TimeSlot, domain.Money) {
	t.Helper()
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatal(err)
	}
	return slot, fee
}

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	slot, fee := parts(t)
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](),
		slot, fee, earlier,
	)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNewBookingRejectsZeroValues(t *testing.T) {
	id := ids.NewID[ids.BookingMarker]()
	member := ids.NewID[ids.MemberMarker]()
	lesson := ids.NewID[ids.LessonMarker]()
	slot, fee := parts(t)
	// #region zero-values
	_, err := domain.NewBooking(
		id, member, lesson, domain.TimeSlot{}, fee, earlier)
	t.Logf("zero TimeSlot: %v", err)
	_, err = domain.NewBooking(
		id, member, lesson, slot, domain.Money{}, earlier)
	t.Logf("zero Money: %v", err)
	// #endregion
}

func TestZeroBooking(t *testing.T) {
	// #region zero-booking
	b := &domain.Booking{}
	t.Logf("Status=%q", b.Status())
	t.Logf("Confirm: %v", b.Confirm())
	t.Logf("Cancel: %v", b.Cancel(earlier))
	// #endregion
}

func TestOverwrite(t *testing.T) {
	// #region overwrite
	b := newBooking(t)
	other := newBooking(t)
	if err := other.Confirm(); err != nil {
		t.Fatal(err)
	}
	before := b.ID()
	*b = *other // 別の予約の中身で上書きする
	t.Logf("Status=%s", b.Status())
	t.Logf("same ID as before: %v", b.ID().Equal(before))
	// #endregion
}

func TestCopyInRange(t *testing.T) {
	// #region range
	bookings := []domain.Booking{*newBooking(t), *newBooking(t)}
	for _, b := range bookings {
		if err := b.Confirm(); err != nil {
			t.Fatal(err)
		}
		t.Logf("loop variable: %s", b.Status())
	}
	t.Logf("in the slice:  %s", bookings[0].Status())
	// #endregion
	if bookings[0].Status() != domain.StatusPending {
		t.Fatal("スライスの要素は変わらないはず")
	}
}

func TestPointerSliceIsConfirmed(t *testing.T) {
	// #region pointer-slice
	bookings := []*domain.Booking{newBooking(t), newBooking(t)}
	for _, b := range bookings {
		if err := b.Confirm(); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("in the slice: %s", bookings[0].Status())
	// #endregion
	if bookings[0].Status() != domain.StatusConfirmed {
		t.Fatal("ポインタならスライスの要素が変わるはず")
	}
}

func TestPointerComparison(t *testing.T) {
	// #region pointer-equal
	a := newBooking(t)
	slot, fee := parts(t)
	// 同じ ID で組み立てる。DB から読み直したときと同じ形である
	same, err := domain.NewBooking(
		a.ID(), a.MemberID(), a.LessonID(), slot, fee, earlier)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("a == same:     %v", a == same)
	t.Logf("a.Equal(same): %v", a.Equal(same))
	// #endregion
}

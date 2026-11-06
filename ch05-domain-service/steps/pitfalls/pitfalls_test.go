// Package pitfalls_test は、第5章で失敗や落とし穴を動かすテストである。
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

func timeSlot(t *testing.T) domain.TimeSlot {
	t.Helper()
	s, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func yen(t *testing.T, amount int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(amount)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// book は会員の予約を、別のレッスンで作って確定する。
func book(t *testing.T, member ids.MemberID) *domain.Booking {
	t.Helper()
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		member,
		ids.NewID[ids.LessonMarker](),
		timeSlot(t), yen(t, 3000), earlier,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Confirm(); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSameMemberSameSlot(t *testing.T) {
	// #region same-slot
	member := ids.NewID[ids.MemberMarker]()
	first := book(t, member)
	second := book(t, member)
	t.Logf("first:  %s", first.Status())
	t.Logf("second: %s", second.Status())
	t.Logf("same member: %v",
		first.MemberID().Equal(second.MemberID()))
	t.Logf("same slot:   %v", first.Slot().Equal(second.Slot()))
	// #endregion
}

func TestZeroSlot(t *testing.T) {
	s := timeSlot(t)
	// #region zero-slot
	var zero domain.TimeSlot
	t.Logf("Overlaps: %v", zero.Overlaps(s))
	existing := []domain.TimeSlot{s}
	t.Logf("DetectOverlap: %v", domain.DetectOverlap(zero, existing))
	// #endregion
}

func TestCheckThenSave(t *testing.T) {
	s := timeSlot(t)
	// #region check-then-save
	var saved []domain.TimeSlot // 保存済みの時間帯。まだ0件
	// 2つのリクエストが、保存の前に同じ一覧で判定する
	t.Logf("request A: %v", domain.DetectOverlap(s, saved))
	t.Logf("request B: %v", domain.DetectOverlap(s, saved))
	// どちらも判定を通ったので、両方を保存する
	saved = append(saved, s, s)
	t.Logf("saved=%d overlaps=%v",
		len(saved), saved[0].Overlaps(saved[1]))
	// #endregion
}

func TestFloatDiscount(t *testing.T) {
	// #region float
	price, rate := 3000.0, 0.29
	t.Logf("3000 * 0.29 = %v", price*rate)
	t.Logf("truncated   = %d", int64(price*rate))
	// #endregion
}

func TestIntegerDiscount(t *testing.T) {
	// #region integer
	c, err := domain.NewCoupon(29)
	if err != nil {
		t.Fatal(err)
	}
	discount, err := domain.CalculateDiscount(yen(t, 3000), c)
	t.Logf("discount = %d, err = %v", discount.Amount(), err)
	// #endregion
}

func TestPercentFromClient(t *testing.T) {
	fee := yen(t, 3000)
	// #region client-percent
	percent := 100 // リクエストの JSON に入っていた値
	c, err := domain.NewCoupon(percent)
	t.Logf("NewCoupon: err = %v", err)
	discount, err := domain.CalculateDiscount(fee, c)
	t.Logf("fee = %d, discount = %d, err = %v",
		fee.Amount(), discount.Amount(), err)
	// #endregion
}

// Package pitfalls_test は、第6章の Booking の動きを確かめるテストである。
package pitfalls_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"github.com/google/go-cmp/cmp"
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

func timeSlot(t *testing.T) domain.TimeSlot {
	t.Helper()
	s, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](),
		timeSlot(t), yen(t, 3000), earlier,
	)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func add(t *testing.T, b *domain.Booking, name string, price int64) {
	t.Helper()
	o, err := domain.NewBookingOption(name, yen(t, price), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.AddOption(o, earlier); err != nil {
		t.Fatal(err)
	}
}

func coupon(t *testing.T, percent int) domain.Coupon {
	t.Helper()
	c, err := domain.NewCoupon(percent)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// #region apply-coupon

func TestApplyCoupon(t *testing.T) {
	b := newBooking(t)
	add(t, b, "シューズ", 1000)

	discount, err := domain.CalculateDiscount(
		b.Total(), coupon(t, 100))
	if err != nil {
		t.Fatal(err)
	}
	err = b.ApplyDiscount(discount)
	t.Logf("割引: %d円 err=%v", discount.Amount(), err)
	t.Logf("合計: %d円", b.Total().Amount())

	err = b.ApplyDiscount(yen(t, 4001))
	t.Logf("4001円の割引: %v", err)
	t.Logf("合計: %d円", b.Total().Amount())
}

// #endregion

// #region discount-order

func TestDiscountAmountIsFixed(t *testing.T) {
	first := newBooking(t)
	add(t, first, "シューズ", 500)
	d, _ := domain.CalculateDiscount(first.Total(), coupon(t, 10))
	if err := first.ApplyDiscount(d); err != nil {
		t.Fatal(err)
	}
	t.Logf("明細→割引: 割引%d円 合計%d円",
		d.Amount(), first.Total().Amount())

	second := newBooking(t)
	d, _ = domain.CalculateDiscount(second.Total(), coupon(t, 10))
	if err := second.ApplyDiscount(d); err != nil {
		t.Fatal(err)
	}
	shoes, err := domain.NewBookingOption("シューズ", yen(t, 500), 1)
	if err != nil {
		t.Fatal(err)
	}
	err = second.AddOption(shoes, earlier)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("err = %v", err)
	}
	t.Logf("割引→明細: %v", err)
}

// #endregion

// #region past-start

func TestNewBookingCannotRestorePast(t *testing.T) {
	lateClock := fixedClock{now: start.AddDate(0, 1, 0)}
	_, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](),
		timeSlot(t), yen(t, 3000), lateClock,
	)
	t.Logf("終わったレッスンの予約: %v", err)
}

// #endregion

// #region broken-state

func TestReconstructRejectsBrokenRow(t *testing.T) {
	st := newBooking(t).State()
	st.DiscountYen = 5000
	_, err := domain.ReconstructBooking(st)
	t.Logf("割引が超える: %v", err)

	st = newBooking(t).State()
	st.Status = "confirmd"
	_, err = domain.ReconstructBooking(st)
	t.Logf("未知の状態: %v", err)
}

// #endregion

// #region bypass

func TestReconstructSkipsNewBookingChecks(t *testing.T) {
	st := domain.BookingState{
		ID:       ids.NewID[ids.BookingMarker](),
		MemberID: ids.NewID[ids.MemberMarker](),
		LessonID: ids.NewID[ids.LessonMarker](),
		Start:    time.Date(2001, time.January, 1, 3, 15, 0, 0, jst),
		End:      time.Date(2001, time.January, 1, 4, 15, 0, 0, jst),
		FeeYen:   1,
		Status:   domain.StatusConfirmed,
	}
	b, err := domain.ReconstructBooking(st)
	t.Logf("状態: %s err=%v", b.Status(), err)
}

// #endregion

// #region cmp-equal

func TestCmpUsesEqual(t *testing.T) {
	a := newBooking(t)
	b, err := domain.ReconstructBooking(a.State())
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Confirm(); err != nil {
		t.Fatal(err)
	}
	t.Logf("状態: %s と %s", a.Status(), b.Status())
	t.Logf("cmp.Diff が空か: %v", cmp.Diff(a, b) == "")
}

// #endregion

// #region cmp-panic

func TestCmpPanicsOnUnexported(t *testing.T) {
	defer func() { t.Log(firstLine(recover())) }()
	cmp.Diff(*newBooking(t), *newBooking(t))
}

// #endregion

func firstLine(v any) string {
	s := fmt.Sprint(v)
	for i, r := range s {
		if r == '\n' {
			return s[:i]
		}
	}
	return s
}

// #region cmp-state

func TestCmpShowsDifferenceInState(t *testing.T) {
	b := newBooking(t)
	add(t, b, "タオル", 200)
	want := b.State()
	if err := b.Confirm(); err != nil {
		t.Fatal(err)
	}
	got := b.State()
	t.Log(cmp.Diff(want, got))
}

// #endregion

// #region stale-copies

func TestStaleCopiesBothAddOption(t *testing.T) {
	b := newBooking(t)
	for i := range 4 {
		add(t, b, fmt.Sprintf("オプション%d", i+1), 100)
	}
	first, err := domain.ReconstructBooking(b.State())
	if err != nil {
		t.Fatal(err)
	}
	second, err := domain.ReconstructBooking(b.State())
	if err != nil {
		t.Fatal(err)
	}
	add(t, first, "シューズ", 500)
	add(t, second, "タオル", 200)
	t.Logf("1つ目: %d件", len(first.Options()))
	t.Logf("2つ目: %d件", len(second.Options()))
}

// #endregion

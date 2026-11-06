package domain_test

import (
	"errors"
	"math"
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
)

func coupon(t *testing.T, percent int) domain.Coupon {
	t.Helper()
	c, err := domain.NewCoupon(percent)
	if err != nil {
		t.Fatalf("NewCoupon(%d): %v", percent, err)
	}
	return c
}

func TestCalculateDiscount_TruncatesFraction(t *testing.T) {
	cases := []struct {
		price   int64
		percent int
		want    int64
	}{
		{1000, 10, 100},
		{999, 10, 99}, // 99.9円は切り捨てて99円
		{1, 50, 0},    // 0.5円は切り捨てて0円
		{3000, 100, 3000},
		{0, 10, 0},
	}
	for _, c := range cases {
		got, err := domain.CalculateDiscount(
			yen(t, c.price), coupon(t, c.percent))
		if err != nil {
			t.Fatalf("%d, %d%%: %v", c.price, c.percent, err)
		}
		if got.Amount() != c.want {
			t.Errorf("%d, %d%%: got %d, want %d",
				c.price, c.percent, got.Amount(), c.want)
		}
	}
}

func TestCalculateDiscount_RejectsZeroCoupon(t *testing.T) {
	_, err := domain.CalculateDiscount(
		yen(t, 1000), domain.Coupon{})
	if !errors.Is(err, domain.ErrInvalidCoupon) {
		t.Fatalf("err = %v", err)
	}
}

func TestCalculateDiscount_Overflow(t *testing.T) {
	_, err := domain.CalculateDiscount(
		yen(t, math.MaxInt64), coupon(t, 50))
	if !errors.Is(err, domain.ErrMoneyOverflow) {
		t.Fatalf("err = %v", err)
	}
}

func TestNewCoupon_Range(t *testing.T) {
	for _, p := range []int{-1, 0, 101} {
		_, err := domain.NewCoupon(p)
		if !errors.Is(err, domain.ErrInvalidCoupon) {
			t.Errorf("NewCoupon(%d) = %v", p, err)
		}
	}
	var zero domain.Coupon
	if !zero.IsZero() {
		t.Error("var で宣言した Coupon は IsZero が true")
	}
	if coupon(t, 1).IsZero() || coupon(t, 100).Percent() != 100 {
		t.Error("有効な Coupon の値が違う")
	}
}

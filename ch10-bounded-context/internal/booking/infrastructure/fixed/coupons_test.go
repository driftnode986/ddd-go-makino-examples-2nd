package fixed_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/fixed"
)

func TestCoupons_FindsKnownCode(t *testing.T) {
	c, err := fixed.NewCoupons().Find(t.Context(), "WELCOME10")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	want, err := domain.NewCoupon(10)
	if err != nil {
		t.Fatalf("NewCoupon: %v", err)
	}
	if !c.Equal(want) {
		t.Errorf("c = %v, want %v", c, want)
	}
}

func TestCoupons_UnknownCode(t *testing.T) {
	for _, code := range []string{"", "UNKNOWN", "welcome10"} {
		_, err := fixed.NewCoupons().Find(t.Context(), code)
		if !errors.Is(err, application.ErrCouponNotFound) {
			t.Errorf("Find(%q) = %v", code, err)
		}
	}
}

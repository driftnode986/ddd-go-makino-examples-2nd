package fixed_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/infrastructure/fixed"
)

func TestCoupons_FindsKnownCode(t *testing.T) {
	c, err := fixed.NewCoupons().Find(t.Context(), "WELCOME10")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if c.Percent() != 10 {
		t.Errorf("Percent = %d", c.Percent())
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

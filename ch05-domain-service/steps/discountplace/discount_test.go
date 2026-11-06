package discountplace_test

import (
	"testing"

	"example.com/trainingbooking/steps/discountplace"
)

func TestRoundingDiffers(t *testing.T) {
	// #region differs
	coupon := discountplace.Coupon{Percent: 10}
	rank := discountplace.RankDiscount{Percent: 10}
	t.Logf("Coupon:       %d", coupon.DiscountFor(999))
	t.Logf("RankDiscount: %d", rank.DiscountFor(999))
	// #endregion
}

// Package fixed はコードに固定した値でポートを実装する。
package fixed

import (
	"context"
	"fmt"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
)

// couponPercents はコードごとの割引率である。
var couponPercents = map[string]int{
	"WELCOME10": 10,
}

// Coupons は固定の表からクーポンを引く。
type Coupons struct{}

// NewCoupons は Coupons を返す。
func NewCoupons() *Coupons { return &Coupons{} }

// Find はコードの Coupon を返す。
func (*Coupons) Find(
	_ context.Context, code string,
) (domain.Coupon, error) {
	percent, ok := couponPercents[code]
	if !ok {
		return domain.Coupon{}, fmt.Errorf(
			"%w: %q", application.ErrCouponNotFound, code)
	}
	return domain.NewCoupon(percent)
}

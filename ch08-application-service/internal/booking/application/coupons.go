package application

import (
	"context"
	"errors"

	"example.com/trainingbooking/internal/booking/domain"
)

// ErrCouponNotFound はクーポンのコードが見つからないことを表す。
var ErrCouponNotFound = errors.New("coupon not found")

// #region coupons

// Coupons はクーポンのコードから Coupon を引く。
type Coupons interface {
	// Find はコードの Coupon を返す。無ければ ErrCouponNotFound を
	// 返す。
	Find(ctx context.Context, code string) (domain.Coupon, error)
}

// #endregion

package domain

import "fmt"

// #region coupon

// Coupon は1〜100パーセントの割引券である。
type Coupon struct {
	percent int
}

// NewCoupon は割引率から Coupon を作る。
func NewCoupon(percent int) (Coupon, error) {
	if percent < 1 || percent > 100 {
		return Coupon{}, fmt.Errorf(
			"%w: percent=%d", ErrInvalidCoupon, percent)
	}
	return Coupon{percent: percent}, nil
}

// #endregion

// IsZero は NewCoupon を通していないとき true を返す。
func (c Coupon) IsZero() bool { return c.percent == 0 }

// Equal は割引率が同じとき true を返す。
func (c Coupon) Equal(other Coupon) bool {
	return c.percent == other.percent
}

// #region calculate-discount

// CalculateDiscount は割引額を計算する。端数は切り捨てる。
func CalculateDiscount(price Money, c Coupon) (Money, error) {
	if c.IsZero() {
		return Money{}, ErrInvalidCoupon
	}
	scaled, err := price.Multiply(c.percent)
	if err != nil {
		return Money{}, err
	}
	return Money{yen: scaled.yen / 100}, nil
}

// #endregion

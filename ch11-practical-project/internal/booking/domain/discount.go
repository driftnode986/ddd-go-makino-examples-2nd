package domain

import "fmt"

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

// Percent は割引率を返す。
func (c Coupon) Percent() int { return c.percent }

// IsZero は NewCoupon を通していないとき true を返す。
func (c Coupon) IsZero() bool { return c.percent == 0 }

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

// Package discountplace は、割引の計算を割引の種類ごとのメソッドに
// 置く例である。Money の代わりに円を int64 で扱う。
package discountplace

// #region methods

// Coupon はクーポンの割引である。
type Coupon struct{ Percent int64 }

// DiscountFor は割引額を計算する。端数は切り捨てる。
func (c Coupon) DiscountFor(price int64) int64 {
	return price * c.Percent / 100
}

// RankDiscount は会員ランクの割引である。別の時期に書き足した。
type RankDiscount struct{ Percent int64 }

// DiscountFor は割引額を計算する。端数は四捨五入する。
func (r RankDiscount) DiscountFor(price int64) int64 {
	return (price*r.Percent + 50) / 100
}

// #endregion

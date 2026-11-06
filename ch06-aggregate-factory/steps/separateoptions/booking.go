// Package separateoptions は、明細を Booking の外に置く例である。
// Money の代わりに円を int64 で扱う。
package separateoptions

import "errors"

// ErrNotAllowed は追加できない明細を表す。
var ErrNotAllowed = errors.New("option is not allowed")

// #region types

// Booking は明細を持たない予約である。
type Booking struct {
	FeeYen      int64
	DiscountYen int64
	Status      string
}

// Option は予約とは別に扱う明細である。
type Option struct {
	Name         string
	UnitPriceYen int64
	Quantity     int64
}

// #endregion

// Total は料金と明細の合計から割引を引く。
func Total(b Booking, options []Option) int64 {
	sum := b.FeeYen
	for _, o := range options {
		sum += o.UnitPriceYen * o.Quantity
	}
	return sum - b.DiscountYen
}

// AddFromForm は画面からの追加である。ルールをここで確かめる。
func AddFromForm(
	b Booking, options []Option, o Option,
) ([]Option, error) {
	if b.Status != "pending" || len(options) >= 5 {
		return nil, ErrNotAllowed
	}
	return append(options, o), nil
}

// #region other-paths

// AddFromBatch は別の時期に足した一括登録である。
func AddFromBatch(options []Option, o Option) []Option {
	return append(options, o)
}

// CorrectQuantity は別の担当者が足した数量の訂正である。
func CorrectQuantity(options []Option, name string, n int64) {
	for i := range options {
		if options[i].Name == name {
			options[i].Quantity = n
		}
	}
}

// #endregion

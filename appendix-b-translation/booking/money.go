// Package booking は Kotlin と Spring の書き方を Go に対応させる
// 最小の例である。本文の方針に合わせ、各所に対応を書く。
package booking

import "errors"

// ErrInvalidMoney は不正な金額を表す。
var ErrInvalidMoney = errors.New("invalid money")

// Money は値オブジェクトである。
//
//	data class Money(val yen: Long) {
//	    init { require(yen >= 0) }
//	}
//
// init と例外の代わりに、コンストラクタ関数が error を返す。
// == と copy は自動で作られないので、Equal を書く。
type Money struct {
	yen int64
}

// NewMoney は Money を作る。
func NewMoney(yen int64) (Money, error) {
	if yen < 0 {
		return Money{}, ErrInvalidMoney
	}
	return Money{yen: yen}, nil
}

// Equal は同じ金額のとき true を返す。
func (m Money) Equal(other Money) bool { return m.yen == other.yen }

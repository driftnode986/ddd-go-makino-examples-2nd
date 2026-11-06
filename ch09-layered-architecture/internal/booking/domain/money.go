// Package domain は Booking コンテキストのドメインモデルである。
package domain

import (
	"errors"
	"fmt"
	"math"
)

// Money の計算で起きるエラーである。
var (
	ErrNegativeMoney = errors.New("negative money")
	ErrMoneyOverflow = errors.New("money overflow")
)

// Money は日本円の金額を表す値オブジェクトである。
type Money struct {
	yen int64
}

// NewMoney は金額から Money を作る。負の金額は拒否する。
func NewMoney(yen int64) (Money, error) {
	if yen < 0 {
		return Money{}, fmt.Errorf("%w: %d", ErrNegativeMoney, yen)
	}
	return Money{yen: yen}, nil
}

// Add は合計を返す。int64 を超えるときはエラーを返す。
func (m Money) Add(other Money) (Money, error) {
	if m.yen > math.MaxInt64-other.yen {
		return Money{}, ErrMoneyOverflow
	}
	return Money{yen: m.yen + other.yen}, nil
}

// Multiply は n 倍の金額を返す。負の n は拒否する。
func (m Money) Multiply(n int) (Money, error) {
	if n < 0 {
		return Money{}, fmt.Errorf(
			"%w: factor %d", ErrNegativeMoney, n)
	}
	if n != 0 && m.yen > math.MaxInt64/int64(n) {
		return Money{}, ErrMoneyOverflow
	}
	return Money{yen: m.yen * int64(n)}, nil
}

// Amount は金額を円で返す。
func (m Money) Amount() int64 { return m.yen }

// IsZero は0円のとき true を返す。
func (m Money) IsZero() bool { return m.yen == 0 }

// Equal は同じ金額のとき true を返す。
func (m Money) Equal(other Money) bool { return m.yen == other.yen }

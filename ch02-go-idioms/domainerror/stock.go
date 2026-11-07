// Package domainerror は、業務の失敗を error で返す2つの形を示す。
package domainerror

import (
	"errors"
	"fmt"
)

// #region sentinel

// ErrSoldOut は判定に種類だけが要るときの sentinel である。
var ErrSoldOut = errors.New("sold out")

// ErrInvalidQuantity は負の在庫や0以下の数量を表す。
var ErrInvalidQuantity = errors.New("invalid quantity")

// #endregion

// #region typed

// ShortageError は不足数も伝えたいときの型つきエラーである。
type ShortageError struct {
	Want int
	Have int
}

// Error はエラーの文字列を返す。
func (e *ShortageError) Error() string {
	return fmt.Sprintf("want %d, have %d", e.Want, e.Have)
}

// #endregion

// #region stock

// Stock は在庫である。
type Stock struct {
	count int
}

// NewStock は在庫を作る。負の数は拒否する。
func NewStock(count int) (*Stock, error) {
	if count < 0 {
		return nil, ErrInvalidQuantity
	}
	return &Stock{count: count}, nil
}

// Take は在庫を減らす。失敗は戻り値で返す。
func (s *Stock) Take(n int) error {
	if n <= 0 {
		return ErrInvalidQuantity
	}
	if s.count == 0 {
		return fmt.Errorf("take %d: %w", n, ErrSoldOut)
	}
	if n > s.count {
		return &ShortageError{Want: n, Have: s.count}
	}
	s.count -= n
	return nil
}

// #endregion

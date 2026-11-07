// Package service は、在庫のルールを型の外に置いた形である。
package service

import (
	"errors"

	"example.com/trainingbooking/steps/setterstock"
)

// ErrShortage は在庫が足りないことを表す。
var ErrShortage = errors.New("shortage")

// #region take

// Take は在庫が足りるときだけ減らす。
func Take(s *setterstock.Stock, n int) error {
	if n > s.Count() {
		return ErrShortage
	}
	s.SetCount(s.Count() - n)
	return nil
}

// #endregion

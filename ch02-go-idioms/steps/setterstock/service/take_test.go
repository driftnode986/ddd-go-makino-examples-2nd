package service_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/steps/setterstock"
	"example.com/trainingbooking/steps/setterstock/service"
)

// #region bypass

func TestBypass(t *testing.T) {
	s := setterstock.NewStock(2)
	err := service.Take(s, 5)
	t.Logf("Take(5) の err = %v, Count = %d", err, s.Count())

	// 別の担当者が足した棚卸しの処理。Take を通らない。
	s.SetCount(s.Count() - 5)
	t.Logf("SetCount の後の Count = %d", s.Count())

	if !errors.Is(err, service.ErrShortage) {
		t.Fatalf("err = %v", err)
	}
}

// #endregion

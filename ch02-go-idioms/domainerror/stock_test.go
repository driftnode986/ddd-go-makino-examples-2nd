package domainerror_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/domainerror"
)

// #region compare

// %w で包むと == では一致しない。errors.Is は包みをたどる。
func TestTake_CompareSentinel(t *testing.T) {
	err := domainerror.NewStock(0).Take(1)

	t.Logf("err = %v", err)
	t.Logf("err == ErrSoldOut: %v", err == domainerror.ErrSoldOut)
	t.Logf("errors.Is: %v", errors.Is(err, domainerror.ErrSoldOut))
	if !errors.Is(err, domainerror.ErrSoldOut) {
		t.Fatal("errors.Is = false")
	}
}

// #endregion

// #region typed

// 型つきエラーは errors.AsType で値を取り出す。
func TestTake_TypedError(t *testing.T) {
	err := domainerror.NewStock(2).Take(3)

	got, ok := errors.AsType[*domainerror.ShortageError](err)

	if !ok {
		t.Fatalf("err = %v", err)
	}
	t.Logf("足りない数: %d", got.Want-got.Have)
}

// #endregion

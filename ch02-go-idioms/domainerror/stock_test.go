package domainerror_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/domainerror"
)

func stock(t *testing.T, count int) *domainerror.Stock {
	t.Helper()
	s, err := domainerror.NewStock(count)
	if err != nil {
		t.Fatalf("NewStock: %v", err)
	}
	return s
}

// #region compare

// %w で包むと == では一致しない。errors.Is は包みをたどる。
func TestTake_CompareSentinel(t *testing.T) {
	err := stock(t, 0).Take(1)

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
	err := stock(t, 2).Take(3)

	got, ok := errors.AsType[*domainerror.ShortageError](err)

	if !ok {
		t.Fatalf("err = %v", err)
	}
	t.Logf("足りない数: %d", got.Want-got.Have)
}

// #endregion

func TestStock_RejectsInvalidQuantity(t *testing.T) {
	if _, err := domainerror.NewStock(-1); !errors.Is(
		err, domainerror.ErrInvalidQuantity) {
		t.Fatalf("NewStock(-1): %v", err)
	}
	s := stock(t, 2)
	for _, n := range []int{0, -5} {
		if err := s.Take(n); !errors.Is(
			err, domainerror.ErrInvalidQuantity) {
			t.Fatalf("Take(%d): %v", n, err)
		}
	}
}

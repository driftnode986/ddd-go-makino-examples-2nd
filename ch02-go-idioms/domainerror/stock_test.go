package domainerror_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/domainerror"
)

// %w で包んでも errors.Is で判定できる。
func TestTake_SentinelIsFoundThroughWrap(t *testing.T) {
	err := domainerror.NewStock(0).Take(1)
	if !errors.Is(err, domainerror.ErrSoldOut) {
		t.Fatalf("err = %v", err)
	}
}

// 型つきエラーは errors.AsType で値を取り出す。
func TestTake_TypedErrorCarriesValues(t *testing.T) {
	err := domainerror.NewStock(2).Take(3)
	got, ok := errors.AsType[*domainerror.ShortageError](err)
	if !ok {
		t.Fatalf("err = %v", err)
	}
	if got.Want != 3 || got.Have != 2 {
		t.Fatalf("got = %+v", got)
	}
}

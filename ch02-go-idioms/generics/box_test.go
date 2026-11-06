package generics_test

import (
	"reflect"
	"strconv"
	"testing"

	"example.com/trainingbooking/generics"
)

func TestBox_Map(t *testing.T) {
	got := generics.NewBox(7).Map(strconv.Itoa)
	if got.Value() != "7" {
		t.Fatalf("Value = %q", got.Value())
	}
}

// ジェネリックメソッドはメソッド集合に入らない。
// 次の interface を満たす型は書けない。
//
//	type Mapper interface {
//		Map[U any](f func(int) U) Box[U] // コンパイルエラー
//	}
func TestBox_GenericMethodIsNotInMethodSet(t *testing.T) {
	typ := reflect.TypeFor[generics.Box[int]]()
	if _, ok := typ.MethodByName("Map"); ok {
		t.Fatal("Map がメソッド集合にある")
	}
	if _, ok := typ.MethodByName("Value"); !ok {
		t.Fatal("Value がメソッド集合にない")
	}
}

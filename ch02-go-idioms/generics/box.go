// Package generics はジェネリクスを使う場面と、Go 1.27 の
// ジェネリックメソッドの制約を示す。
package generics

// Box は1つの値を持つ型である。型引数で中身の型を決める。
type Box[T any] struct {
	value T
}

// NewBox は Box を作る。
func NewBox[T any](v T) Box[T] { return Box[T]{value: v} }

// Value は中身を返す。
func (b Box[T]) Value() T { return b.value }

// Map は中身を変換する。メソッドに型引数を持てるのは Go 1.27
// からである。interface のメソッドには書けず、interface の実装
// にも数えない。
func (b Box[T]) Map[U any](f func(T) U) Box[U] {
	return Box[U]{value: f(b.value)}
}

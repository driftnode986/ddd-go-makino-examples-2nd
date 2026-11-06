// Package shared は、コンテキストをまたいで使う ID を提供する。
// 共有カーネルはこのパッケージの内容だけである。
package shared

import (
	"errors"
	"fmt"
	"uuid"
)

// ErrInvalidID は ID の文字列が不正なことを表す。
var ErrInvalidID = errors.New("invalid id")

// ID は集約の識別子である。T は値を持たないマーカー型で、
// 異なる集約の ID を別の型として区別する。
type ID[T any] struct {
	value uuid.UUID
}

// NewID は時刻順に並ぶ UUID v7 で ID を採番する。
func NewID[T any]() ID[T] {
	return ID[T]{value: uuid.NewV7()}
}

// ParseID は文字列から ID を作る。ゼロの ID は拒否する。
func ParseID[T any](s string) (ID[T], error) {
	v, err := uuid.Parse(s)
	if err != nil {
		return ID[T]{}, fmt.Errorf("%w: %q", ErrInvalidID, s)
	}
	if v == uuid.Nil() {
		return ID[T]{}, fmt.Errorf("%w: nil uuid", ErrInvalidID)
	}
	return ID[T]{value: v}, nil
}

// String は UUID の標準の表記を返す。
func (id ID[T]) String() string { return id.value.String() }

// IsZero は値を持たない ID のとき true を返す。
func (id ID[T]) IsZero() bool { return id.value == uuid.Nil() }

// Equal は同じ ID のとき true を返す。
func (id ID[T]) Equal(other ID[T]) bool {
	return id.value == other.value
}

// 各 ID のマーカー型である。
type (
	MemberMarker  struct{}
	LessonMarker  struct{}
	BookingMarker struct{}
)

// 各集約の ID の型である。
type (
	MemberID  = ID[MemberMarker]
	LessonID  = ID[LessonMarker]
	BookingID = ID[BookingMarker]
)

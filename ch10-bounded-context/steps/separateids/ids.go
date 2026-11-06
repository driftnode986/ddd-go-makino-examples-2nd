// Package separateids は、ID ごとに別の型を書く代案である。
// 本書の ID[T] と比べるために置く。
package separateids

import "uuid"

// #region member
type MemberID struct{ value uuid.UUID }

func NewMemberID() MemberID {
	return MemberID{value: uuid.NewV7()}
}

func (id MemberID) String() string { return id.value.String() }

func (id MemberID) IsZero() bool { return id.value == uuid.Nil() }

func (id MemberID) Equal(other MemberID) bool {
	return id.value == other.value
}

// #endregion

// LessonID はレッスンの ID である。同じメソッドをもう一度書く。
type LessonID struct{ value uuid.UUID }

// NewLessonID は ID を採番する。
func NewLessonID() LessonID {
	return LessonID{value: uuid.NewV7()}
}

// String は UUID の標準の表記を返す。
func (id LessonID) String() string { return id.value.String() }

// IsZero は値を持たない ID のとき true を返す。
func (id LessonID) IsZero() bool { return id.value == uuid.Nil() }

// Equal は同じ ID のとき true を返す。
func (id LessonID) Equal(other LessonID) bool {
	return id.value == other.value
}

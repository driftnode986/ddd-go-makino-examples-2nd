// Package stringargs は、値を string と int のまま渡す設計の例である。
// 第3章の最初の節で、取り違えが通ることを確かめるために使う。
package stringargs

import "time"

// Member は会員である。
type Member struct {
	Name  string
	Email string
}

// Booking は予約である。
type Booking struct {
	MemberID string
	LessonID string
	PriceYen int
	Start    time.Time
}

// #region signatures
func RegisterMember(name, email string) Member {
	return Member{Name: name, Email: email}
}

func PlaceBooking(
	memberID, lessonID string, priceYen int, start time.Time,
) Booking {
	return Booking{memberID, lessonID, priceYen, start}
}

// #endregion

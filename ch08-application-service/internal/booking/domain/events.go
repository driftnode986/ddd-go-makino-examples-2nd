package domain

import (
	"time"

	"example.com/trainingbooking/internal/ids"
)

// #region domain-event

// DomainEvent は集約が記録する出来事である。
type DomainEvent interface {
	EventName() string
	AggregateID() string
	When() time.Time
}

// BookingPlaced は予約が作られたことを表す。
type BookingPlaced struct {
	BookingID  ids.BookingID
	MemberID   ids.MemberID
	LessonID   ids.LessonID
	OccurredAt time.Time
}

// #endregion

// EventName はイベントの名前を返す。
func (BookingPlaced) EventName() string { return "booking.placed" }

// AggregateID はイベントを記録した集約の ID を返す。
func (e BookingPlaced) AggregateID() string {
	return e.BookingID.String()
}

// When は出来事の時刻を返す。
func (e BookingPlaced) When() time.Time { return e.OccurredAt }

// BookingCancelled は確定済みの予約が取り消されたことを表す。
type BookingCancelled struct {
	BookingID  ids.BookingID
	LessonID   ids.LessonID
	OccurredAt time.Time
}

// EventName はイベントの名前を返す。
func (BookingCancelled) EventName() string {
	return "booking.cancelled"
}

// AggregateID はイベントを記録した集約の ID を返す。
func (e BookingCancelled) AggregateID() string {
	return e.BookingID.String()
}

// When は出来事の時刻を返す。
func (e BookingCancelled) When() time.Time { return e.OccurredAt }

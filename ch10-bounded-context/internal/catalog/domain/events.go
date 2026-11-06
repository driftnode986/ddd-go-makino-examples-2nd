package domain

import (
	"time"

	"example.com/trainingbooking/internal/ids"
)

// DomainEvent は集約が記録する出来事である。
type DomainEvent interface {
	EventName() string
	AggregateID() string
	When() time.Time
}

// RejectReason は席を確保できなかった理由である。
type RejectReason string

// 席を確保できなかった理由である。
const (
	RejectLessonFull     RejectReason = "lesson_full"
	RejectLessonNotFound RejectReason = "lesson_not_found"
)

// SeatReserved は予約の席を確保したことを表す。
type SeatReserved struct {
	LessonID   ids.LessonID
	BookingID  ids.BookingID
	OccurredAt time.Time
}

// EventName はイベントの名前を返す。
func (SeatReserved) EventName() string { return "seat.reserved" }

// AggregateID はイベントを記録した集約の ID を返す。
func (e SeatReserved) AggregateID() string {
	return e.LessonID.String()
}

// When は出来事の時刻を返す。
func (e SeatReserved) When() time.Time { return e.OccurredAt }

// SeatRejected は予約の席を確保できなかったことを表す。
type SeatRejected struct {
	LessonID   ids.LessonID
	BookingID  ids.BookingID
	Reason     RejectReason
	OccurredAt time.Time
}

// EventName はイベントの名前を返す。
func (SeatRejected) EventName() string { return "seat.rejected" }

// AggregateID はイベントを記録した集約の ID を返す。
func (e SeatRejected) AggregateID() string {
	return e.LessonID.String()
}

// When は出来事の時刻を返す。
func (e SeatRejected) When() time.Time { return e.OccurredAt }

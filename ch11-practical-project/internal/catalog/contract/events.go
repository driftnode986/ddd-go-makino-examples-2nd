// Package contract は Catalog が他のコンテキストへ公開する型である。
// 他のコンテキストから import できるのはこのパッケージだけである。
package contract

import "time"

// 統合イベントの名前である。保存されるので変えない。
const (
	EventSeatReserved = "seat.reserved"
	EventSeatRejected = "seat.rejected"
)

// 席を確保できなかった理由である。
const (
	ReasonLessonFull     = "lesson_full"
	ReasonLessonNotFound = "lesson_not_found"
)

// SeatReserved は予約の席を確保したことを伝える統合イベントである。
type SeatReserved struct {
	LessonID   string    `json:"lesson_id"`
	BookingID  string    `json:"booking_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

// SeatRejected は予約の席を確保できなかったことを伝える
// 統合イベントである。
type SeatRejected struct {
	LessonID   string    `json:"lesson_id"`
	BookingID  string    `json:"booking_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Package contract は Booking が他のコンテキストへ公開する型である。
// 他のコンテキストから import できるのはこのパッケージだけである。
package contract

import "time"

// 統合イベントの名前である。保存されるので変えない。
const (
	EventBookingPlaced    = "booking.placed"
	EventBookingCancelled = "booking.cancelled"
)

// BookingPlaced は予約を受け付けたことを伝える統合イベントである。
type BookingPlaced struct {
	BookingID  string    `json:"booking_id"`
	MemberID   string    `json:"member_id"`
	LessonID   string    `json:"lesson_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

// BookingCancelled は確定済みの予約を取り消したことを伝える
// 統合イベントである。
type BookingCancelled struct {
	BookingID  string    `json:"booking_id"`
	LessonID   string    `json:"lesson_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

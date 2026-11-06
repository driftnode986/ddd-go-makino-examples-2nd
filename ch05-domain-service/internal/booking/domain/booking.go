package domain

import (
	"fmt"
	"time"

	"example.com/trainingbooking/internal/ids"
)

// Status は予約の状態である。
type Status string

// 予約の状態である。
const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusRejected  Status = "rejected"
	StatusCancelled Status = "cancelled"
)

// changeDeadline はキャンセルの期限である。
// 開始からこの時間より前まで受け付ける。
const changeDeadline = 24 * time.Hour

// Booking は予約のエンティティである。
// レッスンと時間帯は作成後に変えない。
type Booking struct {
	id       ids.BookingID
	memberID ids.MemberID
	lessonID ids.LessonID
	slot     TimeSlot
	fee      Money
	status   Status
}

// NewBooking は pending の予約を作る。
// ID の採番は呼び出し側で行う。
func NewBooking(
	id ids.BookingID,
	member ids.MemberID,
	lesson ids.LessonID,
	slot TimeSlot,
	fee Money,
	clk Clock,
) (*Booking, error) {
	if id.IsZero() || member.IsZero() || lesson.IsZero() ||
		slot.IsZero() || fee.IsZero() || clk == nil {
		return nil, fmt.Errorf(
			"%w: required value is missing", ErrInvalidBooking)
	}
	now := clk.Now()
	if !slot.Start().After(now) {
		return nil, fmt.Errorf(
			"%w: start=%s", ErrStartInPast, slot.Start())
	}
	return &Booking{
		id:       id,
		memberID: member,
		lessonID: lesson,
		slot:     slot,
		fee:      fee,
		status:   StatusPending,
	}, nil
}

// ID は予約の ID を返す。
func (b *Booking) ID() ids.BookingID { return b.id }

// MemberID は会員の ID を返す。
func (b *Booking) MemberID() ids.MemberID { return b.memberID }

// LessonID はレッスンの ID を返す。
func (b *Booking) LessonID() ids.LessonID { return b.lessonID }

// Slot は時間帯を返す。
func (b *Booking) Slot() TimeSlot { return b.slot }

// Status は状態を返す。
func (b *Booking) Status() Status { return b.status }

// Equal は同じ予約のとき true を返す。エンティティなので ID で判定する。
func (b *Booking) Equal(other *Booking) bool {
	return other != nil && b.id.Equal(other.id)
}

// Confirm は席が確保できた予約を確定する。
func (b *Booking) Confirm() error {
	if b.status != StatusPending {
		return b.invalidTransition("confirm")
	}
	b.status = StatusConfirmed
	return nil
}

// Reject は席が確保できなかった予約を拒否の状態にする。
func (b *Booking) Reject() error {
	if b.status != StatusPending {
		return b.invalidTransition("reject")
	}
	b.status = StatusRejected
	return nil
}

// Cancel は確定済みの予約を開始の24時間前まで取り消す。
// pending は席の確保と行き違うので取り消せない。
func (b *Booking) Cancel(clk Clock) error {
	if b.status != StatusConfirmed {
		return b.invalidTransition("cancel")
	}
	now := clk.Now()
	if b.pastDeadline(now) {
		return fmt.Errorf("%w: now=%s", ErrCancelTooLate, now)
	}
	b.status = StatusCancelled
	return nil
}

// pastDeadline は開始の24時間前を過ぎたとき true を返す。
func (b *Booking) pastDeadline(now time.Time) bool {
	return now.After(b.slot.Start().Add(-changeDeadline))
}

func (b *Booking) invalidTransition(op string) error {
	return fmt.Errorf(
		"%w: %s from %q", ErrInvalidTransition, op, b.status)
}

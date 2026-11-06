package domain

import (
	"fmt"
	"slices"
	"time"

	"example.com/trainingbooking/internal/ids"
)

// #region status

// Status は予約の状態である。
type Status string

// 予約の状態である。
const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusRejected  Status = "rejected"
	StatusCancelled Status = "cancelled"
)

// #endregion

const (
	// changeDeadline はキャンセルと明細の追加の期限である。
	// 開始からこの時間より前まで受け付ける。
	changeDeadline = 24 * time.Hour
	// maxOptions は1予約あたりの明細の上限である。
	maxOptions = 5
)

// Booking は予約の集約ルートである。
// レッスンと時間帯は作成後に変えない。
type Booking struct {
	id       ids.BookingID
	memberID ids.MemberID
	lessonID ids.LessonID
	slot     TimeSlot
	fee      Money
	options  []BookingOption
	subtotal Money // 料金と明細の合計
	discount Money
	status   Status
	version  int // 読み込んだときの版。メソッドでは増やさない
	events   []DomainEvent
}

// NewBooking は pending の予約を作り、BookingPlaced を記録する。
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
		subtotal: fee,
		status:   StatusPending,
		events: []DomainEvent{BookingPlaced{
			BookingID:  id,
			MemberID:   member,
			LessonID:   lesson,
			OccurredAt: now,
		}},
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

// #region status-getter

// Status は状態を返す。
func (b *Booking) Status() Status { return b.status }

// #endregion

// #region version-getter

// Version は読み込んだときの版を返す。
func (b *Booking) Version() int { return b.version }

// #endregion

// #region options-getter

// Options は明細のコピーを返す。
func (b *Booking) Options() []BookingOption {
	return slices.Clone(b.options)
}

// #endregion

// #region total

// Total は料金と明細の合計から割引を引いた額を返す。
// 割引が小計を超えないことは setOptions と ApplyDiscount が守る。
func (b *Booking) Total() Money {
	return Money{yen: b.subtotal.yen - b.discount.yen}
}

// #endregion

// #region equal

// Equal は同じ予約のとき true を返す。
// エンティティなので ID で判定する。
func (b *Booking) Equal(other *Booking) bool {
	return other != nil && b.id.Equal(other.id)
}

// #endregion

// #region confirm-reject

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

// #endregion

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
	b.events = append(b.events, BookingCancelled{
		BookingID:  b.id,
		LessonID:   b.lessonID,
		OccurredAt: now,
	})
	return nil
}

// #region add-option

// AddOption は pending の予約に、開始の24時間前まで明細を追加する。
func (b *Booking) AddOption(o BookingOption, clk Clock) error {
	if b.status != StatusPending {
		return b.invalidTransition("add option")
	}
	if now := clk.Now(); b.pastDeadline(now) {
		return fmt.Errorf("%w: now=%s", ErrChangeTooLate, now)
	}
	if o.IsZero() {
		return ErrInvalidOption
	}
	return b.setOptions(append(slices.Clone(b.options), o))
}

// #endregion

// #region apply-discount

// ApplyDiscount は pending の予約に割引額を設定する。
// 前の割引は置き換える。
func (b *Booking) ApplyDiscount(discount Money) error {
	if b.status != StatusPending {
		return b.invalidTransition("apply discount")
	}
	if discount.yen > b.subtotal.yen {
		return fmt.Errorf("%w: discount=%d subtotal=%d",
			ErrDiscountTooLarge, discount.yen, b.subtotal.yen)
	}
	b.discount = discount
	return nil
}

// #endregion

// #region pull-events

// PullEvents は記録したイベントを取り出し、集約から消す。
func (b *Booking) PullEvents() []DomainEvent {
	events := b.events
	b.events = nil
	return events
}

// #endregion

// #region past-deadline

// pastDeadline は開始の24時間前を過ぎたとき true を返す。
func (b *Booking) pastDeadline(now time.Time) bool {
	return now.After(b.slot.Start().Add(-changeDeadline))
}

// #endregion

// #region set-options

// setOptions は明細の不変条件を確かめてから入れ替える。
// 追加と再構築はここを通る。
func (b *Booking) setOptions(options []BookingOption) error {
	if len(options) > maxOptions {
		return fmt.Errorf("%w: %d", ErrOptionLimit, len(options))
	}
	seen := make(map[string]bool, len(options))
	subtotal := b.fee
	for _, o := range options {
		if seen[o.name] {
			return fmt.Errorf("%w: %q", ErrDuplicateOption, o.name)
		}
		seen[o.name] = true
		var err error
		if subtotal, err = subtotal.Add(o.subtotal); err != nil {
			return err
		}
	}
	if b.discount.yen > subtotal.yen {
		return fmt.Errorf("%w: discount=%d subtotal=%d",
			ErrDiscountTooLarge, b.discount.yen, subtotal.yen)
	}
	b.options, b.subtotal = options, subtotal
	return nil
}

// #endregion

// #region invalid-transition

func (b *Booking) invalidTransition(op string) error {
	return fmt.Errorf(
		"%w: %s from %q", ErrInvalidTransition, op, b.status)
}

// #endregion

// valid は定義済みの状態のとき true を返す。
func (s Status) valid() bool {
	return s == StatusPending || s == StatusConfirmed ||
		s == StatusRejected || s == StatusCancelled
}

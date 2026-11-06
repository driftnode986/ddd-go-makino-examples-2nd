package booking

import (
	"errors"
	"time"
	"uuid"
)

// 業務の失敗は例外でなく error で返す。
//
//	class InvalidTransitionException : RuntimeException()
var (
	ErrInvalidBooking    = errors.New("invalid booking")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// BookingID は集約の ID である。
//
//	@Id val id: UUID
type BookingID struct {
	value uuid.UUID
}

// NewBookingID は ID を採番する。
func NewBookingID() BookingID {
	return BookingID{value: uuid.NewV7()}
}

// Status は予約の状態である。
//
//	enum class Status { CONFIRMED, CANCELLED }
type Status string

// 予約の状態である。
const (
	StatusConfirmed Status = "confirmed"
	StatusCancelled Status = "cancelled"
)

// Clock は現在時刻を提供する。time.Now をドメインで呼ばない。
type Clock interface {
	Now() time.Time
}

// Event は集約が記録する出来事である。
//
//	@DomainEvents fun events(): List<Any>
type Event interface {
	EventName() string
}

// BookingCancelled は予約が取り消されたことを表す。
type BookingCancelled struct {
	BookingID BookingID
	At        time.Time
}

// EventName はイベントの名前を返す。
func (BookingCancelled) EventName() string {
	return "booking.cancelled"
}

// Booking はエンティティである。
//
//	@Entity class Booking(@Id val id: UUID, ...) {
//	    fun cancel() { ... }
//	}
//
// アノテーションも永続化の情報も持たない。
// 状態を持つのでポインタレシーバにそろえる。
type Booking struct {
	id      BookingID
	fee     Money
	status  Status
	version int
	events  []Event
}

// NewBooking は確定済みの予約を作る。
func NewBooking(id BookingID, fee Money) *Booking {
	return &Booking{id: id, fee: fee, status: StatusConfirmed}
}

// ID は ID を返す。getter に Get を付けない。
func (b *Booking) ID() BookingID { return b.id }

// Status は状態を返す。
func (b *Booking) Status() Status { return b.status }

// Cancel は予約を取り消す。現在時刻は Clock から受け取る。
func (b *Booking) Cancel(clk Clock) error {
	if b.status != StatusConfirmed {
		return ErrInvalidTransition
	}
	b.status = StatusCancelled
	b.events = append(b.events,
		BookingCancelled{BookingID: b.id, At: clk.Now()})
	return nil
}

// PullEvents は記録したイベントを取り出す。
func (b *Booking) PullEvents() []Event {
	events := b.events
	b.events = nil
	return events
}

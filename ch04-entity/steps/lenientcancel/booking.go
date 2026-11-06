// Package lenientcancel は、保留の予約もキャンセルできるようにした例である。
package lenientcancel

import "example.com/trainingbooking/internal/booking/domain"

// Booking は説明用の小さな予約である。
type Booking struct {
	status domain.Status
}

// New は保留の予約を作る。
func New() *Booking { return &Booking{status: domain.StatusPending} }

// Status は状態を返す。
func (b *Booking) Status() domain.Status { return b.status }

// Confirm は保留の予約を確定する。
func (b *Booking) Confirm() error {
	if b.status != domain.StatusPending {
		return domain.ErrInvalidTransition
	}
	b.status = domain.StatusConfirmed
	return nil
}

// #region cancel

// Cancel は保留の予約も受け付ける。
func (b *Booking) Cancel() error {
	if b.status != domain.StatusPending &&
		b.status != domain.StatusConfirmed {
		return domain.ErrInvalidTransition
	}
	b.status = domain.StatusCancelled
	return nil
}

// #endregion

// Lesson は説明用の小さなレッスンである。
type Lesson struct {
	Reserved int
}

// ReserveSeat は席を1つ確保する。
func (l *Lesson) ReserveSeat() { l.Reserved++ }

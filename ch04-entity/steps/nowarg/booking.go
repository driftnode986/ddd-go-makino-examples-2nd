// Package nowarg は、現在時刻を引数で受け取る例である。
package nowarg

import (
	"time"

	"example.com/trainingbooking/internal/booking/domain"
)

// Booking は説明用の小さな予約である。
type Booking struct {
	slot domain.TimeSlot
}

// New は予約を作る。
func New(slot domain.TimeSlot) *Booking {
	return &Booking{slot: slot}
}

// #region cancel

// Cancel は現在時刻を引数で受け取る。
func (b *Booking) Cancel(now time.Time) error {
	if now.After(b.slot.Start().Add(-24 * time.Hour)) {
		return domain.ErrCancelTooLate
	}
	return nil
}

// #endregion

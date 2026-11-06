// Package nowcancel は、メソッドの中で time.Now を呼ぶ例である。
package nowcancel

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

// Cancel は現在時刻を自分で取得する。
func (b *Booking) Cancel() error {
	deadline := b.slot.Start().Add(-24 * time.Hour)
	if time.Now().After(deadline) {
		return domain.ErrCancelTooLate
	}
	return nil
}

// #endregion

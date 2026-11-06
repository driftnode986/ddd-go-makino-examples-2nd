// Package valuereceiver は、状態を変えるメソッドを値レシーバで書いた例である。
package valuereceiver

import "example.com/trainingbooking/internal/booking/domain"

// #region confirm

// Booking は説明用の小さな予約である。
type Booking struct {
	status domain.Status
}

// Confirm は値レシーバなので、呼び出し元のコピーを変える。
func (b Booking) Confirm() error {
	if b.status != domain.StatusPending {
		return domain.ErrInvalidTransition
	}
	b.status = domain.StatusConfirmed
	return nil
}

// #endregion

// New は保留の予約を作る。
func New() Booking { return Booking{status: domain.StatusPending} }

// Status は状態を返す。
func (b Booking) Status() domain.Status { return b.status }

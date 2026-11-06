package domain

import "errors"

// Booking の操作が返すエラーである。
var (
	ErrInvalidBooking    = errors.New("invalid booking")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrStartInPast       = errors.New("start is in the past")
	ErrCancelTooLate     = errors.New("too late to cancel")
	ErrInvalidCoupon     = errors.New("invalid coupon")
)

// #region overlap-error

// ErrOverlap は同じ会員の時間帯が重なることを表す。
var ErrOverlap = errors.New("booking time slot overlaps")

// #endregion

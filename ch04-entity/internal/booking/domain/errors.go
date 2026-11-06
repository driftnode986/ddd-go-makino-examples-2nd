package domain

import "errors"

// Booking の操作が返すエラーである。
var (
	ErrInvalidBooking    = errors.New("invalid booking")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrStartInPast       = errors.New("start is in the past")
	ErrCancelTooLate     = errors.New("too late to cancel")
)

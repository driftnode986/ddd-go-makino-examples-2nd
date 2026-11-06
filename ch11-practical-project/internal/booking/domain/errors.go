package domain

import "errors"

// リポジトリが返すエラーである。実装の違いをここで吸収する。
var (
	ErrNotFound        = errors.New("booking not found")
	ErrDuplicate       = errors.New("booking already exists")
	ErrVersionConflict = errors.New("booking version conflict")
)

// 集約の操作が返すエラーである。
var (
	ErrInvalidBooking    = errors.New("invalid booking")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrStartInPast       = errors.New("start is in the past")
	ErrCancelTooLate     = errors.New("too late to cancel")
	ErrChangeTooLate     = errors.New("too late to change options")
	ErrInvalidOption     = errors.New("invalid booking option")
	ErrOptionLimit       = errors.New("too many booking options")
	ErrDuplicateOption   = errors.New("duplicate booking option")
	ErrDiscountTooLarge  = errors.New("discount exceeds subtotal")
	ErrInvalidCoupon     = errors.New("invalid coupon")
)

// ErrOverlap は同じ会員の時間帯が重なることを表す。
var ErrOverlap = errors.New("booking time slot overlaps")

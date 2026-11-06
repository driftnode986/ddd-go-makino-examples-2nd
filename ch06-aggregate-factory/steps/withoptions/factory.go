// Package withoptions は、NewBooking を Functional Options の形で
// 包む例である。
package withoptions

import (
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// #region options

// Option は予約を作るときの任意の指定である。
type Option func(*domain.Booking, domain.Clock) error

// WithOption は明細を足す。
func WithOption(o domain.BookingOption) Option {
	return func(b *domain.Booking, clk domain.Clock) error {
		return b.AddOption(o, clk)
	}
}

// WithDiscount は割引額を適用する。
func WithDiscount(m domain.Money) Option {
	return func(b *domain.Booking, _ domain.Clock) error {
		return b.ApplyDiscount(m)
	}
}

// #endregion

// New は予約を作り、指定を順に適用する。
func New(
	id ids.BookingID,
	member ids.MemberID,
	lesson ids.LessonID,
	slot domain.TimeSlot,
	fee domain.Money,
	clk domain.Clock,
	opts ...Option,
) (*domain.Booking, error) {
	b, err := domain.NewBooking(id, member, lesson, slot, fee, clk)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if err := opt(b, clk); err != nil {
			return nil, err
		}
	}
	return b, nil
}

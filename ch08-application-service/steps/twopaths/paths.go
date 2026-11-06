// Package twopaths は同じ手順を2か所に書いた例である。
package twopaths

import "example.com/trainingbooking/internal/booking/domain"

// #region paths

// FromHTTP はハンドラに書いた手順である。
func FromHTTP(
	b *domain.Booking, o domain.BookingOption,
	c domain.Coupon, clk domain.Clock,
) error {
	if err := b.AddOption(o, clk); err != nil {
		return err
	}
	d, err := domain.CalculateDiscount(b.Total(), c)
	if err != nil {
		return err
	}
	return b.ApplyDiscount(d)
}

// FromCSV はバッチに書き写した手順である。割引を先に計算している。
func FromCSV(
	b *domain.Booking, o domain.BookingOption,
	c domain.Coupon, clk domain.Clock,
) error {
	d, err := domain.CalculateDiscount(b.Total(), c)
	if err != nil {
		return err
	}
	if err := b.ApplyDiscount(d); err != nil {
		return err
	}
	return b.AddOption(o, clk)
}

// #endregion

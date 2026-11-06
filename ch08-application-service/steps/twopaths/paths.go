// Package twopaths は同じ手順を2か所に書いた例である。
package twopaths

import "example.com/trainingbooking/internal/booking/domain"

// Row は CSV の1行である。
type Row struct {
	Name      string
	Quantity  int
	UnitPrice domain.Money
}

// #region paths

// FromHTTP はハンドラの手順である。単価はレッスンの表から取る。
func FromHTTP(
	b *domain.Booking, name string, qty int,
	prices map[string]domain.Money, clk domain.Clock,
) error {
	o, err := domain.NewBookingOption(name, prices[name], qty)
	if err != nil {
		return err
	}
	return b.AddOption(o, clk)
}

// FromCSV はバッチに写した手順である。単価は行の値を使っている。
func FromCSV(b *domain.Booking, r Row, clk domain.Clock) error {
	o, err := domain.NewBookingOption(
		r.Name, r.UnitPrice, r.Quantity)
	if err != nil {
		return err
	}
	return b.AddOption(o, clk)
}

// #endregion

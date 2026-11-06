//go:build integration

// Package noversion は版を比べない保存の失敗を見せる。
package noversion

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

// #region save

// Save は ID だけで行を選んで更新する。版を比べない。
func Save(ctx context.Context, b *domain.Booking) error {
	s := b.State()
	tx, _ := pgtx.From(ctx)
	_, err := tx.Exec(ctx, `UPDATE bookings
		SET status = $2, discount_yen = $3 WHERE id = $1`,
		s.ID.String(), string(s.Status), s.DiscountYen)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`DELETE FROM booking_options WHERE booking_id = $1`,
		s.ID.String())
	if err != nil {
		return err
	}
	for i, o := range s.Options {
		_, err = tx.Exec(ctx, `INSERT INTO booking_options
			(booking_id, position, name, unit_price_yen, quantity)
			VALUES ($1, $2, $3, $4, $5)`,
			s.ID.String(), i, o.Name, o.UnitPriceYen, o.Quantity)
		if err != nil {
			return err
		}
	}
	return nil
}

// #endregion

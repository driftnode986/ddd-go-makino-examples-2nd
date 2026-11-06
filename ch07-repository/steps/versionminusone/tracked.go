//go:build integration

// Package versionminusone はメソッドで版を増やす形の失敗を見せる。
package versionminusone

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

// #region tracked

// Tracked は状態を変えるたびに版を増やす予約である。
type Tracked struct {
	*domain.Booking
	Version int
}

// Confirm は確定して版を増やす。
func (t *Tracked) Confirm() error {
	if err := t.Booking.Confirm(); err != nil {
		return err
	}
	t.Version++
	return nil
}

// Cancel はキャンセルして版を増やす。
func (t *Tracked) Cancel(clk domain.Clock) error {
	if err := t.Booking.Cancel(clk); err != nil {
		return err
	}
	t.Version++
	return nil
}

// #endregion

// #region save

// Save は「DB の版は、手元の版の1つ前のはず」と見て更新する。
func Save(ctx context.Context, t *Tracked) error {
	s := t.State()
	tx, _ := pgtx.From(ctx)
	tag, err := tx.Exec(ctx, `UPDATE bookings
		SET status = $3, version = $4
		WHERE id = $1 AND version = $2`,
		s.ID.String(), t.Version-1, string(s.Status), t.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

// #endregion

//go:build integration

// Package autotx は Tx が無ければ自分で開くリポジトリの失敗を見せる。
package autotx

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

// #region repository

// Repository は ctx に Tx が無ければ自分で Tx を開く。
type Repository struct {
	*postgres.BookingRepository
	Tx *pgtx.Transactor
}

// Add は ctx に Tx が無くても保存できる。
func (r Repository) Add(
	ctx context.Context, b *domain.Booking,
) error {
	return r.Tx.Do(ctx, func(ctx context.Context) error {
		return r.BookingRepository.Add(ctx, b)
	})
}

// #endregion

// Package txarg は Tx を引数で渡すインターフェースの例である。
package txarg

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"github.com/jackc/pgx/v5"
)

// #region arg

// BookingRepository は pgx の Tx を引数で受け取る。
type BookingRepository interface {
	Add(ctx context.Context, tx pgx.Tx, b *domain.Booking) error
	Save(ctx context.Context, tx pgx.Tx, b *domain.Booking) error
	FindByID(
		ctx context.Context, tx pgx.Tx, id ids.BookingID,
	) (*domain.Booking, error)
}

// #endregion

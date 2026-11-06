//go:build integration

// Package updatefn は読み書きと Tx をリポジトリに任せる代案である。
package updatefn

import (
	"context"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// Repository は読んで fn を呼び、保存するまでを1つの Tx で行う。
type Repository struct {
	Bookings domain.BookingRepository
	Outbox   application.OutboxWriter
	Tx       application.Transactor
}

// #region update

// Update は予約を読み、fn で変更し、保存とイベントを書く。
func (r Repository) Update(
	ctx context.Context, id ids.BookingID,
	fn func(b *domain.Booking) error,
) error {
	return r.Tx.Do(ctx, func(ctx context.Context) error {
		b, err := r.Bookings.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := fn(b); err != nil {
			return err
		}
		if err := r.Bookings.Save(ctx, b); err != nil {
			return err
		}
		return r.Outbox.Append(ctx, b.PullEvents()...)
	})
}

// #endregion

// #region cancel

// Cancel は Tx も ctx の受け渡しも書かない。
func Cancel(
	ctx context.Context, r Repository,
	id ids.BookingID, clk domain.Clock,
) error {
	return r.Update(ctx, id, func(b *domain.Booking) error {
		return b.Cancel(clk)
	})
}

// #endregion

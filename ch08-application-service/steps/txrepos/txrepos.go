//go:build integration

// Package txrepos は Tx に結び付いた部品をコールバックに渡す代案である。
package txrepos

import (
	"context"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// #region stores

// Stores は1つの Tx に結び付いた読み書きの部品である。
type Stores struct {
	Bookings domain.BookingRepository
	Outbox   application.OutboxWriter
}

// Runner は Tx を開いて、結び付いた部品を fn に渡す。
type Runner interface {
	Do(ctx context.Context, fn func(s Stores) error) error
}

// #endregion

// #region cancel

// Cancel はコールバックの外に部品が無いので、Tx の外で読めない。
func Cancel(
	ctx context.Context, run Runner,
	id ids.BookingID, clk domain.Clock,
) error {
	return run.Do(ctx, func(s Stores) error {
		b, err := s.Bookings.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := b.Cancel(clk); err != nil {
			return err
		}
		if err := s.Bookings.Save(ctx, b); err != nil {
			return err
		}
		return s.Outbox.Append(ctx, b.PullEvents()...)
	})
}

// #endregion

// PgRunner は Transactor の Tx に部品を結び付ける。
type PgRunner struct {
	Bookings domain.BookingRepository
	Outbox   application.OutboxWriter
	Tx       application.Transactor
}

// Do は Tx を開き、渡された ctx を無視して Tx で動く部品を fn に渡す。
func (r PgRunner) Do(
	ctx context.Context, fn func(s Stores) error,
) error {
	return r.Tx.Do(ctx, func(txCtx context.Context) error {
		return fn(Stores{
			Bookings: boundRepo{r.Bookings, txCtx},
			Outbox:   boundOutbox{r.Outbox, txCtx},
		})
	})
}

type boundRepo struct {
	domain.BookingRepository
	txCtx context.Context
}

func (r boundRepo) FindByID(
	_ context.Context, id ids.BookingID,
) (*domain.Booking, error) {
	return r.BookingRepository.FindByID(r.txCtx, id)
}

func (r boundRepo) Add(_ context.Context, b *domain.Booking) error {
	return r.BookingRepository.Add(r.txCtx, b)
}

func (r boundRepo) Save(_ context.Context, b *domain.Booking) error {
	return r.BookingRepository.Save(r.txCtx, b)
}

type boundOutbox struct {
	application.OutboxWriter
	txCtx context.Context
}

func (o boundOutbox) Append(
	_ context.Context, events ...domain.DomainEvent,
) error {
	return o.OutboxWriter.Append(o.txCtx, events...)
}

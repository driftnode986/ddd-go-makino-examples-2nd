//go:build integration

// Package twoselects は親と明細を別の SQL で読む失敗を見せる。
package twoselects

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"github.com/jackc/pgx/v5/pgxpool"
)

// #region find

// Find は親と明細を2回の SELECT で読む。between は2回の間に呼ぶ。
func Find(
	ctx context.Context, pool *pgxpool.Pool,
	id ids.BookingID, member ids.MemberID, lesson ids.LessonID,
	between func(),
) (*domain.Booking, error) {
	s := domain.BookingState{
		ID: id, MemberID: member, LessonID: lesson,
	}
	err := pool.QueryRow(ctx, `SELECT starts_at, ends_at, fee_yen,
		discount_yen, status, version FROM bookings WHERE id = $1`,
		id.String()).Scan(&s.Start, &s.End, &s.FeeYen,
		&s.DiscountYen, &s.Status, &s.Version)
	if err != nil {
		return nil, err
	}
	between()
	rows, err := pool.Query(ctx, `SELECT
		name, unit_price_yen, quantity
		FROM booking_options WHERE booking_id = $1`, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return withOptions(s, rows)
}

// #endregion

type optionRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func withOptions(
	s domain.BookingState, rows optionRows,
) (*domain.Booking, error) {
	for rows.Next() {
		var o domain.OptionState
		err := rows.Scan(&o.Name, &o.UnitPriceYen, &o.Quantity)
		if err != nil {
			return nil, err
		}
		s.Options = append(s.Options, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return domain.ReconstructBooking(s)
}

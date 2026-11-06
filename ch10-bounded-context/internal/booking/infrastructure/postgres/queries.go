package postgres

import (
	"context"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/ids"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BookingQueries は予約の読み取りである。集約を組み立てない。
type BookingQueries struct {
	pool *pgxpool.Pool
}

// NewBookingQueries は BookingQueries を返す。
func NewBookingQueries(pool *pgxpool.Pool) *BookingQueries {
	return &BookingQueries{pool: pool}
}

// ListByMember は会員の予約を開始の順に返す。
func (q *BookingQueries) ListByMember(
	ctx context.Context, member ids.MemberID,
) ([]application.BookingView, error) {
	rows, err := queriesFor(ctx, q.pool).ListBookingsByMember(
		ctx, member.String())
	if err != nil {
		return nil, err
	}
	views := make([]application.BookingView, len(rows))
	for i, r := range rows {
		views[i] = application.BookingView{
			ID:       r.ID,
			LessonID: r.LessonID,
			Start:    r.StartsAt,
			End:      r.EndsAt,
			Status:   r.Status,
			TotalYen: r.TotalYen,
		}
	}
	return views, nil
}

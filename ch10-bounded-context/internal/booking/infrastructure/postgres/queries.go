package postgres

import (
	"context"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
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
	views := make([]application.BookingView, 0, len(rows))
	for _, r := range rows {
		if n := len(views); n == 0 || views[n-1].ID != r.ID {
			views = append(views, application.BookingView{
				ID:       r.ID,
				LessonID: r.LessonID,
				Start:    r.StartsAt,
				End:      r.EndsAt,
				Status:   r.Status,
				TotalYen: r.TotalYen,
				Options:  []application.BookingOptionView{},
			})
		}
		if r.OptionName == nil {
			continue
		}
		last := &views[len(views)-1]
		last.Options = append(last.Options,
			application.BookingOptionView{
				Name:        *r.OptionName,
				Quantity:    int(*r.OptionQuantity),
				SubtotalYen: r.OptionSubtotalYen,
			})
	}
	return views, nil
}

// ActiveSlotsByMember は有効な予約の時間帯を開始の順に返す。
func (q *BookingQueries) ActiveSlotsByMember(
	ctx context.Context, member ids.MemberID,
) ([]domain.TimeSlot, error) {
	rows, err := queriesFor(ctx, q.pool).ListActiveSlotsByMember(
		ctx, member.String())
	if err != nil {
		return nil, err
	}
	slots := make([]domain.TimeSlot, len(rows))
	for i, r := range rows {
		slots[i], err = domain.ReconstructTimeSlot(
			r.StartsAt, r.EndsAt)
		if err != nil {
			return nil, err
		}
	}
	return slots, nil
}

package memory

import (
	"cmp"
	"context"
	"slices"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// BookingQueries は BookingRepository の中身を読む。
type BookingQueries struct {
	repo *BookingRepository
}

// NewBookingQueries は BookingQueries を返す。
func NewBookingQueries(repo *BookingRepository) *BookingQueries {
	return &BookingQueries{repo: repo}
}

// ListByMember は会員の予約を開始の順に返す。
func (q *BookingQueries) ListByMember(
	ctx context.Context, member ids.MemberID,
) ([]application.BookingView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q.repo.mu.RLock()
	defer q.repo.mu.RUnlock()
	views := make([]application.BookingView, 0)
	for _, s := range q.repo.states {
		if s.MemberID != member {
			continue
		}
		b, err := domain.ReconstructBooking(s)
		if err != nil {
			return nil, err
		}
		views = append(views, application.BookingView{
			ID:       s.ID.String(),
			LessonID: s.LessonID.String(),
			Start:    s.Start,
			End:      s.End,
			Status:   string(s.Status),
			TotalYen: b.Total().Amount(),
		})
	}
	slices.SortFunc(views, func(a, b application.BookingView) int {
		return cmp.Or(
			a.Start.Compare(b.Start), cmp.Compare(a.ID, b.ID))
	})
	return views, nil
}

// ActiveSlotsByMember は有効な予約の時間帯を開始の順に返す。
func (q *BookingQueries) ActiveSlotsByMember(
	ctx context.Context, member ids.MemberID,
) ([]domain.TimeSlot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q.repo.mu.RLock()
	defer q.repo.mu.RUnlock()
	slots := make([]domain.TimeSlot, 0)
	for _, s := range q.repo.states {
		if s.MemberID != member || !occupies(s.Status) {
			continue
		}
		slot, err := domain.ReconstructTimeSlot(s.Start, s.End)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	slices.SortFunc(slots, func(a, b domain.TimeSlot) int {
		return a.Start().Compare(b.Start())
	})
	return slots, nil
}

package application

import (
	"context"
	"errors"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
)

// ReserveSeatInput は席の確保の入力である。
type ReserveSeatInput struct {
	LessonID  ids.LessonID
	BookingID ids.BookingID
}

// ReserveSeat は予約の席を確保する。
type ReserveSeat struct {
	repo   domain.LessonRepository
	outbox OutboxWriter
	tx     Transactor
	clock  domain.Clock
}

// NewReserveSeat は ReserveSeat を返す。
func NewReserveSeat(
	repo domain.LessonRepository,
	outbox OutboxWriter,
	tx Transactor,
	clock domain.Clock,
) *ReserveSeat {
	return &ReserveSeat{repo, outbox, tx, clock}
}

// Execute は席を確保し、結果のイベントを同じ Tx で書く。
//
// 満席のときは ErrLessonFull、レッスンが無いときは ErrNotFound を
// 返す。どちらも拒否のイベントはコミットされている。版の競合は
// 読み直して判定し直し、満席に変えない。Tx の中から呼ばない。
func (u *ReserveSeat) Execute(
	ctx context.Context, in ReserveSeatInput,
) error {
	var outcome error
	err := retryOnConflict(ctx, func() error {
		return u.tx.Do(ctx, func(ctx context.Context) error {
			var err error
			outcome, err = u.reserve(ctx, in)
			return err
		})
	})
	if err != nil {
		return err
	}
	return outcome
}

// reserve は1回の試行である。1つ目の戻り値は拒否の理由で、
// Tx を戻さない。2つ目の戻り値は Tx を戻すエラーである。
func (u *ReserveSeat) reserve(
	ctx context.Context, in ReserveSeatInput,
) (rejected, err error) {
	l, err := u.repo.FindByID(ctx, in.LessonID)
	if errors.Is(err, domain.ErrNotFound) {
		event := domain.SeatRejected{
			LessonID:   in.LessonID,
			BookingID:  in.BookingID,
			Reason:     domain.RejectLessonNotFound,
			OccurredAt: u.clock.Now(),
		}
		return err, u.outbox.Append(ctx, event)
	}
	if err != nil {
		return nil, err
	}
	err = l.ReserveSeat(in.BookingID, u.clock)
	if errors.Is(err, domain.ErrLessonFull) {
		return err, u.outbox.Append(ctx, l.PullEvents()...)
	}
	if err != nil {
		return nil, err
	}
	if err := u.repo.Save(ctx, l); err != nil {
		return nil, err
	}
	return nil, u.outbox.Append(ctx, l.PullEvents()...)
}

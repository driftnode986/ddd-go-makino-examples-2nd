package application

import (
	"context"
	"errors"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
)

// ReserveSeatInput は席の確保の入力である。
type ReserveSeatInput struct {
	EventID   string
	LessonID  ids.LessonID
	BookingID ids.BookingID
}

// ReserveSeat は予約の席を確保する。
type ReserveSeat struct {
	repo      domain.LessonRepository
	outbox    OutboxWriter
	processed ProcessedEvents
	tx        Transactor
	clock     domain.Clock
}

// NewReserveSeat は ReserveSeat を返す。
func NewReserveSeat(
	repo domain.LessonRepository,
	outbox OutboxWriter,
	processed ProcessedEvents,
	tx Transactor,
	clock domain.Clock,
) *ReserveSeat {
	return &ReserveSeat{repo, outbox, processed, tx, clock}
}

// #region execute

// Execute は処理済みの記録・席の保存・結果のイベントを同じ Tx で
// 書く。rejected が true なら、拒否のイベントをコミットした。
// err は Tx を戻したことだけを表すので、配信処理は再配信する。
// 版の競合は読み直して判定し直し、満席に変えない。
func (u *ReserveSeat) Execute(
	ctx context.Context, in ReserveSeatInput,
) (rejected bool, err error) {
	err = retryOnConflict(ctx, func() error {
		return u.tx.Do(ctx, func(ctx context.Context) error {
			first, err := u.processed.MarkProcessed(
				ctx, Subscriber, in.EventID)
			if err != nil {
				return err
			}
			rejected = false
			if !first {
				return nil
			}
			rejected, err = u.reserve(ctx, in)
			return err
		})
	})
	return rejected && err == nil, err
}

// #endregion

// #region reserve

// reserve は1回の試行である。満席とレッスンが無いことは rejected で
// 返し、Tx を戻さない。err は Tx を戻すエラーである。
func (u *ReserveSeat) reserve(
	ctx context.Context, in ReserveSeatInput,
) (rejected bool, err error) {
	l, err := u.repo.FindByID(ctx, in.LessonID)
	if errors.Is(err, domain.ErrNotFound) {
		event := domain.SeatRejected{
			LessonID:   in.LessonID,
			BookingID:  in.BookingID,
			Reason:     domain.RejectLessonNotFound,
			OccurredAt: u.clock.Now(),
		}
		return true, u.outbox.Append(ctx, event)
	}
	if err != nil {
		return false, err
	}
	err = l.ReserveSeat(in.BookingID, u.clock)
	if errors.Is(err, domain.ErrLessonFull) {
		return true, u.outbox.Append(ctx, l.PullEvents()...)
	}
	if err != nil {
		return false, err
	}
	if err := u.repo.Save(ctx, l); err != nil {
		return false, err
	}
	return false, u.outbox.Append(ctx, l.PullEvents()...)
}

// #endregion

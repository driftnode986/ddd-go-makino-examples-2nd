package application

import (
	"context"
	"errors"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
)

// ReleaseSeatInput は席を戻す入力である。
type ReleaseSeatInput struct {
	EventID   string
	LessonID  ids.LessonID
	BookingID ids.BookingID
}

// ReleaseSeat はキャンセルされた予約の席を戻す。
type ReleaseSeat struct {
	repo      domain.LessonRepository
	processed ProcessedEvents
	tx        Transactor
}

// NewReleaseSeat は ReleaseSeat を返す。
func NewReleaseSeat(
	repo domain.LessonRepository,
	processed ProcessedEvents,
	tx Transactor,
) *ReleaseSeat {
	return &ReleaseSeat{repo, processed, tx}
}

// Execute は処理済みの記録と席の保存を同じ Tx で書く。処理済みの
// イベントなら何もしない。レッスンが無ければ戻す席も無いので成功に
// する。版の競合は読み直して判定し直す。Tx の中から呼ばない。
func (u *ReleaseSeat) Execute(
	ctx context.Context, in ReleaseSeatInput,
) error {
	return retryOnConflict(ctx, func() error {
		return u.tx.Do(ctx, func(ctx context.Context) error {
			return u.release(ctx, in)
		})
	})
}

func (u *ReleaseSeat) release(
	ctx context.Context, in ReleaseSeatInput,
) error {
	first, err := u.processed.MarkProcessed(
		ctx, Subscriber, in.EventID)
	if err != nil || !first {
		return err
	}
	l, err := u.repo.FindByID(ctx, in.LessonID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := l.ReleaseSeat(in.BookingID); err != nil {
		return err
	}
	return u.repo.Save(ctx, l)
}

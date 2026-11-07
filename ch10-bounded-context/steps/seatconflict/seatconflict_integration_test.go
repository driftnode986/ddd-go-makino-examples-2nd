//go:build integration

// Package seatconflict は版の競合を満席と取り違えると、席が残っている
// のに予約を拒否することを確かめる。
package seatconflict_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/catalog/application"
	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/catalog/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC),
}

func newLesson(t *testing.T, capacity int) *domain.Lesson {
	t.Helper()
	l, err := domain.NewLesson(domain.LessonSpec{
		ID:       ids.NewID[ids.LessonMarker](),
		Title:    "朝のヨガ",
		Start:    clock.now.AddDate(0, 1, 0),
		Duration: time.Hour,
		FeeYen:   3000,
		Capacity: capacity,
	})
	if err != nil {
		t.Fatalf("NewLesson: %v", err)
	}
	return l
}

func addLesson(
	t *testing.T, repo domain.LessonRepository, tx *pgtx.Transactor,
	capacity int,
) *domain.Lesson {
	t.Helper()
	l := newLesson(t, capacity)
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		return repo.Add(ctx, l)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return l
}

func find(
	t *testing.T, repo domain.LessonRepository, id ids.LessonID,
) *domain.Lesson {
	t.Helper()
	l, err := repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	return l
}

// 残席2のレッスンを2つの処理が同じ版で読む。競合した処理が満席の
// 拒否を記録すると、席が残っているのに SeatRejected がコミットされる。
func TestConflictTreatedAsFull(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations, outbox.Migrations)
	repo := postgres.NewLessonRepository(pool)
	writer := postgres.NewOutboxWriter(outbox.NewStore(pool))
	tx := pgtx.NewTransactor(pool)
	ctx := t.Context()
	lesson := addLesson(t, repo, tx, 2)
	a := find(t, repo, lesson.ID())
	b := find(t, repo, lesson.ID())
	reserve := func(l *domain.Lesson) string {
		booking := ids.NewID[ids.BookingMarker]()
		if err := l.ReserveSeat(booking, clock); err != nil {
			t.Fatalf("ReserveSeat: %v", err)
		}
		err := tx.Do(ctx, func(ctx context.Context) error {
			if err := repo.Save(ctx, l); err != nil {
				return err
			}
			return writer.Append(ctx, l.PullEvents()...)
		})
		if err == nil {
			return "確保"
		}
		if !errors.Is(err, domain.ErrVersionConflict) {
			t.Fatalf("Save: %v", err)
		}
		err = tx.Do(ctx, func(ctx context.Context) error {
			return writer.Append(ctx, domain.SeatRejected{
				LessonID:   l.ID(),
				BookingID:  booking,
				Reason:     domain.RejectLessonFull,
				OccurredAt: clock.now,
			})
		})
		if err != nil {
			t.Fatalf("Append: %v", err)
		}
		return "満席として拒否"
	}
	t.Logf("処理 A: %s", reserve(a))
	t.Logf("処理 B: %s", reserve(b))
	seats := find(t, repo, lesson.ID()).ReservedSeats()
	var rejected int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox
		WHERE event_type = 'seat.rejected'`).Scan(&rejected)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if seats != 1 || rejected != 1 {
		t.Fatalf("seats = %d, rejected = %d", seats, rejected)
	}
	t.Logf("定員 2 のレッスン: 席 %d, SeatRejected %d 件", seats, rejected)
}

// interleaved は最初の読み込みの直後に、別の処理を割り込ませる。
// 順序を作るので、毎回必ず版の競合になる。
type interleaved struct {
	domain.LessonRepository
	other     func() error
	conflicts int
}

func (r *interleaved) FindByID(
	ctx context.Context, id ids.LessonID,
) (*domain.Lesson, error) {
	l, err := r.LessonRepository.FindByID(ctx, id)
	if other := r.other; other != nil {
		r.other = nil
		if err := other(); err != nil {
			return nil, err
		}
	}
	return l, err
}

func (r *interleaved) Save(
	ctx context.Context, l *domain.Lesson,
) error {
	err := r.LessonRepository.Save(ctx, l)
	if errors.Is(err, domain.ErrVersionConflict) {
		r.conflicts++
	}
	return err
}

type discardOutbox struct{}

func (discardOutbox) Append(
	context.Context, ...domain.DomainEvent,
) error {
	return nil
}

// 本書の ReserveSeat は、同じ順序で競合しても読み直して確保する。
func TestConflictIsRetried(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations, outbox.Migrations)
	inner := postgres.NewLessonRepository(pool)
	tx := pgtx.NewTransactor(pool)
	ctx := t.Context()
	lesson := addLesson(t, inner, tx, 2)
	processed := outbox.NewProcessedEvents(pool)
	input := func() application.ReserveSeatInput {
		return application.ReserveSeatInput{
			EventID:   outbox.NewEventID(),
			LessonID:  lesson.ID(),
			BookingID: ids.NewID[ids.BookingMarker](),
		}
	}
	other := application.NewReserveSeat(
		inner, discardOutbox{}, processed, tx, clock)
	repo := &interleaved{
		LessonRepository: inner,
		other: func() error {
			_, err := other.Execute(context.Background(), input())
			return err
		},
	}
	reserve := application.NewReserveSeat(
		repo, discardOutbox{}, processed, tx, clock)
	if _, err := reserve.Execute(ctx, input()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repo.conflicts != 1 {
		t.Fatalf("版の競合 = %d 回", repo.conflicts)
	}
	got, err := inner.FindByID(ctx, lesson.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ReservedSeats() != 2 {
		t.Fatalf("席 = %d", got.ReservedSeats())
	}
	t.Logf("版の競合 %d 回, 定員 2 のレッスンの席: %d",
		repo.conflicts, got.ReservedSeats())
}

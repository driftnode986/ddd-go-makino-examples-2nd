package application_test

import (
	"context"
	"testing"
	"time"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
)

var (
	jst   = time.FixedZone("JST", 9*60*60)
	now   = time.Date(2030, time.May, 1, 0, 0, 0, 0, jst)
	start = time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func newLesson(t *testing.T, capacity int) *domain.Lesson {
	t.Helper()
	l, err := domain.NewLesson(domain.LessonSpec{
		ID:       ids.NewID[ids.LessonMarker](),
		Title:    "朝のヨガ",
		Start:    start,
		Duration: time.Hour,
		FeeYen:   3000,
		Capacity: capacity,
		Options: []domain.LessonOption{
			{Name: "シューズ", UnitPriceYen: 300},
		},
	})
	if err != nil {
		t.Fatalf("NewLesson: %v", err)
	}
	return l
}

type transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

func addLesson(
	t *testing.T,
	repo domain.LessonRepository,
	tx transactor,
	capacity int,
) ids.LessonID {
	t.Helper()
	l := newLesson(t, capacity)
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		return repo.Add(ctx, l)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return l.ID()
}

func seats(
	t *testing.T, repo domain.LessonRepository, id ids.LessonID,
) int {
	t.Helper()
	l, err := repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	return l.ReservedSeats()
}

// discardOutbox は何もしない OutboxWriter である。
type discardOutbox struct{}

func (discardOutbox) Append(
	context.Context, ...domain.DomainEvent,
) error {
	return nil
}

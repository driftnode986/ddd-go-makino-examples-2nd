package application

import (
	"context"
	"errors"
	"fmt"

	"example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
)

// LessonReader は contract.LessonReader の実装である。
// 集約を外へ出さず、contract の型に変えて返す。
type LessonReader struct {
	repo domain.LessonRepository
}

// NewLessonReader は LessonReader を返す。
func NewLessonReader(repo domain.LessonRepository) *LessonReader {
	return &LessonReader{repo: repo}
}

// FindLesson はレッスンを返す。無ければ contract.ErrLessonNotFound
// を返す。
func (r *LessonReader) FindLesson(
	ctx context.Context, id ids.LessonID,
) (contract.Lesson, error) {
	l, err := r.repo.FindByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return contract.Lesson{}, fmt.Errorf(
			"%w: %s", contract.ErrLessonNotFound, id)
	}
	if err != nil {
		return contract.Lesson{}, err
	}
	s := l.State()
	options := make([]contract.LessonOption, len(s.Options))
	for i, o := range s.Options {
		options[i] = contract.LessonOption{
			Name:         o.Name,
			UnitPriceYen: o.UnitPriceYen,
		}
	}
	return contract.Lesson{
		ID:       s.ID,
		Title:    s.Title,
		Start:    s.Start,
		Duration: s.Duration,
		FeeYen:   s.FeeYen,
		Options:  options,
	}, nil
}

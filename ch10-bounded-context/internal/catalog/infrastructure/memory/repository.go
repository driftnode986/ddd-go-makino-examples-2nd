// Package memory は Catalog の in-memory 実装である。
// テストダブルとして使う。
package memory

import (
	"context"
	"fmt"
	"sync"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

// LessonRepository は集約の中身をコピーで保持する。
type LessonRepository struct {
	mu     sync.RWMutex
	states map[ids.LessonID]domain.LessonState
}

// NewLessonRepository は空のリポジトリを返す。
func NewLessonRepository() *LessonRepository {
	return &LessonRepository{
		states: make(map[ids.LessonID]domain.LessonState),
	}
}

// Add は新しいレッスンを ctx の Tx の中で保存する。
func (r *LessonRepository) Add(
	ctx context.Context, l *domain.Lesson,
) error {
	state, cur, err := prepare(ctx, l)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.states[state.ID]; ok {
		return fmt.Errorf("%w: %s", domain.ErrDuplicate, state.ID)
	}
	r.states[state.ID] = state
	cur.OnRollback(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.states, state.ID)
	})
	return nil
}

// Save は読み込んだときの版で判定し、ctx の Tx の中で更新する。
func (r *LessonRepository) Save(
	ctx context.Context, l *domain.Lesson,
) error {
	state, cur, err := prepare(ctx, l)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.states[state.ID]
	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, state.ID)
	}
	if stored.Version != state.Version {
		return fmt.Errorf(
			"%w: %s", domain.ErrVersionConflict, state.ID)
	}
	state.Version++
	r.states[state.ID] = state
	cur.OnRollback(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.states[state.ID] = stored
	})
	return nil
}

// FindByID は保存された中身から集約を組み立てる。
func (r *LessonRepository) FindByID(
	ctx context.Context, id ids.LessonID,
) (*domain.Lesson, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.states[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	return domain.ReconstructLesson(state)
}

func prepare(
	ctx context.Context, l *domain.Lesson,
) (domain.LessonState, *memtx.Tx, error) {
	if err := ctx.Err(); err != nil {
		return domain.LessonState{}, nil, err
	}
	state := l.State()
	if _, err := domain.ReconstructLesson(state); err != nil {
		return domain.LessonState{}, nil, err
	}
	cur, ok := memtx.From(ctx)
	if !ok {
		return domain.LessonState{}, nil, txn.ErrNoTransaction
	}
	return state, cur, nil
}

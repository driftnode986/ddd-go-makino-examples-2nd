// Package memory は BookingRepository の in-memory 実装である。
// テストダブルとして使う。
package memory

import (
	"context"
	"fmt"
	"sync"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

// #region type

// BookingRepository は集約の中身をコピーで保持する。
// 呼び出し側とポインタを共有しない。
type BookingRepository struct {
	mu     sync.RWMutex
	states map[ids.BookingID]domain.BookingState
}

// #endregion

// NewBookingRepository は空のリポジトリを返す。
func NewBookingRepository() *BookingRepository {
	return &BookingRepository{
		states: make(map[ids.BookingID]domain.BookingState),
	}
}

// Add は新しい予約を ctx の Tx の中で保存する。
func (r *BookingRepository) Add(
	ctx context.Context, b *domain.Booking,
) error {
	state, cur, err := prepare(ctx, b)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.states[state.ID]; ok {
		return fmt.Errorf("%w: %s", domain.ErrDuplicate, state.ID)
	}
	if err := r.checkOverlap(state); err != nil {
		return err
	}
	r.states[state.ID] = state
	cur.OnRollback(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.states, state.ID)
	})
	return nil
}

// #region save

// Save は読み込んだときの版で判定し、ctx の Tx の中で更新する。
func (r *BookingRepository) Save(
	ctx context.Context, b *domain.Booking,
) error {
	state, cur, err := prepare(ctx, b)
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
	if err := r.checkOverlap(state); err != nil {
		return err
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

// #endregion

// FindByID は保存された中身から集約を組み立てる。
func (r *BookingRepository) FindByID(
	ctx context.Context, id ids.BookingID,
) (*domain.Booking, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.states[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	return domain.ReconstructBooking(state)
}

// prepare は保存できる集約かと ctx の Tx を確かめて、
// 中身のコピーと Tx を返す。
func prepare(
	ctx context.Context, b *domain.Booking,
) (domain.BookingState, *memtx.Tx, error) {
	if err := ctx.Err(); err != nil {
		return domain.BookingState{}, nil, err
	}
	state := b.State()
	if _, err := domain.ReconstructBooking(state); err != nil {
		return domain.BookingState{}, nil, err
	}
	cur, ok := memtx.From(ctx)
	if !ok {
		return domain.BookingState{}, nil, txn.ErrNoTransaction
	}
	return state, cur, nil
}

// checkOverlap は PostgreSQL の排他制約と同じ判定を行う。
func (r *BookingRepository) checkOverlap(
	s domain.BookingState,
) error {
	if !occupies(s.Status) {
		return nil
	}
	for id, other := range r.states {
		if id == s.ID || other.MemberID != s.MemberID ||
			!occupies(other.Status) {
			continue
		}
		if s.Start.Before(other.End) && other.Start.Before(s.End) {
			return fmt.Errorf("%w: %s", domain.ErrOverlap, other.ID)
		}
	}
	return nil
}

// occupies は時間帯を占める状態のとき true を返す。
func occupies(s domain.Status) bool {
	return s == domain.StatusPending || s == domain.StatusConfirmed
}

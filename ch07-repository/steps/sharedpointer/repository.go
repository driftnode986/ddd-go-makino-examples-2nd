// Package sharedpointer はポインタをそのまま保持する in-memory の
// リポジトリが、本物と違う動きをすることを見せる。
package sharedpointer

import (
	"context"
	"fmt"
	"sync"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

// #region repository

// Repository は受け取ったポインタをそのまま保持して返す。
type Repository struct {
	mu    sync.RWMutex
	items map[ids.BookingID]*domain.Booking
}

// New は空のリポジトリを返す。
func New() *Repository {
	return &Repository{items: map[ids.BookingID]*domain.Booking{}}
}

// FindByID は保持しているポインタを返す。
func (r *Repository) FindByID(
	_ context.Context, id ids.BookingID,
) (*domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	return b, nil
}

// #endregion

// Add はポインタを保持する。
func (r *Repository) Add(
	ctx context.Context, b *domain.Booking,
) error {
	if _, ok := memtx.From(ctx); !ok {
		return txn.ErrNoTransaction
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[b.ID()]; ok {
		return fmt.Errorf("%w: %s", domain.ErrDuplicate, b.ID())
	}
	r.items[b.ID()] = b
	return nil
}

// Save はポインタを置き換える。
func (r *Repository) Save(
	ctx context.Context, b *domain.Booking,
) error {
	if _, ok := memtx.From(ctx); !ok {
		return txn.ErrNoTransaction
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[b.ID()]; !ok {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, b.ID())
	}
	r.items[b.ID()] = b
	return nil
}

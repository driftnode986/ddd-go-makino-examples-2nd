// Package memory は MemberRepository の in-memory 実装である。
// テストダブルとして使う。
package memory

import (
	"context"
	"fmt"
	"sync"

	"example.com/trainingbooking/internal/identity/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

// MemberRepository は集約の中身をコピーで保持する。
type MemberRepository struct {
	mu     sync.RWMutex
	states map[ids.MemberID]domain.MemberState
}

// NewMemberRepository は空のリポジトリを返す。
func NewMemberRepository() *MemberRepository {
	return &MemberRepository{
		states: make(map[ids.MemberID]domain.MemberState),
	}
}

// Add は新しい会員を ctx の Tx の中で保存する。
func (r *MemberRepository) Add(
	ctx context.Context, m *domain.Member,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	state := m.State()
	if _, err := domain.ReconstructMember(state); err != nil {
		return err
	}
	cur, ok := memtx.From(ctx)
	if !ok {
		return txn.ErrNoTransaction
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, other := range r.states {
		if other.ID == state.ID || other.Email == state.Email {
			return fmt.Errorf(
				"%w: %s", domain.ErrDuplicate, state.ID)
		}
	}
	r.states[state.ID] = state
	cur.OnRollback(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.states, state.ID)
	})
	return nil
}

// FindByID は保存された中身から集約を組み立てる。
func (r *MemberRepository) FindByID(
	ctx context.Context, id ids.MemberID,
) (*domain.Member, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.states[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	return domain.ReconstructMember(state)
}

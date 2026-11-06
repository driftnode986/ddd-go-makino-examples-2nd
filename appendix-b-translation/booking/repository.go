package booking

import (
	"context"
	"errors"
	"sync"
)

// リポジトリが返すエラーである。実装の違いをここで吸収する。
var (
	ErrNotFound        = errors.New("booking not found")
	ErrDuplicate       = errors.New("booking already exists")
	ErrVersionConflict = errors.New("booking version conflict")
)

// Repository は集約単位の保存と読み出しである。
//
//	interface BookingRepository : JpaRepository<Booking, UUID>
//
// 実装は生成されない。メソッドは使う分だけ宣言する。
type Repository interface {
	Add(ctx context.Context, b *Booking) error
	Save(ctx context.Context, b *Booking) error
	FindByID(ctx context.Context, id BookingID) (*Booking, error)
}

// MemoryRepository は Repository の in-memory 実装である。
// @Version の判定に当たる版の比較もここで行う。
type MemoryRepository struct {
	mu     sync.Mutex
	stored map[BookingID]Booking
}

// NewMemoryRepository は空のリポジトリを返す。
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{stored: make(map[BookingID]Booking)}
}

// Add は新しい予約を保存する。
func (r *MemoryRepository) Add(
	_ context.Context, b *Booking,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.stored[b.id]; ok {
		return ErrDuplicate
	}
	next := *b
	next.events = nil
	r.stored[b.id] = next
	return nil
}

// Save は読み込んだときの版で判定して更新する。
func (r *MemoryRepository) Save(
	_ context.Context, b *Booking,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.stored[b.id]
	if !ok {
		return ErrNotFound
	}
	if cur.version != b.version {
		return ErrVersionConflict
	}
	next := *b
	next.events = nil
	next.version++
	r.stored[b.id] = next
	return nil
}

// FindByID はコピーを返す。呼び出し側とポインタを共有しない。
func (r *MemoryRepository) FindByID(
	_ context.Context, id BookingID,
) (*Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.stored[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &cur, nil
}

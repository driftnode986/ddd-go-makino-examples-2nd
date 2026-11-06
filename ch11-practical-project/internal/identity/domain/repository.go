package domain

import (
	"context"

	"example.com/trainingbooking/internal/ids"
)

// MemberRepository は Member 集約を保存し、読み出す。
// Add は ctx の Tx に参加する。自分では Tx を開かない。
// ctx に Tx が無ければ実装はエラー（txn.ErrNoTransaction）を返す。
type MemberRepository interface {
	// Add は新しい会員を保存する。ID か Email が重複すれば
	// ErrDuplicate を返す。
	Add(ctx context.Context, m *Member) error
	// FindByID は会員を読み出す。Tx なしでも読める。
	// 会員が無ければ ErrNotFound を返す。
	FindByID(ctx context.Context, id ids.MemberID) (*Member, error)
}

package domain

import (
	"errors"
	"fmt"

	"example.com/trainingbooking/internal/ids"
)

// リポジトリが返すエラーである。
var (
	ErrNotFound  = errors.New("member not found")
	ErrDuplicate = errors.New("member already exists")
)

// ErrInvalidMember は会員の値が不正なことを表す。
var ErrInvalidMember = errors.New("invalid member")

// Member は会員の集約ルートである。
type Member struct {
	id    ids.MemberID
	email Email
	name  string
}

// #region new-member

// NewMember は会員を作る。ID の採番は呼び出し側で行う。
func NewMember(
	id ids.MemberID, email Email, name string,
) (*Member, error) {
	if id.IsZero() || email.IsZero() || name == "" {
		return nil, fmt.Errorf(
			"%w: required value is missing", ErrInvalidMember)
	}
	return &Member{id: id, email: email, name: name}, nil
}

// #endregion

// ID は会員の ID を返す。
func (m *Member) ID() ids.MemberID { return m.id }

// MemberState は保存と再構築のための集約の中身である。
type MemberState struct {
	ID    ids.MemberID
	Email string
	Name  string
}

// State は集約の中身のコピーを返す。
func (m *Member) State() MemberState {
	return MemberState{
		ID:    m.id,
		Email: m.email.String(),
		Name:  m.name,
	}
}

// ReconstructMember は保存された中身から集約を組み立てる。
// 保存されていてはいけない値はエラーにする。
func ReconstructMember(s MemberState) (*Member, error) {
	email, err := NewEmail(s.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidMember, err)
	}
	return NewMember(s.ID, email, s.Name)
}

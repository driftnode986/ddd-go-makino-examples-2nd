// Package application は Identity のユースケースである。
package application

import (
	"context"
	"errors"

	"example.com/trainingbooking/internal/identity/domain"
	"example.com/trainingbooking/internal/ids"
)

// MemberReader は contract.MemberReader の実装である。
type MemberReader struct {
	repo domain.MemberRepository
}

// NewMemberReader は MemberReader を返す。
func NewMemberReader(repo domain.MemberRepository) *MemberReader {
	return &MemberReader{repo: repo}
}

// #region exists

// MemberExists は会員がいれば true を返す。
func (r *MemberReader) MemberExists(
	ctx context.Context, id ids.MemberID,
) (bool, error) {
	_, err := r.repo.FindByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// #endregion

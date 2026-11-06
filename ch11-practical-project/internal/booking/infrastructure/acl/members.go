package acl

import (
	"context"

	"example.com/trainingbooking/internal/identity/contract"
	"example.com/trainingbooking/internal/ids"
)

// MemberDirectory は Identity の会員の有無を Booking に伝える。
type MemberDirectory struct {
	reader contract.MemberReader
}

// NewMemberDirectory は MemberDirectory を返す。
func NewMemberDirectory(
	reader contract.MemberReader,
) *MemberDirectory {
	return &MemberDirectory{reader: reader}
}

// Exists は会員がいれば true を返す。
func (d *MemberDirectory) Exists(
	ctx context.Context, id ids.MemberID,
) (bool, error) {
	return d.reader.MemberExists(ctx, id)
}

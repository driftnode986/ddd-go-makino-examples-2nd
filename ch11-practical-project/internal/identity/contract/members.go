// Package contract は Identity が他のコンテキストへ公開する型である。
// 他のコンテキストから import できるのはこのパッケージだけである。
package contract

import (
	"context"

	"example.com/trainingbooking/internal/ids"
)

// MemberReader は会員の有無を答える。Email などの個人情報は
// 他のコンテキストに見せない。
type MemberReader interface {
	MemberExists(ctx context.Context, id ids.MemberID) (bool, error)
}

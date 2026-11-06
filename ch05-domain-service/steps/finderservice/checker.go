// Package finderservice は、ドメインサービスが読み取りのインターフェースを
// 持つ形の例である。
package finderservice

import (
	"context"
	"fmt"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// #region checker

// SlotFinder は、会員の有効な予約の時間帯を読む。
type SlotFinder interface {
	ActiveSlots(
		ctx context.Context, member ids.MemberID,
	) ([]domain.TimeSlot, error)
}

// OverlapChecker は SlotFinder で読んで重なりを判定する。
type OverlapChecker struct {
	finder SlotFinder
}

// Check は会員の予約と重なれば ErrOverlap を返す。
func (c *OverlapChecker) Check(
	ctx context.Context, member ids.MemberID, slot domain.TimeSlot,
) error {
	existing, err := c.finder.ActiveSlots(ctx, member)
	if err != nil {
		return fmt.Errorf("read active slots: %w", err)
	}
	return domain.DetectOverlap(slot, existing)
}

// #endregion

// New は OverlapChecker を返す。
func New(finder SlotFinder) *OverlapChecker {
	return &OverlapChecker{finder: finder}
}

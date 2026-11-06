package domain

import "fmt"

// #region detect-overlap

// DetectOverlap は slot が existing のどれかと重なるとき、
// ErrOverlap を包んで返す。1つの Booking では他の予約が見えない
// ので、Booking のメソッドでなくドメインサービスにする。
func DetectOverlap(slot TimeSlot, existing []TimeSlot) error {
	if slot.IsZero() {
		return ErrInvalidTimeSlot
	}
	for _, e := range existing {
		if slot.Overlaps(e) {
			return fmt.Errorf(
				"%w: %s to %s", ErrOverlap, e.Start(), e.End())
		}
	}
	return nil
}

// #endregion

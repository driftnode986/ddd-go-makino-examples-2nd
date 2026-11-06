// Package closedslot は、終了の時刻も時間帯に含めて判定する例である。
package closedslot

import "example.com/trainingbooking/internal/booking/domain"

// #region overlaps

// Overlaps は開始と終了の両方を時間帯に含めて判定する。
func Overlaps(a, b domain.TimeSlot) bool {
	return !a.End().Before(b.Start()) &&
		!b.End().Before(a.Start())
}

// #endregion

// Package naiveslot は、タイムゾーンを考えずに営業時間を判定する例である。
package naiveslot

import "time"

// #region naive
func InBusinessHours(start time.Time) bool {
	h := start.Hour()
	return h >= 9 && h < 21
}

// #endregion

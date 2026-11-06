// Package statefuldetector は、判定した時間帯を覚えるサービスの例である。
package statefuldetector

import "example.com/trainingbooking/internal/booking/domain"

// #region detector

// Detector は、これまでに通した時間帯をフィールドに覚える。
type Detector struct {
	seen []domain.TimeSlot
}

// Detect は覚えている時間帯と重なれば ErrOverlap を返す。
func (d *Detector) Detect(slot domain.TimeSlot) error {
	if err := domain.DetectOverlap(slot, d.seen); err != nil {
		return err
	}
	d.seen = append(d.seen, slot)
	return nil
}

// #endregion

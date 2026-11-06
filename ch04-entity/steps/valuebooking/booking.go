// Package valuebooking は、予約を ID の無い値として書いた例である。
package valuebooking

import (
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// #region type

// Booking は中身だけで比べる予約である。ID を持たない。
type Booking struct {
	Member ids.MemberID
	Lesson ids.LessonID
	Slot   domain.TimeSlot
	Status domain.Status
}

// #endregion

// Package openstatus は、状態を公開フィールドにした例である。
package openstatus

import "example.com/trainingbooking/internal/booking/domain"

// #region type

// Booking は状態を公開している。
type Booking struct {
	Status domain.Status
}

// #endregion

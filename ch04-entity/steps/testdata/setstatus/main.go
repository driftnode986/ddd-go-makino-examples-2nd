// わざとコンパイルできないコードである。
package main

import "example.com/trainingbooking/internal/booking/domain"

// #region assign

func main() {
	b := &domain.Booking{}
	b.status = domain.StatusConfirmed
}

// #endregion

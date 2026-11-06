package anemic_test

import (
	"testing"
	"time"

	"example.com/trainingbooking/steps/anemic"
)

func TestRuleIsBypassed(t *testing.T) {
	now := time.Date(2030, time.May, 1, 10, 0, 0, 0, time.UTC)
	start := now.AddDate(0, 1, 0)
	// #region bypass
	b := &anemic.Booking{Status: "pending", Start: start}
	err := anemic.CancelService{}.Cancel(b, now)
	t.Logf("CancelService:  %v, Status=%s", err, b.Status)
	anemic.CleanupService{}.CancelStale(b)
	t.Logf("CleanupService: Status=%s", b.Status)
	// #endregion
}

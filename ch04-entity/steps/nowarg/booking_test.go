package nowarg_test

import (
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/steps/nowarg"
)

func TestDeadline(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	b := nowarg.New(slot)
	deadline := start.Add(-24 * time.Hour)
	// #region test
	if err := b.Cancel(deadline); err != nil {
		t.Errorf("期限ちょうど: err = %v", err)
	}
	err = b.Cancel(deadline.Add(1))
	if !errors.Is(err, domain.ErrCancelTooLate) {
		t.Errorf("期限の直後: err = %v", err)
	}
	// #endregion
}

package nowcancel_test

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/steps/nowcancel"
)

var jst = time.FixedZone("JST", 9*60*60)

func bookingAt(t *testing.T, start time.Time) *nowcancel.Booking {
	t.Helper()
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	return nowcancel.New(slot)
}

// #region bubble

func TestDeadlineInBubble(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// バブルの中の時計は 2000年1月1日0時(UTC)から始まる
		start := time.Date(2000, time.January, 3, 10, 0, 0, 0, jst)
		b := bookingAt(t, start)

		time.Sleep(time.Until(start.Add(-24 * time.Hour)))
		if err := b.Cancel(); err != nil {
			t.Errorf("期限ちょうど: err = %v", err)
		}
		time.Sleep(time.Nanosecond)
		if err := b.Cancel(); !errors.Is(
			err, domain.ErrCancelTooLate) {
			t.Errorf("期限の直後: err = %v", err)
		}
	})
}

// #endregion

func TestResultDependsOnToday(t *testing.T) {
	// #region today
	future := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	t.Logf("lesson in 2030: err=%v", bookingAt(t, future).Cancel())
	past := time.Date(2025, time.June, 1, 10, 0, 0, 0, jst)
	t.Logf("lesson in 2025: err=%v", bookingAt(t, past).Cancel())
	// #endregion
}

func TestRelativeStartIsRejected(t *testing.T) {
	// #region relative
	start := time.Now().Add(23 * time.Hour)
	_, err := domain.NewTimeSlot(start, time.Hour, jst)
	t.Logf("not on the hour: %v",
		errors.Is(err, domain.ErrNotOnTheHour))
	// #endregion
}

package closedslot_test

import (
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/steps/closedslot"
)

var jst = time.FixedZone("JST", 9*60*60)

func slotAt(t *testing.T, hour int) domain.TimeSlot {
	t.Helper()
	start := time.Date(2030, time.June, 1, hour, 0, 0, 0, jst)
	s, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAdjacentSlots(t *testing.T) {
	// #region adjacent
	// 10〜11時と11〜12時
	ten, eleven := slotAt(t, 10), slotAt(t, 11)
	t.Logf("closed:   %v", closedslot.Overlaps(ten, eleven))
	t.Logf("Overlaps: %v", ten.Overlaps(eleven))
	// #endregion
}

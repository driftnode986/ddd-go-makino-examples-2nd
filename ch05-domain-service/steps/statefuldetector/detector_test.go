package statefuldetector_test

import (
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/steps/statefuldetector"
)

func TestSharedAcrossMembers(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	// #region shared
	// main で1つ作って、すべてのリクエストで共有する
	detector := &statefuldetector.Detector{}
	t.Logf("member A: %v", detector.Detect(slot))
	// 別の会員の、同じ時間帯の予約
	err = detector.Detect(slot)
	t.Logf("member B: ErrOverlap=%v",
		errors.Is(err, domain.ErrOverlap))
	// #endregion
}

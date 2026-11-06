package finderservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/steps/finderservice"
)

// fakeFinder はテスト用の SlotFinder である。
type fakeFinder struct {
	slots []domain.TimeSlot
	err   error
}

func (f fakeFinder) ActiveSlots(
	context.Context, ids.MemberID,
) ([]domain.TimeSlot, error) {
	return f.slots, f.err
}

func TestOverlapChecker(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	member := ids.NewID[ids.MemberMarker]()
	boom := errors.New("db down")

	c := finderservice.New(
		fakeFinder{slots: []domain.TimeSlot{slot}})
	err = c.Check(t.Context(), member, slot)
	if !errors.Is(err, domain.ErrOverlap) {
		t.Errorf("err = %v", err)
	}
	c = finderservice.New(fakeFinder{err: boom})
	err = c.Check(t.Context(), member, slot)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

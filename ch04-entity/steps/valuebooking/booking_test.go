package valuebooking_test

import (
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/steps/valuebooking"
)

func TestNoIdentity(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	member := ids.NewID[ids.MemberMarker]()
	lesson := ids.NewID[ids.LessonMarker]()

	// #region no-identity
	// 同じ予約が保留から確定に変わる
	before := valuebooking.Booking{
		Member: member, Lesson: lesson, Slot: slot,
		Status: domain.StatusPending,
	}
	after := before
	after.Status = domain.StatusConfirmed
	t.Logf("same booking, before and after: %v", before == after)

	// 予約してキャンセルし、取り直してまたキャンセルした2件
	first := valuebooking.Booking{
		Member: member, Lesson: lesson, Slot: slot,
		Status: domain.StatusCancelled,
	}
	second := first
	t.Logf("two different bookings: %v", first == second)
	// #endregion

	if before == after || first != second {
		t.Fatal("中身だけの比較では予約を見分けられないはず")
	}
}

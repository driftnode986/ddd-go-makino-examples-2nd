package openstatus_test

import (
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/steps/openstatus"
)

func TestAnyStatusCanBeAssigned(t *testing.T) {
	// #region assign
	b := openstatus.Booking{Status: domain.StatusRejected}
	// 拒否された予約を確定にする
	b.Status = domain.StatusConfirmed
	t.Logf("rejected -> %s", b.Status)
	// 存在しない状態にする
	b.Status = "done"
	t.Logf("confirmed -> %s", b.Status)
	// #endregion
	if b.Status != "done" {
		t.Fatal("公開フィールドは何でも入るはず")
	}
}

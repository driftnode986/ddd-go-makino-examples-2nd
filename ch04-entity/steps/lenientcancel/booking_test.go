package lenientcancel_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/steps/lenientcancel"
)

func TestSeatIsLeft(t *testing.T) {
	// #region race
	b := lenientcancel.New()
	lesson := &lenientcancel.Lesson{}

	// 席の確保の結果を待っている間に、会員がキャンセルする
	err := b.Cancel()
	t.Logf("Cancel: err=%v status=%s", err, b.Status())

	// キャンセルを知らない Catalog が席を確保し、結果が返る
	lesson.ReserveSeat()
	err = b.Confirm()
	t.Logf("Confirm: err=%v", err)
	t.Logf("status=%s reserved=%d", b.Status(), lesson.Reserved)
	// #endregion
	if !errors.Is(err, domain.ErrInvalidTransition) ||
		lesson.Reserved != 1 {
		t.Fatal("席だけが確保されたまま残るはず")
	}
}

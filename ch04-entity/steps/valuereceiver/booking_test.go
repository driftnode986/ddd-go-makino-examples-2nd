package valuereceiver_test

import (
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/steps/valuereceiver"
)

func TestConfirmHasNoEffect(t *testing.T) {
	// #region run
	b := valuereceiver.New()
	err := b.Confirm()
	t.Logf("err=%v status=%s", err, b.Status())
	// #endregion
	if err != nil || b.Status() != domain.StatusPending {
		t.Fatal("値レシーバの変更は呼び出し元に届かないはず")
	}
}

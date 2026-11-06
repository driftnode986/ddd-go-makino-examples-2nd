package rawrebuild_test

import (
	"testing"

	"example.com/trainingbooking/steps/rawrebuild"
)

// #region broken-row

func TestBrokenRowIsAccepted(t *testing.T) {
	b := rawrebuild.Rebuild(rawrebuild.State{
		FeeYen:      3000,
		DiscountYen: 5000,
		Status:      "confirmd", // 綴りの誤り
	})
	t.Logf("合計: %d円 状態: %s", b.Total(), b.Status())
}

// #endregion

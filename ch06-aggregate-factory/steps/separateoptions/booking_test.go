package separateoptions_test

import (
	"fmt"
	"testing"

	sep "example.com/trainingbooking/steps/separateoptions"
)

// #region negative-total

func TestTotalGoesNegative(t *testing.T) {
	b := sep.Booking{FeeYen: 3000, Status: "pending"}
	shoes := sep.Option{
		Name: "シューズ", UnitPriceYen: 500, Quantity: 2,
	}
	options, err := sep.AddFromForm(b, nil, shoes)
	if err != nil {
		t.Fatal(err)
	}
	// 100% のクーポンを、その時点の合計に適用する
	b.DiscountYen = sep.Total(b, options)
	t.Logf("割引: %d円", b.DiscountYen)
	t.Logf("合計: %d円", sep.Total(b, options))

	sep.CorrectQuantity(options, "シューズ", 1)
	t.Logf("数量の訂正後の合計: %d円", sep.Total(b, options))
}

// #endregion

// #region six-options

func TestLimitIsBypassed(t *testing.T) {
	b := sep.Booking{FeeYen: 3000, Status: "confirmed"}
	var options []sep.Option
	for i := range 6 {
		options = sep.AddFromBatch(options, sep.Option{
			Name:         fmt.Sprintf("オプション%d", i+1),
			UnitPriceYen: 100,
			Quantity:     1,
		})
	}
	t.Logf("%s の予約の明細: %d件", b.Status, len(options))
}

// #endregion

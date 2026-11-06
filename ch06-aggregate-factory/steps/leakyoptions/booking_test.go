package leakyoptions_test

import (
	"slices"
	"testing"

	"example.com/trainingbooking/steps/leakyoptions"
)

// #region rewrite

func TestRewriteThroughGetter(t *testing.T) {
	b := leakyoptions.New(3000)
	b.AddOption(leakyoptions.Option{Name: "シューズ", PriceYen: 500})

	got := b.Options()
	got[0].PriceYen = 0

	t.Logf("Options の単価: %d円", b.Options()[0].PriceYen)
	t.Logf("Total: %d円", b.Total())
}

// #endregion

// #region shallow

func TestCloneIsShallow(t *testing.T) {
	shoes := &leakyoptions.Option{Name: "シューズ", PriceYen: 500}
	internal := []*leakyoptions.Option{shoes}

	got := slices.Clone(internal)
	got[0].PriceYen = 0

	t.Logf("ポインタのスライスの単価: %d円", internal[0].PriceYen)
}

// #endregion

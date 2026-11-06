package naiveslot_test

import (
	"testing"
	"time"

	"example.com/trainingbooking/steps/naiveslot"
)

// #region zones
func TestSameInstantDifferentResult(t *testing.T) {
	a, _ := time.Parse(time.RFC3339, "2030-06-01T09:00:00+09:00")
	b, _ := time.Parse(time.RFC3339, "2030-06-01T00:00:00Z")
	t.Logf("same instant: %v", a.Equal(b))
	t.Logf("+09:00: %v", naiveslot.InBusinessHours(a))
	t.Logf("Z:      %v", naiveslot.InBusinessHours(b))
	if naiveslot.InBusinessHours(a) == naiveslot.InBusinessHours(b) {
		t.Fatal("同じ瞬間で結果が分かれるはず")
	}
}

// #endregion

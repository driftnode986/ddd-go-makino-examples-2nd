package receiver_test

import (
	"testing"

	"example.com/trainingbooking/receiver"
)

func TestPoints_AddKeepsOriginal(t *testing.T) {
	a := receiver.NewPoints(1)
	b := a.Add(receiver.NewPoints(2))
	if a.Count() != 1 || b.Count() != 3 {
		t.Fatalf("a = %d, b = %d", a.Count(), b.Count())
	}
}

func chargeCopy(c receiver.Card) {
	c.Charge(receiver.NewPoints(100))
}

// struct を値で渡すとコピーに対してメソッドが動く。
func TestCard_CopyDoesNotChangeOriginal(t *testing.T) {
	var c receiver.Card
	chargeCopy(c)
	if c.Balance().Count() != 0 {
		t.Fatalf("Balance = %d", c.Balance().Count())
	}
	c.Charge(receiver.NewPoints(100))
	if c.Balance().Count() != 100 {
		t.Fatalf("Balance = %d", c.Balance().Count())
	}
}

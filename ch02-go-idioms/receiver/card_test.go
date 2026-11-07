package receiver_test

import (
	"testing"

	"example.com/trainingbooking/receiver"
)

// #region add

func TestPoints_AddKeepsOriginal(t *testing.T) {
	a := receiver.NewPoints(1)
	b := a.Add(receiver.NewPoints(2))

	if a.Count() != 1 || b.Count() != 3 {
		t.Fatalf("a = %d, b = %d", a.Count(), b.Count())
	}
}

// #endregion

// #region charge

type charger interface {
	Charge(p receiver.Points)
}

func topUp(c charger) { c.Charge(receiver.NewPoints(100)) }

func TestCard_PointerSatisfiesInterface(t *testing.T) {
	c := &receiver.Card{}

	topUp(c)

	if c.Balance().Count() != 100 {
		t.Fatalf("Balance = %d", c.Balance().Count())
	}
}

// #endregion

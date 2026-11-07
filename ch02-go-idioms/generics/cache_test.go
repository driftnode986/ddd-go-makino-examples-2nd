package generics_test

import (
	"testing"

	"example.com/trainingbooking/generics"
)

// #region get

func TestCache_GetWithTypeArgument(t *testing.T) {
	c := generics.NewCache()
	if err := c.Put("count", 7); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := c.Get[int]("count")

	if err != nil || got != 7 {
		t.Fatalf("got = %d, err = %v", got, err)
	}
}

// #endregion

package generics_test

import (
	"testing"

	"example.com/trainingbooking/generics"
)

func TestStore_FindWithTypeArgument(t *testing.T) {
	s := generics.NewStore()
	if err := s.Put("count", 7); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := s.Find[int]("count")

	if err != nil || got != 7 {
		t.Fatalf("got = %d, err = %v", got, err)
	}
}

package twoaggregates_test

import (
	"errors"
	"testing"

	two "example.com/trainingbooking/steps/twoaggregates"
)

// #region one-tx-test

func TestOneTxFailsWithSeatsLeft(t *testing.T) {
	s := two.NewStore(10)
	readA, readB := s.Lesson(), s.Lesson() // 2人が同時に読む

	t.Logf("A: %v", s.PlaceInOneTx("A", readA))
	t.Logf("B: %v", s.PlaceInOneTx("B", readB))
	t.Logf("予約: %v 残席: %d", s.Bookings, s.Lesson().Seats)
}

// #endregion

// #region separate-test

func TestSeparateTxKeepsBothBookings(t *testing.T) {
	s := two.NewStore(10)
	s.Place("A")
	s.Place("B")

	readA, readB := s.Lesson(), s.Lesson()
	t.Logf("A の席: %v", s.ReserveSeat(readA))
	err := s.ReserveSeat(readB)
	t.Logf("B の席: %v", err)
	if errors.Is(err, two.ErrConflict) {
		// 予約は残っているので、読み直して席だけやり直す
		err = s.ReserveSeat(s.Lesson())
		t.Logf("B の席（読み直し後）: %v", err)
	}
	t.Logf("予約: %v 残席: %d", s.Bookings, s.Lesson().Seats)
}

// #endregion

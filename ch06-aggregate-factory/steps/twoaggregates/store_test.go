package twoaggregates_test

import (
	"testing"

	two "example.com/trainingbooking/steps/twoaggregates"
)

// retry は成功するか回数を使い切るまで fn を呼び直す。
func retry(times int, fn func() error) error {
	var err error
	for range times {
		if err = fn(); err == nil {
			return nil
		}
	}
	return err
}

// otherTakesSeat はほかの予約が先に席を確保する。
func otherTakesSeat(t *testing.T, s *two.Store) {
	t.Helper()
	if err := s.ReserveSeat(s.Lesson()); err != nil {
		t.Fatal(err)
	}
}

// #region reread

func TestOneTxPassesAfterReread(t *testing.T) {
	s := two.NewStore(10)
	readA, readB := s.Lesson(), s.Lesson() // 2人が同時に読む

	err := s.PlaceInOneTx("A", readA)
	t.Logf("A: %v", err)
	err = s.PlaceInOneTx("B", readB)
	t.Logf("B: %v", err)
	err = s.PlaceInOneTx("B", s.Lesson()) // 読み直す
	t.Logf("B（読み直し後）: %v", err)
	t.Logf("予約: %v 残席: %d", s.Bookings, s.Lesson().Seats)
}

// #endregion

// #region one-tx-load

func TestOneTxUnderLoad(t *testing.T) {
	s := two.NewStore(10)
	err := retry(3, func() error {
		read := s.Lesson()
		otherTakesSeat(t, s)
		return s.PlaceInOneTx("B", read)
	})
	t.Logf("B（3回）: %v", err)
	t.Logf("予約: %v 残席: %d", s.Bookings, s.Lesson().Seats)
}

// #endregion

// #region separate-load

func TestSeparateTxUnderLoad(t *testing.T) {
	s := two.NewStore(10)
	s.Place("B") // ここで利用者に返せる
	err := retry(3, func() error {
		read := s.Lesson()
		otherTakesSeat(t, s)
		return s.ReserveSeat(read)
	})
	t.Logf("B の席（3回）: %v", err)
	t.Logf("予約: %v 残席: %d", s.Bookings, s.Lesson().Seats)

	// 集中が収まった後に、席の確保だけをやり直す
	err = s.ReserveSeat(s.Lesson())
	t.Logf("B の席（後で再試行）: %v", err)
	t.Logf("予約: %v 残席: %d", s.Bookings, s.Lesson().Seats)
}

// #endregion

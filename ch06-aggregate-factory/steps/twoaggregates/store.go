// Package twoaggregates は、予約とレッスンの席を1つの Tx で
// 更新する場合と、分ける場合を比べる。DB の代わりに map を使う。
package twoaggregates

import "errors"

// ErrConflict は読み込んだ後に別の処理が更新したことを表す。
var ErrConflict = errors.New("version conflict")

// Lesson はレッスンの残席と、読み込んだときの版である。
type Lesson struct {
	Seats   int
	Version int
}

// Store は1つのレッスンと予約の一覧を持つ。
type Store struct {
	lesson   Lesson
	Bookings []string
}

// NewStore は残席を指定して Store を返す。
func NewStore(seats int) *Store {
	return &Store{lesson: Lesson{Seats: seats}}
}

// Lesson はレッスンを読み込む。
func (s *Store) Lesson() Lesson { return s.lesson }

// #region one-tx

// PlaceInOneTx は予約の保存と席の確保を1つの Tx で行う。
// レッスンの版が変わっていれば、予約も保存しない。
func (s *Store) PlaceInOneTx(id string, read Lesson) error {
	if read.Version != s.lesson.Version {
		return ErrConflict
	}
	s.lesson = Lesson{
		Seats: read.Seats - 1, Version: read.Version + 1,
	}
	s.Bookings = append(s.Bookings, id)
	return nil
}

// #endregion

// Place は予約だけを保存する。席の確保は後で別に行う。
func (s *Store) Place(id string) {
	s.Bookings = append(s.Bookings, id)
}

// ReserveSeat は席を確保する。版が変わっていれば失敗する。
func (s *Store) ReserveSeat(read Lesson) error {
	if read.Version != s.lesson.Version {
		return ErrConflict
	}
	s.lesson = Lesson{
		Seats: read.Seats - 1, Version: read.Version + 1,
	}
	return nil
}

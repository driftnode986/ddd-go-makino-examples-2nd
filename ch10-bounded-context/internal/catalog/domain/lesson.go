// Package domain は Catalog コンテキストのドメインモデルである。
package domain

import (
	"fmt"
	"slices"
	"time"

	"example.com/trainingbooking/internal/ids"
)

// LessonOption はレッスンで選べるオプションと単価である。
type LessonOption struct {
	Name         string
	UnitPriceYen int64
}

// LessonSpec は新しいレッスンの内容である。
type LessonSpec struct {
	ID       ids.LessonID
	Title    string
	Start    time.Time
	Duration time.Duration
	FeeYen   int64
	Capacity int
	Options  []LessonOption
}

// Lesson はレッスンの集約ルートである。
// 不変条件は、席を持つ予約の数が定員を超えないことである。
type Lesson struct {
	id       ids.LessonID
	title    string
	start    time.Time
	duration time.Duration
	feeYen   int64
	capacity int
	options  []LessonOption
	seats    []ids.BookingID
	version  int // 読み込んだときの版。メソッドでは増やさない
	events   []DomainEvent
}

// NewLesson は席の空いたレッスンを作る。
func NewLesson(s LessonSpec) (*Lesson, error) {
	return ReconstructLesson(LessonState{
		ID:       s.ID,
		Title:    s.Title,
		Start:    s.Start,
		Duration: s.Duration,
		FeeYen:   s.FeeYen,
		Capacity: s.Capacity,
		Options:  s.Options,
	})
}

// ID はレッスンの ID を返す。
func (l *Lesson) ID() ids.LessonID { return l.id }

// ReservedSeats は確保済みの席の数を返す。
func (l *Lesson) ReservedSeats() int { return len(l.seats) }

// Version は読み込んだときの版を返す。
func (l *Lesson) Version() int { return l.version }

// Equal は同じレッスンのとき true を返す。
func (l *Lesson) Equal(other *Lesson) bool {
	return other != nil && l.id.Equal(other.id)
}

// ReserveSeat は予約の席を確保する。同じ予約で2回呼んでも席は
// 1つで、イベントも1回だけ記録する。満席なら SeatRejected を
// 記録し、ErrLessonFull を返す。
func (l *Lesson) ReserveSeat(
	booking ids.BookingID, clk Clock,
) error {
	if booking.IsZero() {
		return fmt.Errorf(
			"%w: booking id is missing", ErrInvalidLesson)
	}
	if slices.Contains(l.seats, booking) {
		return nil
	}
	now := clk.Now()
	if len(l.seats) >= l.capacity {
		l.events = append(l.events, SeatRejected{
			LessonID:   l.id,
			BookingID:  booking,
			Reason:     RejectLessonFull,
			OccurredAt: now,
		})
		return fmt.Errorf("%w: %s", ErrLessonFull, l.id)
	}
	l.seats = append(l.seats, booking)
	l.events = append(l.events, SeatReserved{
		LessonID:   l.id,
		BookingID:  booking,
		OccurredAt: now,
	})
	return nil
}

// ReleaseSeat は予約の席を戻す。席を持たない予約では何もしない。
// 同じ予約で2回呼んでも1回だけ効く。
func (l *Lesson) ReleaseSeat(booking ids.BookingID) error {
	if booking.IsZero() {
		return fmt.Errorf(
			"%w: booking id is missing", ErrInvalidLesson)
	}
	i := slices.Index(l.seats, booking)
	if i >= 0 {
		l.seats = slices.Delete(slices.Clone(l.seats), i, i+1)
	}
	return nil
}

// PullEvents は記録したイベントを取り出し、集約から消す。
func (l *Lesson) PullEvents() []DomainEvent {
	events := l.events
	l.events = nil
	return events
}

package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"example.com/trainingbooking/internal/ids"
)

// LessonState は保存と再構築のための集約の中身である。
type LessonState struct {
	ID       ids.LessonID
	Title    string
	Start    time.Time
	Duration time.Duration
	FeeYen   int64
	Capacity int
	Options  []LessonOption
	Seats    []ids.BookingID
	Version  int
}

// State は集約の中身のコピーを返す。
func (l *Lesson) State() LessonState {
	return LessonState{
		ID:       l.id,
		Title:    l.title,
		Start:    l.start,
		Duration: l.duration,
		FeeYen:   l.feeYen,
		Capacity: l.capacity,
		Options:  slices.Clone(l.options),
		Seats:    slices.Clone(l.seats),
		Version:  l.version,
	}
}

// ReconstructLesson は保存された中身から集約を組み立てる。
// 保存されていてはいけない値はエラーにする。
// イベントは記録しない。
func ReconstructLesson(s LessonState) (*Lesson, error) {
	if err := validate(s); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidLesson, err)
	}
	return &Lesson{
		id:       s.ID,
		title:    s.Title,
		start:    s.Start,
		duration: s.Duration,
		feeYen:   s.FeeYen,
		capacity: s.Capacity,
		options:  slices.Clone(s.Options),
		seats:    slices.Clone(s.Seats),
		version:  s.Version,
	}, nil
}

func validate(s LessonState) error {
	switch {
	case s.ID.IsZero():
		return errors.New("id is missing")
	case s.Title == "":
		return errors.New("title is empty")
	case s.Start.IsZero():
		return errors.New("start is missing")
	case s.Duration <= 0:
		return fmt.Errorf("duration %s", s.Duration)
	case s.FeeYen <= 0:
		return fmt.Errorf("fee %d", s.FeeYen)
	case s.Capacity <= 0:
		return fmt.Errorf("capacity %d", s.Capacity)
	case s.Version < 0:
		return fmt.Errorf("version %d", s.Version)
	}
	if err := validateOptions(s.Options); err != nil {
		return err
	}
	return validateSeats(s.Seats, s.Capacity)
}

func validateOptions(options []LessonOption) error {
	seen := make(map[string]bool, len(options))
	for _, o := range options {
		if o.Name == "" || o.UnitPriceYen < 0 || seen[o.Name] {
			return fmt.Errorf("option %+v", o)
		}
		seen[o.Name] = true
	}
	return nil
}

func validateSeats(seats []ids.BookingID, capacity int) error {
	if len(seats) > capacity {
		return fmt.Errorf("seats %d over capacity %d",
			len(seats), capacity)
	}
	seen := make(map[ids.BookingID]bool, len(seats))
	for _, b := range seats {
		if b.IsZero() || seen[b] {
			return fmt.Errorf("seat %s", b)
		}
		seen[b] = true
	}
	return nil
}

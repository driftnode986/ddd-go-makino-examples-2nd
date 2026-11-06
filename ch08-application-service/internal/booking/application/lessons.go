package application

import (
	"errors"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
)

// ErrUnknownOption はレッスンで選べないオプションのエラーである。
var ErrUnknownOption = errors.New("option is not offered")

// OfferedOption はレッスンで選べるオプションと単価である。
type OfferedOption struct {
	Name      string
	UnitPrice domain.Money
}

// LessonTerms は予約に使うレッスンの条件である。
type LessonTerms struct {
	Start    time.Time
	Duration time.Duration
	Fee      domain.Money
	Location *time.Location
	Options  []OfferedOption
}

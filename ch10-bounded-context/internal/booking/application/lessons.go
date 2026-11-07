package application

import (
	"context"
	"errors"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// LessonCatalog が返すエラーと、選べないオプションのエラーである。
var (
	ErrLessonNotFound = errors.New("lesson not found")
	ErrUnknownOption  = errors.New("option is not offered")
)

// OfferedOption はレッスンで選べるオプションと単価である。
type OfferedOption struct {
	Name      string
	UnitPrice domain.Money
}

// #region lesson-terms

// LessonTerms は予約に使うレッスンの条件である。Booking の言葉で
// 表した値で、Catalog の型はここに現れない。
type LessonTerms struct {
	Start    time.Time
	Duration time.Duration
	Fee      domain.Money
	Location *time.Location
	Options  []OfferedOption
}

// #endregion

// #region lesson-catalog

// LessonCatalog はレッスンの条件を読む。
type LessonCatalog interface {
	// Terms はレッスンの条件を返す。無ければ ErrLessonNotFound を
	// 返す。
	Terms(ctx context.Context, id ids.LessonID) (LessonTerms, error)
}

// #endregion

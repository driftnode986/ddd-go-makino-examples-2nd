package contract

import (
	"context"
	"errors"
	"time"

	"example.com/trainingbooking/internal/ids"
)

// ErrLessonNotFound はレッスンが無いことを表す。
var ErrLessonNotFound = errors.New("lesson not found")

// LessonOption はレッスンで選べるオプションと単価である。
type LessonOption struct {
	Name         string
	UnitPriceYen int64
}

// #region lesson

// Lesson は他のコンテキストに見せるレッスンの情報である。
// 席の数は結果整合なので見せない。
type Lesson struct {
	ID       ids.LessonID
	Title    string
	Start    time.Time
	Duration time.Duration
	FeeYen   int64
	Options  []LessonOption
}

// #endregion

// #region lesson-reader

// LessonReader はレッスンを読む。
type LessonReader interface {
	// FindLesson はレッスンを返す。無ければ ErrLessonNotFound を返す。
	FindLesson(ctx context.Context, id ids.LessonID) (Lesson, error)
}

// #endregion

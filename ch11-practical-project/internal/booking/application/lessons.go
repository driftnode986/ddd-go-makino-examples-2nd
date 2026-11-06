package application

import (
	"context"
	"errors"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// 他のコンテキストの情報に基づいて入力を拒否するエラーである。
var (
	ErrLessonNotFound = errors.New("lesson not found")
	ErrUnknownOption  = errors.New("option is not offered")
	ErrMemberNotFound = errors.New("member not found")
)

// OfferedOption はレッスンで選べるオプションと単価である。
type OfferedOption struct {
	Name      string
	UnitPrice domain.Money
}

// LessonTerms は予約に使うレッスンの条件である。Booking の言葉で
// 表した値で、Catalog の型はここに現れない。
type LessonTerms struct {
	Start    time.Time
	Duration time.Duration
	Fee      domain.Money
	Location *time.Location
	Options  []OfferedOption
}

// LessonCatalog はレッスンの条件を読む。
type LessonCatalog interface {
	// Terms はレッスンの条件を返す。無ければ ErrLessonNotFound を
	// 返す。
	Terms(ctx context.Context, id ids.LessonID) (LessonTerms, error)
}

// MemberDirectory は会員の有無を答える。
type MemberDirectory interface {
	Exists(ctx context.Context, id ids.MemberID) (bool, error)
}

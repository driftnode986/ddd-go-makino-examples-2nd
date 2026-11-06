package domain

import (
	"errors"
	"fmt"
	"time"
)

// #region declarations

// 営業時間である。開始は9時から、終了は21時までとする。
const (
	openingHour = 9
	closingHour = 21
)

// TimeSlot の生成で起きるエラーである。
var (
	ErrInvalidTimeSlot    = errors.New("invalid time slot")
	ErrInvalidDuration    = errors.New("invalid duration")
	ErrNotOnTheHour       = errors.New("start is not on the hour")
	ErrOutOfBusinessHours = errors.New("out of business hours")
)

// TimeSlot は日付を含む開始時刻と長さで時間帯を表す。
type TimeSlot struct {
	start    time.Time
	duration time.Duration
}

// #endregion

// #region new-time-slot

// NewTimeSlot は時間帯を作る。正時と営業時間は loc の時刻で
// 判定するので、同じ瞬間ならどのゾーンで渡しても結果は同じである。
func NewTimeSlot(
	start time.Time, d time.Duration, loc *time.Location,
) (TimeSlot, error) {
	if start.IsZero() || loc == nil {
		return TimeSlot{}, ErrInvalidTimeSlot
	}
	if d <= 0 {
		return TimeSlot{}, fmt.Errorf(
			"%w: %s", ErrInvalidDuration, d)
	}
	local := start.In(loc)
	if local.Minute() != 0 || local.Second() != 0 ||
		local.Nanosecond() != 0 {
		return TimeSlot{}, fmt.Errorf(
			"%w: %s", ErrNotOnTheHour, local)
	}
	closing := time.Date(local.Year(), local.Month(), local.Day(),
		closingHour, 0, 0, 0, loc)
	if local.Hour() < openingHour || local.Add(d).After(closing) {
		return TimeSlot{}, fmt.Errorf(
			"%w: %s", ErrOutOfBusinessHours, local)
	}
	return TimeSlot{start: start, duration: d}, nil
}

// #endregion

// Start は開始時刻を返す。
func (s TimeSlot) Start() time.Time { return s.start }

// End は終了時刻を返す。
func (s TimeSlot) End() time.Time { return s.start.Add(s.duration) }

// Overlaps は時間帯が重なるとき true を返す。
// 半開区間として扱うので、接するだけなら重ならない。
func (s TimeSlot) Overlaps(other TimeSlot) bool {
	return s.Start().Before(other.End()) &&
		other.Start().Before(s.End())
}

// #region slot-equal

// Equal は同じ瞬間に始まり同じ長さのとき true を返す。
// ゾーンの違いは問わない。
func (s TimeSlot) Equal(other TimeSlot) bool {
	return s.start.Equal(other.start) &&
		s.duration == other.duration
}

// #endregion

// IsZero は NewTimeSlot を通していないとき true を返す。
func (s TimeSlot) IsZero() bool {
	return s.start.IsZero() && s.duration == 0
}

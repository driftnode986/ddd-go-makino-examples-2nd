package domain_test

import (
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
)

var jst = time.FixedZone("JST", 9*60*60)

func at(day, hour int) time.Time {
	return time.Date(2030, time.June, day, hour, 0, 0, 0, jst)
}

func slot(
	t *testing.T, start time.Time, d time.Duration,
) domain.TimeSlot {
	t.Helper()
	s, err := domain.NewTimeSlot(start, d, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	return s
}

// #region same-instant

func TestTimeSlot_SameInstantAcrossZones(t *testing.T) {
	// 同じ瞬間を JST と UTC で渡すと、同じ時間帯になる
	a := slot(t, at(1, 9), time.Hour)
	b := slot(t, at(1, 9).UTC(), time.Hour)
	if !a.Equal(b) || !b.Equal(a) {
		t.Fatal("同じ瞬間の TimeSlot は Equal が true")
	}

	// 営業時間外は、どちらの形で渡しても拒否する
	early := at(1, 8)
	_, errA := domain.NewTimeSlot(early, time.Hour, jst)
	_, errB := domain.NewTimeSlot(early.UTC(), time.Hour, jst)
	if !errors.Is(errA, domain.ErrOutOfBusinessHours) ||
		!errors.Is(errB, domain.ErrOutOfBusinessHours) {
		t.Fatalf("errA=%v errB=%v", errA, errB)
	}
}

// #endregion

// #region business-hours-cases

func TestTimeSlot_BusinessHours(t *testing.T) {
	cases := []struct {
		name    string
		start   time.Time
		d       time.Duration
		wantErr error
	}{
		{"開始は9時から", at(1, 9), time.Hour, nil},
		{"終了は21時まで", at(1, 20), time.Hour, nil},
		{"8時台は不可", at(1, 8), time.Hour,
			domain.ErrOutOfBusinessHours},
		{"21時開始は不可", at(1, 21), time.Hour,
			domain.ErrOutOfBusinessHours},
		{"終了が21時を超える", at(1, 20), 2 * time.Hour,
			domain.ErrOutOfBusinessHours},
		{"日をまたぐ", at(1, 20), 14 * time.Hour,
			domain.ErrOutOfBusinessHours},
		// #endregion
		{"90分は可", at(1, 10), 90 * time.Minute, nil},
		{"長さ0は不可", at(1, 10), 0, domain.ErrInvalidDuration},
		{"負の長さは不可", at(1, 10), -time.Hour,
			domain.ErrInvalidDuration},
		{"開始の分が0でない", at(1, 10).Add(time.Minute),
			time.Hour, domain.ErrNotOnTheHour},
		{"開始の秒が0でない", at(1, 10).Add(time.Second),
			time.Hour, domain.ErrNotOnTheHour},
		{"開始のナノ秒が0でない", at(1, 10).Add(1), time.Hour,
			domain.ErrNotOnTheHour},
		{"ゼロの時刻は不可", time.Time{}, time.Hour,
			domain.ErrInvalidTimeSlot},
		// #region business-hours-run
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := domain.NewTimeSlot(c.start, c.d, jst)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("got %v, want %v", err, c.wantErr)
			}
		})
	}
}

// #endregion

func TestTimeSlot_NilLocationIsAnError(t *testing.T) {
	_, err := domain.NewTimeSlot(at(1, 10), time.Hour, nil)
	if !errors.Is(err, domain.ErrInvalidTimeSlot) {
		t.Fatalf("got %v", err)
	}
}

// 正時の判定は loc の時刻で行う。UTC では 3時30分でも
// +05:30 のゾーンでは9時ちょうどである。
func TestTimeSlot_OnTheHourIsJudgedInLocation(t *testing.T) {
	india := time.FixedZone("IST", 5*60*60+30*60)
	start := time.Date(2030, time.June, 1, 3, 30, 0, 0, time.UTC)
	if _, err := domain.NewTimeSlot(
		start, time.Hour, india); err != nil {
		t.Fatalf("IST: %v", err)
	}
	_, err := domain.NewTimeSlot(start, time.Hour, time.UTC)
	if !errors.Is(err, domain.ErrNotOnTheHour) {
		t.Fatalf("UTC: %v", err)
	}
}

func TestTimeSlot_StartAndEnd(t *testing.T) {
	s := slot(t, at(1, 10), 90*time.Minute)
	if !s.Start().Equal(at(1, 10)) {
		t.Errorf("Start = %v", s.Start())
	}
	want := at(1, 10).Add(90 * time.Minute)
	if !s.End().Equal(want) {
		t.Errorf("End = %v", s.End())
	}
}

func TestTimeSlot_Overlaps(t *testing.T) {
	base := slot(t, at(1, 10), 2*time.Hour)
	cases := []struct {
		name  string
		other domain.TimeSlot
		want  bool
	}{
		{"同じ時間帯", base, true},
		{"前半が重なる", slot(t, at(1, 9), 2*time.Hour), true},
		{"後半が重なる", slot(t, at(1, 11), 2*time.Hour), true},
		{"内側にある", slot(t, at(1, 11), 30*time.Minute), true},
		{"外側にある", slot(t, at(1, 9), 4*time.Hour), true},
		// 半開区間なので、接するだけなら重ならない
		{"直前に接する", slot(t, at(1, 9), time.Hour), false},
		{"直後に接する", slot(t, at(1, 12), time.Hour), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := base.Overlaps(c.other); got != c.want {
				t.Errorf("Overlaps = %v", got)
			}
			if got := c.other.Overlaps(base); got != c.want {
				t.Errorf("逆向きの Overlaps = %v", got)
			}
		})
	}
}

func TestTimeSlot_OverlapsAcrossDates(t *testing.T) {
	a := slot(t, at(1, 10), time.Hour)
	b := slot(t, at(2, 10), time.Hour)
	if a.Overlaps(b) || b.Overlaps(a) {
		t.Fatal("6月1日と6月2日の10時は重ならない")
	}
	if a.Equal(b) {
		t.Fatal("日付が違えば Equal は false")
	}
}

func TestTimeSlot_EqualComparesLength(t *testing.T) {
	a := slot(t, at(1, 10), time.Hour)
	b := slot(t, at(1, 10), 2*time.Hour)
	if a.Equal(b) {
		t.Fatal("長さが違えば Equal は false")
	}
}

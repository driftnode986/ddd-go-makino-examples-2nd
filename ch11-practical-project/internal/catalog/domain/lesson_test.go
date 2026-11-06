package domain_test

import (
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
)

var (
	jst   = time.FixedZone("JST", 9*60*60)
	now   = time.Date(2030, time.May, 1, 0, 0, 0, 0, jst)
	start = time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{now}

func newLesson(t *testing.T, capacity int) *domain.Lesson {
	t.Helper()
	l, err := domain.NewLesson(domain.LessonSpec{
		ID:       ids.NewID[ids.LessonMarker](),
		Title:    "朝のヨガ",
		Start:    start,
		Duration: time.Hour,
		FeeYen:   3000,
		Capacity: capacity,
		Options: []domain.LessonOption{
			{Name: "シューズ", UnitPriceYen: 300},
		},
	})
	if err != nil {
		t.Fatalf("NewLesson: %v", err)
	}
	return l
}

func newBookingID() ids.BookingID {
	return ids.NewID[ids.BookingMarker]()
}

func TestNewLesson_RejectsInvalidValues(t *testing.T) {
	valid := func() domain.LessonSpec {
		return domain.LessonSpec{
			ID:       ids.NewID[ids.LessonMarker](),
			Title:    "朝のヨガ",
			Start:    start,
			Duration: time.Hour,
			FeeYen:   3000,
			Capacity: 1,
		}
	}
	cases := []struct {
		name   string
		change func(*domain.LessonSpec)
	}{
		{"ゼロの ID", func(s *domain.LessonSpec) {
			s.ID = ids.LessonID{}
		}},
		{"空の題名", func(s *domain.LessonSpec) { s.Title = "" }},
		{"ゼロの開始", func(s *domain.LessonSpec) {
			s.Start = time.Time{}
		}},
		{"長さが0", func(s *domain.LessonSpec) { s.Duration = 0 }},
		{"料金が0", func(s *domain.LessonSpec) { s.FeeYen = 0 }},
		{"定員が0", func(s *domain.LessonSpec) { s.Capacity = 0 }},
		{"オプションの名前が空", func(s *domain.LessonSpec) {
			s.Options = []domain.LessonOption{{UnitPriceYen: 1}}
		}},
		{"オプションの価格が負", func(s *domain.LessonSpec) {
			s.Options = []domain.LessonOption{
				{Name: "x", UnitPriceYen: -1},
			}
		}},
		{"オプションの名前が重複", func(s *domain.LessonSpec) {
			s.Options = []domain.LessonOption{
				{Name: "x", UnitPriceYen: 1},
				{Name: "x", UnitPriceYen: 2},
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := valid()
			c.change(&s)
			_, err := domain.NewLesson(s)
			if !errors.Is(err, domain.ErrInvalidLesson) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestReserveSeat_UpToCapacity(t *testing.T) {
	l := newLesson(t, 2)
	for i := range 2 {
		if err := l.ReserveSeat(newBookingID(), clock); err != nil {
			t.Fatalf("%d 席目: %v", i+1, err)
		}
	}
	if l.ReservedSeats() != 2 {
		t.Errorf("ReservedSeats = %d", l.ReservedSeats())
	}
	err := l.ReserveSeat(newBookingID(), clock)
	if !errors.Is(err, domain.ErrLessonFull) {
		t.Fatalf("定員の超過: %v", err)
	}
	if l.ReservedSeats() != 2 {
		t.Errorf("超過の後 ReservedSeats = %d", l.ReservedSeats())
	}
}

func TestReserveSeat_SameBookingTakesOneSeat(t *testing.T) {
	l := newLesson(t, 2)
	id := newBookingID()
	for range 2 {
		if err := l.ReserveSeat(id, clock); err != nil {
			t.Fatalf("ReserveSeat: %v", err)
		}
	}
	if l.ReservedSeats() != 1 {
		t.Errorf("ReservedSeats = %d", l.ReservedSeats())
	}
	if got := l.PullEvents(); len(got) != 1 {
		t.Errorf("events = %v", got)
	}
}

// 満席でも、既に席を持つ予約の再実行は成功のままにする。
func TestReserveSeat_SameBookingWhenFullIsNotRejected(t *testing.T) {
	l := newLesson(t, 1)
	id := newBookingID()
	if err := l.ReserveSeat(id, clock); err != nil {
		t.Fatalf("ReserveSeat: %v", err)
	}
	if err := l.ReserveSeat(id, clock); err != nil {
		t.Fatalf("2回目: %v", err)
	}
}

func TestReserveSeat_RejectsZeroBookingID(t *testing.T) {
	l := newLesson(t, 1)
	err := l.ReserveSeat(ids.BookingID{}, clock)
	if !errors.Is(err, domain.ErrInvalidLesson) {
		t.Fatalf("err = %v", err)
	}
}

func TestReserveSeat_RecordsEvents(t *testing.T) {
	l := newLesson(t, 1)
	first, second := newBookingID(), newBookingID()
	if err := l.ReserveSeat(first, clock); err != nil {
		t.Fatalf("ReserveSeat: %v", err)
	}
	err := l.ReserveSeat(second, clock)
	if !errors.Is(err, domain.ErrLessonFull) {
		t.Fatalf("err = %v", err)
	}
	events := l.PullEvents()
	if len(events) != 2 {
		t.Fatalf("events = %v", events)
	}
	reserved, ok := events[0].(domain.SeatReserved)
	if !ok || !reserved.BookingID.Equal(first) ||
		!reserved.LessonID.Equal(l.ID()) ||
		!reserved.OccurredAt.Equal(now) {
		t.Errorf("events[0] = %#v", events[0])
	}
	rejected, ok := events[1].(domain.SeatRejected)
	if !ok || !rejected.BookingID.Equal(second) ||
		rejected.Reason != domain.RejectLessonFull {
		t.Errorf("events[1] = %#v", events[1])
	}
	if got := l.PullEvents(); len(got) != 0 {
		t.Errorf("2回目の PullEvents = %v", got)
	}
}

func TestReleaseSeat_SecondCallHasNoEffect(t *testing.T) {
	l := newLesson(t, 2)
	kept, released := newBookingID(), newBookingID()
	for _, id := range []ids.BookingID{kept, released} {
		if err := l.ReserveSeat(id, clock); err != nil {
			t.Fatalf("ReserveSeat: %v", err)
		}
	}
	for range 2 {
		if err := l.ReleaseSeat(released); err != nil {
			t.Fatalf("ReleaseSeat: %v", err)
		}
		if l.ReservedSeats() != 1 {
			t.Fatalf("ReservedSeats = %d", l.ReservedSeats())
		}
	}
}

func TestReleaseSeat_UnknownBookingHasNoEffect(t *testing.T) {
	l := newLesson(t, 1)
	if err := l.ReserveSeat(newBookingID(), clock); err != nil {
		t.Fatalf("ReserveSeat: %v", err)
	}
	if err := l.ReleaseSeat(newBookingID()); err != nil {
		t.Fatalf("ReleaseSeat: %v", err)
	}
	if l.ReservedSeats() != 1 {
		t.Errorf("ReservedSeats = %d", l.ReservedSeats())
	}
}

func TestReleaseSeat_MakesRoomForAnother(t *testing.T) {
	l := newLesson(t, 1)
	first := newBookingID()
	if err := l.ReserveSeat(first, clock); err != nil {
		t.Fatalf("ReserveSeat: %v", err)
	}
	if err := l.ReleaseSeat(first); err != nil {
		t.Fatalf("ReleaseSeat: %v", err)
	}
	if err := l.ReserveSeat(newBookingID(), clock); err != nil {
		t.Fatalf("戻した席の確保: %v", err)
	}
}

func TestVersion_NotChangedByMethods(t *testing.T) {
	l := newLesson(t, 1)
	id := newBookingID()
	if err := l.ReserveSeat(id, clock); err != nil {
		t.Fatalf("ReserveSeat: %v", err)
	}
	if err := l.ReleaseSeat(id); err != nil {
		t.Fatalf("ReleaseSeat: %v", err)
	}
	if l.Version() != 0 {
		t.Errorf("Version = %d", l.Version())
	}
}

func TestReconstruct_RoundTripsAndRecordsNoEvent(t *testing.T) {
	l := newLesson(t, 1)
	if err := l.ReserveSeat(newBookingID(), clock); err != nil {
		t.Fatalf("ReserveSeat: %v", err)
	}
	got, err := domain.ReconstructLesson(l.State())
	if err != nil {
		t.Fatalf("ReconstructLesson: %v", err)
	}
	if got.ReservedSeats() != 1 || !got.Equal(l) {
		t.Errorf("got = %+v", got.State())
	}
	if len(got.PullEvents()) != 0 {
		t.Error("再構築でイベントが記録された")
	}
}

func TestReconstruct_RejectsBrokenState(t *testing.T) {
	cases := []struct {
		name   string
		change func(*domain.LessonState)
	}{
		{"定員を超える席", func(s *domain.LessonState) {
			s.Seats = append(s.Seats, newBookingID(), newBookingID())
		}},
		{"重複した席", func(s *domain.LessonState) {
			id := newBookingID()
			s.Capacity = 2
			s.Seats = []ids.BookingID{id, id}
		}},
		{"ゼロの席", func(s *domain.LessonState) {
			s.Seats = []ids.BookingID{{}}
		}},
		{"負の版", func(s *domain.LessonState) { s.Version = -1 }},
		{"ゼロの ID", func(s *domain.LessonState) {
			s.ID = ids.LessonID{}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newLesson(t, 1).State()
			c.change(&s)
			_, err := domain.ReconstructLesson(s)
			if !errors.Is(err, domain.ErrInvalidLesson) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

// State は呼び出し側に中身を共有しない。
func TestState_ReturnsCopy(t *testing.T) {
	l := newLesson(t, 1)
	s := l.State()
	s.Options[0].Name = "変更"
	if l.State().Options[0].Name != "シューズ" {
		t.Error("State の変更が集約に届いた")
	}
}

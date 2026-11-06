package domain_test

import (
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// #region fixed-clock

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// #endregion

// 予約の開始は6月1日10時、現在は5月1日とする。
var (
	lessonStart = at(1, 10)
	earlier     = fixedClock{now: lessonStart.AddDate(0, -1, 0)}
)

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](),
		slot(t, lessonStart, time.Hour),
		yen(t, 3000),
		earlier,
	)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

// inStatus は指定の状態の予約を公開メソッドだけで作る。
func inStatus(t *testing.T, s domain.Status) *domain.Booking {
	t.Helper()
	b := newBooking(t)
	var err error
	switch s {
	case domain.StatusPending:
	case domain.StatusConfirmed:
		err = b.Confirm()
	case domain.StatusRejected:
		err = b.Reject()
	case domain.StatusCancelled:
		if err = b.Confirm(); err == nil {
			err = b.Cancel(earlier)
		}
	default:
		t.Fatalf("未知の状態: %s", s)
	}
	if err != nil {
		t.Fatalf("%s の準備: %v", s, err)
	}
	return b
}

func TestNewBooking_StartsPending(t *testing.T) {
	b := newBooking(t)
	if b.Status() != domain.StatusPending {
		t.Errorf("Status = %s", b.Status())
	}
}

func TestNewBooking_RejectsZeroValues(t *testing.T) {
	id := ids.NewID[ids.BookingMarker]()
	member := ids.NewID[ids.MemberMarker]()
	lesson := ids.NewID[ids.LessonMarker]()
	s := slot(t, lessonStart, time.Hour)
	fee := yen(t, 3000)

	cases := []struct {
		name string
		make func() (*domain.Booking, error)
	}{
		{"予約 ID", func() (*domain.Booking, error) {
			return domain.NewBooking(
				ids.BookingID{}, member, lesson, s, fee, earlier)
		}},
		{"会員 ID", func() (*domain.Booking, error) {
			return domain.NewBooking(
				id, ids.MemberID{}, lesson, s, fee, earlier)
		}},
		{"レッスン ID", func() (*domain.Booking, error) {
			return domain.NewBooking(
				id, member, ids.LessonID{}, s, fee, earlier)
		}},
		{"時間帯", func() (*domain.Booking, error) {
			return domain.NewBooking(
				id, member, lesson, domain.TimeSlot{}, fee, earlier)
		}},
		{"料金", func() (*domain.Booking, error) {
			return domain.NewBooking(
				id, member, lesson, s, domain.Money{}, earlier)
		}},
		{"Clock", func() (*domain.Booking, error) {
			return domain.NewBooking(
				id, member, lesson, s, fee, nil)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := c.make()
			if !errors.Is(err, domain.ErrInvalidBooking) {
				t.Fatalf("err = %v", err)
			}
			if b != nil {
				t.Fatal("エラーのときは nil を返す")
			}
		})
	}
}

// #region past-start

func TestNewBooking_RejectsPastStart(t *testing.T) {
	for _, now := range []time.Time{
		lessonStart, lessonStart.Add(time.Hour),
	} {
		_, err := domain.NewBooking(
			ids.NewID[ids.BookingMarker](),
			ids.NewID[ids.MemberMarker](),
			ids.NewID[ids.LessonMarker](),
			slot(t, lessonStart, time.Hour),
			yen(t, 3000),
			fixedClock{now: now},
		)
		if !errors.Is(err, domain.ErrStartInPast) {
			t.Errorf("now=%s: err = %v", now, err)
		}
	}
}

// #endregion

func TestBooking_ZeroValueMethodsReturnErrors(t *testing.T) {
	b := &domain.Booking{}
	checks := map[string]error{
		"Confirm": b.Confirm(),
		"Reject":  b.Reject(),
		"Cancel":  b.Cancel(earlier),
	}
	for name, err := range checks {
		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// 4つの状態と3つの操作の組み合わせ12通りを確かめる。
func TestBooking_Transitions(t *testing.T) {
	const (
		pending   = domain.StatusPending
		confirmed = domain.StatusConfirmed
		rejected  = domain.StatusRejected
		cancelled = domain.StatusCancelled
	)
	invalid := domain.ErrInvalidTransition
	confirm := func(b *domain.Booking) error { return b.Confirm() }
	reject := func(b *domain.Booking) error { return b.Reject() }
	cancel := func(b *domain.Booking) error {
		return b.Cancel(earlier)
	}

	// #region transitions
	cases := []struct {
		name    string
		from    domain.Status
		op      func(*domain.Booking) error
		want    domain.Status
		wantErr error
	}{
		{"Confirm", pending, confirm, confirmed, nil},
		{"Confirm", confirmed, confirm, confirmed, invalid},
		{"Confirm", rejected, confirm, rejected, invalid},
		{"Confirm", cancelled, confirm, cancelled, invalid},
		{"Reject", pending, reject, rejected, nil},
		{"Reject", confirmed, reject, confirmed, invalid},
		{"Reject", rejected, reject, rejected, invalid},
		{"Reject", cancelled, reject, cancelled, invalid},
		{"Cancel", pending, cancel, pending, invalid},
		{"Cancel", confirmed, cancel, cancelled, nil},
		{"Cancel", rejected, cancel, rejected, invalid},
		{"Cancel", cancelled, cancel, cancelled, invalid},
	}
	// #endregion
	for _, c := range cases {
		t.Run(c.name+"/"+string(c.from), func(t *testing.T) {
			b := inStatus(t, c.from)
			err := c.op(b)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if b.Status() != c.want {
				t.Fatalf("Status = %s, want %s", b.Status(), c.want)
			}
		})
	}
}

// #region pending-cancel

func TestCancel_PendingIsRejected(t *testing.T) {
	b := newBooking(t)
	err := b.Cancel(earlier)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("err = %v", err)
	}
	if b.Status() != domain.StatusPending {
		t.Fatalf("Status = %s", b.Status())
	}
}

// #endregion

// #region within-24h

func TestCancel_Within24Hours(t *testing.T) {
	deadline := lessonStart.Add(-24 * time.Hour)

	b := inStatus(t, domain.StatusConfirmed)
	err := b.Cancel(fixedClock{now: deadline.Add(1)})
	if !errors.Is(err, domain.ErrCancelTooLate) {
		t.Fatalf("期限の直後: err = %v", err)
	}
	if b.Status() != domain.StatusConfirmed {
		t.Fatalf("Status = %s", b.Status())
	}

	// ちょうど24時間前まではキャンセルできる
	b = inStatus(t, domain.StatusConfirmed)
	if err := b.Cancel(fixedClock{now: deadline}); err != nil {
		t.Fatalf("期限ちょうど: err = %v", err)
	}
}

// #endregion

// #region equal

func TestBooking_EqualComparesIdentity(t *testing.T) {
	a := newBooking(t)
	same, err := domain.NewBooking(
		a.ID(), a.MemberID(), a.LessonID(),
		a.Slot(), yen(t, 3000), earlier)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if err := same.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !a.Equal(same) {
		t.Error("同じ ID なら状態が違っても同じ予約である")
	}
	if a.Equal(newBooking(t)) {
		t.Error("ID が違えば別の予約である")
	}
	if a.Equal(nil) {
		t.Error("nil とは等しくない")
	}
}

// #endregion

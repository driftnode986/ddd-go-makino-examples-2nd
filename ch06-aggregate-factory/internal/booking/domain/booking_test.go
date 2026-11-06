package domain_test

import (
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

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

func option(
	t *testing.T, name string, price int64, n int,
) domain.BookingOption {
	t.Helper()
	o, err := domain.NewBookingOption(name, yen(t, price), n)
	if err != nil {
		t.Fatalf("NewBookingOption: %v", err)
	}
	return o
}

func TestNewBooking_StartsPending(t *testing.T) {
	b := newBooking(t)
	if b.Status() != domain.StatusPending {
		t.Errorf("Status = %s", b.Status())
	}
	if !b.Total().Equal(yen(t, 3000)) {
		t.Errorf("Total = %d", b.Total().Amount())
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

func TestBooking_ZeroValueMethodsReturnErrors(t *testing.T) {
	b := &domain.Booking{}
	opt := option(t, "シューズ", 500, 1)
	checks := map[string]error{
		"Confirm":       b.Confirm(),
		"Reject":        b.Reject(),
		"Cancel":        b.Cancel(earlier),
		"AddOption":     b.AddOption(opt, earlier),
		"ApplyDiscount": b.ApplyDiscount(yen(t, 1)),
	}
	for name, err := range checks {
		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// 4つの状態と、状態を見る4つの操作の組み合わせ16通りを確かめる。
func TestBooking_Transitions(t *testing.T) {
	const (
		pending   = domain.StatusPending
		confirmed = domain.StatusConfirmed
		rejected  = domain.StatusRejected
		cancelled = domain.StatusCancelled
	)
	invalid := domain.ErrInvalidTransition
	opt := option(t, "シューズ", 500, 1)
	confirm := func(b *domain.Booking) error { return b.Confirm() }
	reject := func(b *domain.Booking) error { return b.Reject() }
	cancel := func(b *domain.Booking) error {
		return b.Cancel(earlier)
	}
	add := func(b *domain.Booking) error {
		return b.AddOption(opt, earlier)
	}

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
		{"AddOption", pending, add, pending, nil},
		{"AddOption", confirmed, add, confirmed, invalid},
		{"AddOption", rejected, add, rejected, invalid},
		{"AddOption", cancelled, add, cancelled, invalid},
	}
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

func TestAddOption_AfterDeadline(t *testing.T) {
	deadline := lessonStart.Add(-24 * time.Hour)
	late := fixedClock{now: deadline.Add(1)}
	opt := option(t, "シューズ", 500, 1)

	b := newBooking(t)
	err := b.AddOption(opt, late)
	if !errors.Is(err, domain.ErrChangeTooLate) {
		t.Fatalf("AddOption: err = %v", err)
	}
	if len(b.Options()) != 0 {
		t.Fatalf("Options = %v", b.Options())
	}

	if err := b.AddOption(
		opt, fixedClock{now: deadline}); err != nil {
		t.Fatalf("期限ちょうど: err = %v", err)
	}
	err = b.AddOption(option(t, "タオル", 200, 1), late)
	if !errors.Is(err, domain.ErrChangeTooLate) {
		t.Fatalf("2件目: err = %v", err)
	}
	if len(b.Options()) != 1 {
		t.Fatalf("Options = %v", b.Options())
	}
}

func TestAddOption_Limit(t *testing.T) {
	b := newBooking(t)
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		err := b.AddOption(option(t, name, 100, 1), earlier)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	err := b.AddOption(option(t, "f", 100, 1), earlier)
	if !errors.Is(err, domain.ErrOptionLimit) {
		t.Fatalf("err = %v", err)
	}
	if len(b.Options()) != 5 {
		t.Fatalf("Options = %d件", len(b.Options()))
	}
}

func TestAddOption_RejectsDuplicateAndZero(t *testing.T) {
	b := newBooking(t)
	opt := option(t, "シューズ", 500, 1)
	if err := b.AddOption(opt, earlier); err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	err := b.AddOption(option(t, "シューズ", 700, 2), earlier)
	if !errors.Is(err, domain.ErrDuplicateOption) {
		t.Errorf("同じ名前: err = %v", err)
	}
	err = b.AddOption(domain.BookingOption{}, earlier)
	if !errors.Is(err, domain.ErrInvalidOption) {
		t.Errorf("ゼロ値: err = %v", err)
	}
}

func TestNewBookingOption_Validation(t *testing.T) {
	price := yen(t, 500)
	cases := []struct {
		name string
		opt  string
		n    int
	}{
		{"名前なし", "", 1},
		{"数量0", "シューズ", 0},
		{"負の数量", "シューズ", -1},
	}
	for _, c := range cases {
		_, err := domain.NewBookingOption(c.opt, price, c.n)
		if !errors.Is(err, domain.ErrInvalidOption) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	_, err := domain.NewBookingOption(
		"シューズ", yen(t, 1<<62), 4)
	if !errors.Is(err, domain.ErrMoneyOverflow) {
		t.Errorf("あふれ: err = %v", err)
	}
	var zero domain.BookingOption
	if !zero.IsZero() {
		t.Error("var で宣言した明細は IsZero が true")
	}
}

func TestTotal_SumsFeeOptionsAndDiscount(t *testing.T) {
	b := newBooking(t)
	for _, o := range []domain.BookingOption{
		option(t, "シューズ", 500, 2),
		option(t, "タオル", 200, 1),
	} {
		if err := b.AddOption(o, earlier); err != nil {
			t.Fatalf("AddOption: %v", err)
		}
	}
	if !b.Total().Equal(yen(t, 4200)) {
		t.Fatalf("Total = %d", b.Total().Amount())
	}
	if err := b.ApplyDiscount(yen(t, 700)); err != nil {
		t.Fatalf("ApplyDiscount: %v", err)
	}
	if !b.Total().Equal(yen(t, 3500)) {
		t.Fatalf("割引後の Total = %d", b.Total().Amount())
	}
	// 割引は積み上げず、最後の指定で置き換える
	if err := b.ApplyDiscount(yen(t, 200)); err != nil {
		t.Fatalf("ApplyDiscount: %v", err)
	}
	if !b.Total().Equal(yen(t, 4000)) {
		t.Fatalf("置き換え後の Total = %d", b.Total().Amount())
	}
}

func TestTotal_RejectsOverflowingOption(t *testing.T) {
	b := newBooking(t)
	big := option(t, "高額", 1<<62, 1)
	if err := b.AddOption(big, earlier); err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	err := b.AddOption(option(t, "高額2", 1<<62, 1), earlier)
	if !errors.Is(err, domain.ErrMoneyOverflow) {
		t.Fatalf("err = %v", err)
	}
	if len(b.Options()) != 1 {
		t.Fatalf("Options = %d件", len(b.Options()))
	}
}

func TestDiscount_NeverExceedsSubtotal(t *testing.T) {
	b := newBooking(t)
	err := b.ApplyDiscount(yen(t, 3001))
	if !errors.Is(err, domain.ErrDiscountTooLarge) {
		t.Fatalf("料金を超える割引: err = %v", err)
	}
	if err := b.ApplyDiscount(yen(t, 3000)); err != nil {
		t.Fatalf("料金ちょうどの割引: %v", err)
	}
	if !b.Total().IsZero() {
		t.Fatalf("Total = %d", b.Total().Amount())
	}
}

func TestApplyDiscount_Status(t *testing.T) {
	for _, s := range []domain.Status{
		domain.StatusRejected, domain.StatusCancelled,
	} {
		err := inStatus(t, s).ApplyDiscount(yen(t, 100))
		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Errorf("%s: err = %v", s, err)
		}
	}
}

func TestOptions_ReturnsCopy(t *testing.T) {
	b := newBooking(t)
	opt := option(t, "シューズ", 500, 1)
	if err := b.AddOption(opt, earlier); err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	got := b.Options()
	got[0] = option(t, "書き換え", 1, 1)
	if !b.Options()[0].Equal(opt) {
		t.Fatalf("内部の明細が変わった: %v", b.Options())
	}
	st := b.State()
	st.Options[0].Name = "書き換え"
	if b.State().Options[0].Name != "シューズ" {
		t.Fatal("State の明細が内部を共有している")
	}
}

func TestBooking_EqualComparesIdentity(t *testing.T) {
	a := newBooking(t)
	same, err := domain.ReconstructBooking(a.State())
	if err != nil {
		t.Fatalf("ReconstructBooking: %v", err)
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

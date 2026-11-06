// Package querytest は BookingQueries の契約テストである。
// in-memory と PostgreSQL の両方に同じテストを流す。
package querytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// Fixture は契約テストに渡す部品である。
type Fixture struct {
	Repo    domain.BookingRepository
	Tx      application.Transactor
	Queries application.BookingQueries
}

var jst = time.FixedZone("JST", 9*60*60)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, jst),
}

// Run は契約テストを実行する。fixture はテストごとに空の部品を返す。
func Run(t *testing.T, fixture func(t *testing.T) Fixture) {
	t.Helper()
	cases := []struct {
		name string
		run  func(*testing.T, Fixture)
	}{
		{"開始の順に自分の予約だけを返す", testOrderAndMember},
		{"合計は料金と明細から割引を引いた額", testTotal},
		{"すべての状態を返す", testStatuses},
		{"予約が無ければ空のスライス", testEmpty},
		{"キャンセル済みの context はエラー", testCanceled},
		{"有効な予約の時間帯だけを開始の順に返す", testActiveSlots},
		{"有効な予約が無ければ空のスライス", testActiveSlotsEmpty},
		{"時間帯の読み取りも context に従う", testActiveSlotsCanceled},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.run(t, fixture(t)) })
	}
}

func yen(t *testing.T, v int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(v)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}

func booking(
	t *testing.T, member ids.MemberID, day int,
) *domain.Booking {
	t.Helper()
	start := time.Date(2030, time.June, day, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](), member,
		ids.NewID[ids.LessonMarker](), slot, yen(t, 3000), clock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

func (f Fixture) add(t *testing.T, b *domain.Booking) {
	t.Helper()
	err := f.Tx.Do(t.Context(), func(ctx context.Context) error {
		return f.Repo.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
}

func (f Fixture) list(
	t *testing.T, member ids.MemberID,
) []application.BookingView {
	t.Helper()
	views, err := f.Queries.ListByMember(t.Context(), member)
	if err != nil {
		t.Fatalf("ListByMember: %v", err)
	}
	return views
}

func testOrderAndMember(t *testing.T, f Fixture) {
	member := ids.NewID[ids.MemberMarker]()
	later := booking(t, member, 3)
	sooner := booking(t, member, 2)
	f.add(t, later)
	f.add(t, sooner)
	f.add(t, booking(t, ids.NewID[ids.MemberMarker](), 1))

	views := f.list(t, member)
	if len(views) != 2 {
		t.Fatalf("views = %+v", views)
	}
	first, second := views[0], views[1]
	if first.ID != sooner.ID().String() ||
		second.ID != later.ID().String() {
		t.Fatalf("順序が違う: %+v", views)
	}
	if first.LessonID != sooner.LessonID().String() {
		t.Errorf("LessonID = %s", first.LessonID)
	}
	if !first.Start.Equal(sooner.Slot().Start()) ||
		!first.End.Equal(sooner.Slot().End()) {
		t.Errorf("時間帯 = %v, %v", first.Start, first.End)
	}
	if first.Status != "pending" {
		t.Errorf("Status = %s", first.Status)
	}
}

func testTotal(t *testing.T, f Fixture) {
	member := ids.NewID[ids.MemberMarker]()
	b := booking(t, member, 1)
	for _, o := range []struct {
		name string
		yen  int64
		n    int
	}{{"シューズ", 500, 2}, {"タオル", 200, 1}} {
		opt, err := domain.NewBookingOption(
			o.name, yen(t, o.yen), o.n)
		if err != nil {
			t.Fatalf("NewBookingOption: %v", err)
		}
		if err := b.AddOption(opt, clock); err != nil {
			t.Fatalf("AddOption: %v", err)
		}
	}
	if err := b.ApplyDiscount(yen(t, 300)); err != nil {
		t.Fatalf("ApplyDiscount: %v", err)
	}
	f.add(t, b)

	views := f.list(t, member)
	if len(views) != 1 {
		t.Fatalf("views = %+v", views)
	}
	// 3000 + 500*2 + 200 - 300。集約の Total と一致する
	const want = 3900
	if views[0].TotalYen != want || b.Total().Amount() != want {
		t.Errorf("TotalYen = %d, Total = %d",
			views[0].TotalYen, b.Total().Amount())
	}
}

func testStatuses(t *testing.T, f Fixture) {
	member := ids.NewID[ids.MemberMarker]()
	rejected := booking(t, member, 1)
	if err := rejected.Reject(); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	cancelled := booking(t, member, 2)
	if err := cancelled.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if err := cancelled.Cancel(clock); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	confirmed := booking(t, member, 3)
	if err := confirmed.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	f.add(t, rejected)
	f.add(t, cancelled)
	f.add(t, confirmed)
	f.add(t, booking(t, member, 4))

	want := []string{"rejected", "cancelled", "confirmed", "pending"}
	views := f.list(t, member)
	if len(views) != len(want) {
		t.Fatalf("views = %+v", views)
	}
	for i, s := range want {
		if views[i].Status != s {
			t.Errorf("views[%d].Status = %s", i, views[i].Status)
		}
	}
}

func testEmpty(t *testing.T, f Fixture) {
	views := f.list(t, ids.NewID[ids.MemberMarker]())
	if views == nil || len(views) != 0 {
		t.Fatalf("views = %#v", views)
	}
}

func testCanceled(t *testing.T, f Fixture) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	member := ids.NewID[ids.MemberMarker]()
	_, err := f.Queries.ListByMember(ctx, member)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func testActiveSlots(t *testing.T, f Fixture) {
	member := ids.NewID[ids.MemberMarker]()
	rejected := booking(t, member, 1)
	if err := rejected.Reject(); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	cancelled := booking(t, member, 2)
	if err := cancelled.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if err := cancelled.Cancel(clock); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	confirmed := booking(t, member, 3)
	if err := confirmed.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	pending := booking(t, member, 4)
	f.add(t, pending)
	f.add(t, rejected)
	f.add(t, confirmed)
	f.add(t, cancelled)
	f.add(t, booking(t, ids.NewID[ids.MemberMarker](), 3))

	slots, err := f.Queries.ActiveSlotsByMember(t.Context(), member)
	if err != nil {
		t.Fatalf("ActiveSlotsByMember: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("slots = %+v", slots)
	}
	if !slots[0].Equal(confirmed.Slot()) ||
		!slots[1].Equal(pending.Slot()) {
		t.Errorf("slots = %+v", slots)
	}
}

func testActiveSlotsEmpty(t *testing.T, f Fixture) {
	slots, err := f.Queries.ActiveSlotsByMember(
		t.Context(), ids.NewID[ids.MemberMarker]())
	if err != nil {
		t.Fatalf("ActiveSlotsByMember: %v", err)
	}
	if slots == nil || len(slots) != 0 {
		t.Fatalf("slots = %#v", slots)
	}
}

func testActiveSlotsCanceled(t *testing.T, f Fixture) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := f.Queries.ActiveSlotsByMember(
		ctx, ids.NewID[ids.MemberMarker]())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

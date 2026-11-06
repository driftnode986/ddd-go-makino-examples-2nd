package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/fixed"
	"example.com/trainingbooking/internal/booking/infrastructure/memory"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

var (
	jst = time.FixedZone("JST", 9*60*60)
	// 予約はすべて2030年6月で、現在は5月1日とする。
	now = time.Date(2030, time.May, 1, 0, 0, 0, 0, jst)
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func yen(t *testing.T, v int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(v)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}

func terms(t *testing.T) application.LessonTerms {
	t.Helper()
	return application.LessonTerms{
		Fee:      yen(t, 3000),
		Location: jst,
	}
}

func placeInput() application.PlaceBookingInput {
	return application.PlaceBookingInput{
		MemberID: ids.NewID[ids.MemberMarker]().String(),
		LessonID: ids.NewID[ids.LessonMarker]().String(),
		Start:    time.Date(2030, time.June, 1, 10, 0, 0, 0, jst),
		Duration: time.Hour,
	}
}

// memoryDeps はユースケースが使う in-memory の部品である。
type memoryDeps struct {
	repo    *memory.BookingRepository
	queries application.BookingQueries
	outbox  *memory.OutboxWriter
	tx      *memtx.Transactor
	coupons application.Coupons
}

func newMemoryDeps() memoryDeps {
	repo := memory.NewBookingRepository()
	return memoryDeps{
		repo:    repo,
		queries: memory.NewBookingQueries(repo),
		outbox:  memory.NewOutboxWriter(),
		tx:      memtx.NewTransactor(),
		coupons: fixed.NewCoupons(),
	}
}

// addConfirmed は確定済みの予約を保存して ID を返す。
func addConfirmed(t *testing.T, d memoryDeps) ids.BookingID {
	t.Helper()
	in := placeInput()
	id := ids.NewID[ids.BookingMarker]()
	member, err := ids.ParseID[ids.MemberMarker](in.MemberID)
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	lesson, err := ids.ParseID[ids.LessonMarker](in.LessonID)
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	slot, err := domain.NewTimeSlot(in.Start, in.Duration, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	b, err := domain.NewBooking(id, member, lesson, slot,
		yen(t, 3000), fixedClock{now})
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if err := b.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	err = d.tx.Do(t.Context(), func(ctx context.Context) error {
		return d.repo.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return id
}

// wantKind は err が kind の ApplicationError であることを確かめる。
func wantKind(
	t *testing.T, err error, kind application.Kind,
) *application.ApplicationError {
	t.Helper()
	got, ok := errors.AsType[*application.ApplicationError](err)
	if !ok {
		t.Fatalf("ApplicationError でない: %v", err)
	}
	if got.Kind != kind {
		t.Fatalf("Kind = %s, want %s (%v)", got.Kind, kind, err)
	}
	if got.Message == "" {
		t.Error("Message が空")
	}
	return got
}

// failingOutbox は常に失敗する OutboxWriter である。
type failingOutbox struct{ err error }

func (o failingOutbox) Append(
	context.Context, ...domain.DomainEvent,
) error {
	return o.err
}

// failingRepository は Add と Save で err を返す。
type failingRepository struct {
	domain.BookingRepository
	err error
}

func (r failingRepository) Add(
	context.Context, *domain.Booking,
) error {
	return r.err
}

func (r failingRepository) Save(
	context.Context, *domain.Booking,
) error {
	return r.err
}

// discardOutbox は何もしない OutboxWriter である。
type discardOutbox struct{}

func (discardOutbox) Append(
	context.Context, ...domain.DomainEvent,
) error {
	return nil
}

// staleQueries は、同時に来た予約がまだ見えていない読み取りである。
type staleQueries struct{ application.BookingQueries }

func (staleQueries) ActiveSlotsByMember(
	context.Context, ids.MemberID,
) ([]domain.TimeSlot, error) {
	return nil, nil
}

// failingQueries は時間帯の読み取りで err を返す。
type failingQueries struct {
	application.BookingQueries
	err error
}

func (q failingQueries) ActiveSlotsByMember(
	context.Context, ids.MemberID,
) ([]domain.TimeSlot, error) {
	return nil, q.err
}

// failingCoupons はどのコードでも err を返す。
type failingCoupons struct{ err error }

func (c failingCoupons) Find(
	context.Context, string,
) (domain.Coupon, error) {
	return domain.Coupon{}, c.err
}

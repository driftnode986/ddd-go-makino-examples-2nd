//go:build integration

package noversion_test

import (
	"context"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"example.com/trainingbooking/steps/noversion"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC),
}

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](), slot, fee, clock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

func option(t *testing.T, name string) domain.BookingOption {
	t.Helper()
	price, err := domain.NewMoney(500)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	o, err := domain.NewBookingOption(name, price, 1)
	if err != nil {
		t.Fatalf("NewBookingOption: %v", err)
	}
	return o
}

func names(b *domain.Booking) []string {
	var got []string
	for _, o := range b.State().Options {
		got = append(got, o.Name)
	}
	return got
}

// #region lost-update

func TestSecondSaveOverwritesFirst(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	ctx := t.Context()
	b := newBooking(t)
	save := func(fn func(context.Context) error) error {
		return tx.Do(ctx, fn)
	}
	err := save(func(ctx context.Context) error {
		return repo.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	first, _ := repo.FindByID(ctx, b.ID())
	second, _ := repo.FindByID(ctx, b.ID())
	_ = first.AddOption(option(t, "シューズ"), clock)
	_ = second.AddOption(option(t, "タオル"), clock)
	t.Logf("1人目の保存: %v", save(func(ctx context.Context) error {
		return noversion.Save(ctx, first)
	}))
	t.Logf("2人目の保存: %v", save(func(ctx context.Context) error {
		return noversion.Save(ctx, second)
	}))
	got, _ := repo.FindByID(ctx, b.ID())
	t.Logf("保存された明細: %v", names(got))
}

// #endregion

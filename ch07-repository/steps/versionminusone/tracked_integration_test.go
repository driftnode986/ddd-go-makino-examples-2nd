//go:build integration

package versionminusone_test

import (
	"context"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	vm "example.com/trainingbooking/steps/versionminusone"
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

// #region cases

func TestVersionInMethods(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	load := func() *vm.Tracked {
		b := newBooking(t)
		err := tx.Do(t.Context(), func(ctx context.Context) error {
			return repo.Add(ctx, b)
		})
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		return &vm.Tracked{Booking: b, Version: b.Version()}
	}
	save := func(b *vm.Tracked) error {
		return tx.Do(t.Context(), func(ctx context.Context) error {
			return vm.Save(ctx, b)
		})
	}

	once := load()
	_ = once.Confirm()
	t.Logf("確定だけ: %v", save(once))

	twice := load()
	_ = twice.Confirm()
	_ = twice.Cancel(clock)
	t.Logf("確定してキャンセル: %v", save(twice))
	t.Logf("変更なし: %v", save(load()))
}

// #endregion

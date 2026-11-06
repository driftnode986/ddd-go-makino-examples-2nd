//go:build integration

package twoselects_test

import (
	"context"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"example.com/trainingbooking/steps/twoselects"
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
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		ids.NewID[ids.MemberMarker](),
		ids.NewID[ids.LessonMarker](), slot, yen(t, 3000), clock)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

func yen(t *testing.T, n int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(n)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}

// #region mixed

func TestTwoSelectsMixVersions(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	ctx := t.Context()
	b := newBooking(t)
	err := tx.Do(ctx, func(ctx context.Context) error {
		return repo.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	t.Logf("更新の前の合計: %d", b.Total().Amount())

	// 2つの SELECT の間に、明細と割引を足す更新がコミットされる
	got, err := twoselects.Find(ctx, pool,
		b.ID(), b.MemberID(), b.LessonID(), func() {
			addShoesAndDiscount(t, repo, tx, b.ID())
		})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	t.Logf("2回に分けて読んだ合計: %d", got.Total().Amount())

	after, err := repo.FindByID(ctx, b.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	t.Logf("更新の後の合計: %d", after.Total().Amount())
}

// #endregion

func addShoesAndDiscount(
	t *testing.T, repo *postgres.BookingRepository,
	tx *pgtx.Transactor, id ids.BookingID,
) {
	t.Helper()
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		loaded, err := repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		shoes, err := domain.NewBookingOption(
			"シューズ", yen(t, 1000), 1)
		if err != nil {
			return err
		}
		if err := loaded.AddOption(shoes, clock); err != nil {
			return err
		}
		err = loaded.ApplyDiscount(yen(t, 500))
		if err != nil {
			return err
		}
		return repo.Save(ctx, loaded)
	})
	if err != nil {
		t.Fatalf("更新: %v", err)
	}
}

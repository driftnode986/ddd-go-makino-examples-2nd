//go:build integration

package autotx_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"example.com/trainingbooking/steps/autotx"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

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
		ids.NewID[ids.LessonMarker](), slot, fee,
		fixedClock{now: start.AddDate(0, -1, 0)})
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

// #region outer-ctx

func TestOuterContextCommitsSeparately(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	tx := pgtx.NewTransactor(pool)
	repo := autotx.Repository{
		BookingRepository: postgres.NewBookingRepository(pool),
		Tx:                tx,
	}
	b := newBooking(t)
	boom := errors.New("後の処理の失敗")

	outer := t.Context()
	err := tx.Do(outer, func(context.Context) error {
		if err := repo.Add(outer, b); err != nil {
			return err // Tx の ctx でなく外側の ctx を渡した
		}
		return boom
	})
	t.Logf("Do: %v", err)
	_, err = repo.FindByID(outer, b.ID())
	t.Logf("読み戻し: %v", err)
}

// #endregion

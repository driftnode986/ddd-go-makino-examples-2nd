package booking_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/booking"
)

// @MockBean の代わりに、小さい型を手で書いて渡す。
type directTx struct{}

func (directTx) Do(
	ctx context.Context, fn func(context.Context) error,
) error {
	return fn(ctx)
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type recorder struct{ events []booking.Event }

func (r *recorder) Append(
	_ context.Context, events ...booking.Event,
) error {
	r.events = append(r.events, events...)
	return nil
}

func saved(
	t *testing.T, repo booking.Repository,
) *booking.Booking {
	t.Helper()
	fee, err := booking.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b := booking.NewBooking(booking.NewBookingID(), fee)
	if err := repo.Add(t.Context(), b); err != nil {
		t.Fatalf("Add: %v", err)
	}
	return b
}

func TestCancelBooking_RecordsEvent(t *testing.T) {
	repo := booking.NewMemoryRepository()
	b := saved(t, repo)
	events := &recorder{}
	u := booking.NewCancelBooking(repo, events, directTx{},
		fixedClock{time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)})

	if err := u.Execute(t.Context(), b.ID()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := repo.FindByID(t.Context(), b.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status() != booking.StatusCancelled {
		t.Fatalf("Status = %s", got.Status())
	}
	if len(events.events) != 1 {
		t.Fatalf("events = %v", events.events)
	}
}

// 例外の種類の判定は errors.Is で行う。
func TestCancelBooking_ErrorsAreValues(t *testing.T) {
	repo := booking.NewMemoryRepository()
	b := saved(t, repo)
	u := booking.NewCancelBooking(repo, &recorder{}, directTx{},
		fixedClock{})
	if err := u.Execute(t.Context(), b.ID()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	err := u.Execute(t.Context(), b.ID())
	if !errors.Is(err, booking.ErrInvalidTransition) {
		t.Fatalf("2回目: %v", err)
	}
	err = u.Execute(t.Context(), booking.NewBookingID())
	if !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("無い予約: %v", err)
	}
}

// @Version と同じく、古い版の保存は競合になる。
func TestRepository_VersionConflict(t *testing.T) {
	repo := booking.NewMemoryRepository()
	b := saved(t, repo)
	first, err := repo.FindByID(t.Context(), b.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	second, err := repo.FindByID(t.Context(), b.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if err := repo.Save(t.Context(), first); err != nil {
		t.Fatalf("Save: %v", err)
	}
	err = repo.Save(t.Context(), second)
	if !errors.Is(err, booking.ErrVersionConflict) {
		t.Fatalf("err = %v", err)
	}
}

func TestMoney_RejectsNegative(t *testing.T) {
	_, err := booking.NewMoney(-1)
	if !errors.Is(err, booking.ErrInvalidMoney) {
		t.Fatalf("err = %v", err)
	}
}

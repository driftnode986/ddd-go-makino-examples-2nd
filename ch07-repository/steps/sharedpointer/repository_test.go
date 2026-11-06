package sharedpointer_test

import (
	"context"
	"os"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/domain/repositorytest"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn/memtx"
	"example.com/trainingbooking/steps/sharedpointer"
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

// #region unsaved

func TestUnsavedChangeIsVisible(t *testing.T) {
	repo := sharedpointer.New()
	tx := memtx.NewTransactor()
	ctx := t.Context()
	b := newBooking(t)
	err := tx.Do(ctx, func(ctx context.Context) error {
		return repo.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	loaded, _ := repo.FindByID(ctx, b.ID())
	if err := loaded.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	// Save を呼ばずに読み直す
	again, _ := repo.FindByID(ctx, b.ID())
	t.Logf("読み直した状態: %s", again.Status())
	t.Logf("同じポインタか: %v", loaded == again)
}

// #endregion

// #region contract

// 契約テストはこの実装で失敗する。go test ./... を緑に保つため、
// 環境変数を付けたときだけ動かす。
func TestContractFails(t *testing.T) {
	if os.Getenv("SHOW_CONTRACT_FAILURES") == "" {
		t.Skip("SHOW_CONTRACT_FAILURES=1 で実行する")
	}
	repositorytest.Run(t, func(*testing.T) (
		domain.BookingRepository, repositorytest.Transactor,
	) {
		return sharedpointer.New(), memtx.NewTransactor()
	})
}

// #endregion

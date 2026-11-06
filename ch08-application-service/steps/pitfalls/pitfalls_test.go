package pitfalls_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/fixed"
	"example.com/trainingbooking/internal/booking/infrastructure/memory"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/logging"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

// #region separate-tx

func TestSeparateTxLeavesBookingWithoutEvent(t *testing.T) {
	repo := memory.NewBookingRepository()
	tx := memtx.NewTransactor()
	outbox := memory.NewOutboxWriter()
	failing := &flakyOutbox{next: outbox, fails: 1}
	b := newBooking(t)

	ctx := t.Context()
	err := tx.Do(ctx, func(ctx context.Context) error {
		return repo.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	err = tx.Do(ctx, func(ctx context.Context) error {
		return failing.Append(ctx, b.PullEvents()...)
	})
	t.Logf("2つ目の Tx のエラー: %v", err)

	got, err := repo.FindByID(ctx, b.ID())
	if err != nil {
		t.Fatalf("予約が残っていない: %v", err)
	}
	t.Logf("予約: %s / イベント: %d 件",
		got.Status(), len(outbox.Events()))
	if len(outbox.Events()) != 0 {
		t.Fatal("イベントが書かれた")
	}
}

// #endregion

// #region retry-same

func TestRetryWithSameBookingLosesEvent(t *testing.T) {
	repo := memory.NewBookingRepository()
	tx := memtx.NewTransactor()
	outbox := memory.NewOutboxWriter()
	flaky := &flakyOutbox{next: outbox, fails: 1}
	b := newBooking(t)

	save := func() error {
		return tx.Do(t.Context(), func(ctx context.Context) error {
			if err := repo.Add(ctx, b); err != nil {
				return err
			}
			return flaky.Append(ctx, b.PullEvents()...)
		})
	}
	t.Logf("1回目: %v", save())
	err := save()
	t.Logf("2回目: %v / イベント: %d 件", err, len(outbox.Events()))
	if err != nil || len(outbox.Events()) != 0 {
		t.Fatalf("err = %v, events = %v", err, outbox.Events())
	}
}

// #endregion

// #region plain-log

func TestStandardHandlerIgnoresContext(t *testing.T) {
	// 時刻は実行ごとに変わるので出力から外す
	opts := &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}
	var plain, wrapped bytes.Buffer
	ctx := logging.WithRequestID(t.Context(), "req-1")

	slog.New(slog.NewJSONHandler(&plain, opts)).
		InfoContext(ctx, "予約を受け付けた")
	slog.New(logging.NewContextHandler(
		slog.NewJSONHandler(&wrapped, opts))).
		InfoContext(ctx, "予約を受け付けた")

	t.Logf("標準: %s", plain.String())
	t.Logf("自前: %s", wrapped.String())
	if bytes.Contains(plain.Bytes(), []byte("req-1")) ||
		!bytes.Contains(wrapped.Bytes(), []byte("req-1")) {
		t.Fatal("想定と違う")
	}
}

// #endregion

// failingRepository は Add で err を返す。
type failingRepository struct {
	domain.BookingRepository
	err error
}

func (r failingRepository) Add(
	context.Context, *domain.Booking,
) error {
	return r.err
}

func TestCauseStaysOutOfMessage(t *testing.T) {
	db := errors.New("connection refused: secret-host:5432")
	repo := memory.NewBookingRepository()
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	place := application.NewPlaceBooking(
		failingRepository{repo, fmt.Errorf("add: %w", db)},
		memory.NewBookingQueries(repo), memory.NewOutboxWriter(),
		memtx.NewTransactor(), clock, fixed.NewCoupons(),
		application.LessonTerms{
			Start:    start,
			Duration: time.Hour,
			Fee:      fee,
			Location: jst,
		})
	in := application.PlaceBookingInput{
		MemberID: ids.NewID[ids.MemberMarker]().String(),
		LessonID: ids.NewID[ids.LessonMarker]().String(),
	}
	_, err = place.Execute(t.Context(), in)
	got, ok := errors.AsType[*application.ApplicationError](err)
	if !ok {
		t.Fatalf("err = %v", err)
	}
	t.Logf("Kind: %s", got.Kind)
	t.Logf("Message: %s", got.Message)
	t.Logf("Error(): %v", err)
	t.Logf("errors.Is(err, db): %v", errors.Is(err, db))
	if !errors.Is(err, db) {
		t.Fatal("原因をたどれない")
	}
}

package application_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/outbox"
)

func newHandler(
	d memoryDeps, repo domain.BookingRepository,
) *application.SeatResultHandler {
	return application.NewSeatResultHandler(
		repo, outbox.NewMemoryProcessedEvents(), d.tx)
}

// addPending は pending の予約を保存して ID を返す。
func addPending(t *testing.T, d memoryDeps) ids.BookingID {
	t.Helper()
	out, err := newPlace(t, d).Execute(t.Context(), placeInput())
	if err != nil {
		t.Fatalf("PlaceBooking: %v", err)
	}
	id, err := ids.ParseID[ids.BookingMarker](out.BookingID)
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	return id
}

func statusOf(
	t *testing.T, d memoryDeps, id ids.BookingID,
) domain.Status {
	t.Helper()
	got, err := d.repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	return got.Status()
}

func TestSeatReserved_ConfirmsBooking(t *testing.T) {
	d := newMemoryDeps()
	id := addPending(t, d)
	h := newHandler(d, d.repo)
	err := h.OnSeatReserved(t.Context(), uuid.New().String(),
		contract.SeatReserved{BookingID: id.String()})
	if err != nil {
		t.Fatalf("OnSeatReserved: %v", err)
	}
	if got := statusOf(t, d, id); got != domain.StatusConfirmed {
		t.Errorf("Status = %s", got)
	}
}

func TestSeatRejected_RejectsBooking(t *testing.T) {
	d := newMemoryDeps()
	id := addPending(t, d)
	h := newHandler(d, d.repo)
	err := h.OnSeatRejected(t.Context(), uuid.New().String(),
		contract.SeatRejected{BookingID: id.String()})
	if err != nil {
		t.Fatalf("OnSeatRejected: %v", err)
	}
	if got := statusOf(t, d, id); got != domain.StatusRejected {
		t.Errorf("Status = %s", got)
	}
}

// 同じイベントの再配信は何もしない。処理済みの判定が無ければ
// Confirm の2回目が ErrInvalidTransition になる。
func TestSeatReserved_RedeliveryIsNoOp(t *testing.T) {
	d := newMemoryDeps()
	id := addPending(t, d)
	h := newHandler(d, d.repo)
	event := contract.SeatReserved{BookingID: id.String()}
	eventID := uuid.New().String()
	for range 2 {
		err := h.OnSeatReserved(t.Context(), eventID, event)
		if err != nil {
			t.Fatalf("OnSeatReserved: %v", err)
		}
	}
	got, err := d.repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Version() != 1 {
		t.Errorf("Version = %d", got.Version())
	}
}

// 保存に失敗すれば処理済みの記録も戻り、再配信で処理できる。
func TestSeatReserved_FailedSaveCanBeRedelivered(t *testing.T) {
	d := newMemoryDeps()
	id := addPending(t, d)
	processed := outbox.NewMemoryProcessedEvents()
	boom := errors.New("boom")
	failing := application.NewSeatResultHandler(
		failingRepository{d.repo, boom}, processed, d.tx)
	event := contract.SeatReserved{BookingID: id.String()}
	eventID := uuid.New().String()

	err := failing.OnSeatReserved(t.Context(), eventID, event)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if got := statusOf(t, d, id); got != domain.StatusPending {
		t.Fatalf("Status = %s", got)
	}

	h := application.NewSeatResultHandler(d.repo, processed, d.tx)
	err = h.OnSeatReserved(t.Context(), eventID, event)
	if err != nil {
		t.Fatalf("再配信: %v", err)
	}
	if got := statusOf(t, d, id); got != domain.StatusConfirmed {
		t.Errorf("Status = %s", got)
	}
}

func TestSeatReserved_BrokenIDIsError(t *testing.T) {
	d := newMemoryDeps()
	h := newHandler(d, d.repo)
	err := h.OnSeatReserved(context.Background(),
		uuid.New().String(), contract.SeatReserved{BookingID: "x"})
	if !errors.Is(err, ids.ErrInvalidID) {
		t.Fatalf("err = %v", err)
	}
}

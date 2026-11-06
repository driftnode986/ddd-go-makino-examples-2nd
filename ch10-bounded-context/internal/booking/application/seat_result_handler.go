package application

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/ids"
)

// Subscriber は処理済みの記録に使う購読者の名前である。
const Subscriber = "booking"

// SeatResultHandler は席の確保の結果を購読して、予約の状態を進める。
type SeatResultHandler struct {
	repo      domain.BookingRepository
	processed ProcessedEvents
	tx        Transactor
}

// NewSeatResultHandler は SeatResultHandler を返す。
func NewSeatResultHandler(
	repo domain.BookingRepository,
	processed ProcessedEvents,
	tx Transactor,
) *SeatResultHandler {
	return &SeatResultHandler{repo, processed, tx}
}

// OnSeatReserved は予約を確定する。
func (h *SeatResultHandler) OnSeatReserved(
	ctx context.Context, eventID string, event contract.SeatReserved,
) error {
	return h.apply(ctx, eventID, event.BookingID,
		(*domain.Booking).Confirm)
}

// OnSeatRejected は予約を拒否の状態にする。
func (h *SeatResultHandler) OnSeatRejected(
	ctx context.Context, eventID string, event contract.SeatRejected,
) error {
	return h.apply(ctx, eventID, event.BookingID,
		(*domain.Booking).Reject)
}

// apply は処理済みの記録と集約の保存を同じ Tx で行う。途中で止まれ
// ばどちらも戻るので、再配信で処理をやり直せる。Confirm と Reject は
// イベントを記録しないので、結果のイベントは書かない。
func (h *SeatResultHandler) apply(
	ctx context.Context,
	eventID, bookingID string,
	change func(*domain.Booking) error,
) error {
	id, err := ids.ParseID[ids.BookingMarker](bookingID)
	if err != nil {
		return err
	}
	return h.tx.Do(ctx, func(ctx context.Context) error {
		first, err := h.processed.MarkProcessed(
			ctx, Subscriber, eventID)
		if err != nil || !first {
			return err
		}
		b, err := h.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := change(b); err != nil {
			return err
		}
		return h.repo.Save(ctx, b)
	})
}

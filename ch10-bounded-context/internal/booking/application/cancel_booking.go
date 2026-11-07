package application

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// CancelBookingInput はキャンセルの入力である。
type CancelBookingInput struct {
	BookingID string
}

// CancelBookingOutput はキャンセルの結果である。
type CancelBookingOutput struct {
	BookingID string
	Status    string
}

// CancelBooking は確定済みの予約をキャンセルする。
type CancelBooking struct {
	repo   domain.BookingRepository
	outbox OutboxWriter
	tx     Transactor
	clock  domain.Clock
}

// NewCancelBooking は CancelBooking を返す。
func NewCancelBooking(
	repo domain.BookingRepository,
	outbox OutboxWriter,
	tx Transactor,
	clock domain.Clock,
) *CancelBooking {
	return &CancelBooking{repo, outbox, tx, clock}
}

// Execute は ID を値にしてキャンセルし、エラーを変換する。
func (u *CancelBooking) Execute(
	ctx context.Context, in CancelBookingInput,
) (CancelBookingOutput, error) {
	id, err := ids.ParseID[ids.BookingMarker](in.BookingID)
	if err != nil {
		return CancelBookingOutput{},
			invalid("予約 ID の形式が正しくありません", err)
	}
	status, err := u.cancel(ctx, id)
	if err != nil {
		return CancelBookingOutput{}, translate(err)
	}
	return CancelBookingOutput{
		BookingID: id.String(),
		Status:    string(status),
	}, nil
}

// #region cancel

// cancel は予約を読み、キャンセルして保存し、同じ Tx で
// BookingCancelled を outbox に書く。失敗した集約は捨てる。
func (u *CancelBooking) cancel(
	ctx context.Context, id ids.BookingID,
) (domain.Status, error) {
	var status domain.Status
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		b, err := u.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := b.Cancel(u.clock); err != nil {
			return err
		}
		if err := u.repo.Save(ctx, b); err != nil {
			return err
		}
		status = b.Status()
		return u.outbox.Append(ctx, b.PullEvents()...)
	})
	return status, err
}

// #endregion

package application

import (
	"context"

	bookingcontract "example.com/trainingbooking/internal/booking/contract"
	"example.com/trainingbooking/internal/ids"
)

// BookingEventHandler は Booking の統合イベントを購読する。
type BookingEventHandler struct {
	reserve *ReserveSeat
	release *ReleaseSeat
}

// NewBookingEventHandler は BookingEventHandler を返す。
func NewBookingEventHandler(
	reserve *ReserveSeat, release *ReleaseSeat,
) *BookingEventHandler {
	return &BookingEventHandler{reserve, release}
}

// #region on-booking-placed

// OnBookingPlaced は席を確保する。満席とレッスンが無いことは、拒否の
// イベントをコミットした成功である。エラーだけを配信の失敗にする。
func (h *BookingEventHandler) OnBookingPlaced(
	ctx context.Context,
	eventID string,
	event bookingcontract.BookingPlaced,
) error {
	lesson, booking, err := parse(event.LessonID, event.BookingID)
	if err != nil {
		return err
	}
	_, err = h.reserve.Execute(ctx, ReserveSeatInput{
		EventID:   eventID,
		LessonID:  lesson,
		BookingID: booking,
	})
	return err
}

// #endregion

// OnBookingCancelled は席を戻す。
func (h *BookingEventHandler) OnBookingCancelled(
	ctx context.Context,
	eventID string,
	event bookingcontract.BookingCancelled,
) error {
	lesson, booking, err := parse(event.LessonID, event.BookingID)
	if err != nil {
		return err
	}
	return h.release.Execute(ctx, ReleaseSeatInput{
		EventID:   eventID,
		LessonID:  lesson,
		BookingID: booking,
	})
}

func parse(
	lessonID, bookingID string,
) (ids.LessonID, ids.BookingID, error) {
	lesson, err := ids.ParseID[ids.LessonMarker](lessonID)
	if err != nil {
		return ids.LessonID{}, ids.BookingID{}, err
	}
	booking, err := ids.ParseID[ids.BookingMarker](bookingID)
	if err != nil {
		return ids.LessonID{}, ids.BookingID{}, err
	}
	return lesson, booking, nil
}

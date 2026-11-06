// Package presentation は Booking の HTTP の入口である。
package presentation

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/ids"
)

// #region use-case-interfaces

// placeBooking は POST /bookings が使うユースケースである。
type placeBooking interface {
	Execute(
		ctx context.Context, in application.PlaceBookingInput,
	) (application.PlaceBookingOutput, error)
}

// cancelBooking は POST /bookings/{id}/cancel が使う
// ユースケースである。
type cancelBooking interface {
	Execute(
		ctx context.Context, in application.CancelBookingInput,
	) (application.CancelBookingOutput, error)
}

// memberBookings は GET /members/{id}/bookings が使う
// 読み取りである。
type memberBookings interface {
	ListByMember(
		ctx context.Context, member ids.MemberID,
	) ([]application.BookingView, error)
}

// #endregion

// Deps は NewRouter が受け取る部品である。
type Deps struct {
	Place   placeBooking
	Cancel  cancelBooking
	Queries memberBookings
	Logger  *slog.Logger
	// Location は応答の時刻を表すタイムゾーンである。
	Location *time.Location
	// Timeout は1リクエストの処理の期限である。
	Timeout time.Duration
}

// #region new-router

// NewRouter は経路とミドルウェアを組み立てる。外側から順に
// request_id の付与、アクセスログ、panic の回復、期限である。
// この順なら panic の記録にも 500 のアクセスログにも
// request_id が付く。
func NewRouter(d Deps) http.Handler {
	h := &handlers{
		place:   d.Place,
		cancel:  d.Cancel,
		queries: d.Queries,
		logger:  d.Logger,
		loc:     d.Location,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /bookings", h.postBooking)
	mux.HandleFunc("POST /bookings/{id}/cancel", h.postCancel)
	mux.HandleFunc("GET /members/{id}/bookings", h.getBookings)
	return withRequestID(withAccessLog(d.Logger,
		withRecovery(h, withTimeout(d.Timeout, mux))))
}

// #endregion

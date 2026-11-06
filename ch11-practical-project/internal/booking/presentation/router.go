// Package presentation は Booking の HTTP の入口である。
package presentation

import (
	"context"
	"log/slog"
	"net/http"

	"example.com/trainingbooking/internal/booking/application"
)

// placeBooking は POST /bookings が使うユースケースである。
type placeBooking interface {
	Execute(
		ctx context.Context, in application.PlaceBookingInput,
	) (application.PlaceBookingOutput, error)
}

// cancelBooking は POST /bookings/{id}/cancel が使うユースケースである。
type cancelBooking interface {
	Execute(
		ctx context.Context, in application.CancelBookingInput,
	) (application.CancelBookingOutput, error)
}

// Deps は NewRouter が受け取る部品である。
type Deps struct {
	Place   placeBooking
	Cancel  cancelBooking
	Queries application.BookingQueries
	Logger  *slog.Logger
}

// NewRouter は経路とミドルウェアを組み立てる。外側から順に
// request_id の付与、アクセスログ、panic の回復である。
// この順なら panic の記録にも 500 のアクセスログにも request_id が付く。
func NewRouter(d Deps) http.Handler {
	h := &handlers{
		place:   d.Place,
		cancel:  d.Cancel,
		queries: d.Queries,
		logger:  d.Logger,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /bookings", h.postBooking)
	mux.HandleFunc("POST /bookings/{id}/cancel", h.postCancel)
	mux.HandleFunc("GET /members/{id}/bookings", h.getBookings)
	return withRequestID(
		withAccessLog(d.Logger, withRecovery(h, mux)))
}

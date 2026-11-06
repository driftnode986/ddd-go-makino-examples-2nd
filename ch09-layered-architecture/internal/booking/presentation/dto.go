package presentation

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/trainingbooking/internal/booking/application"
)

const (
	maxBodyBytes       = 1 << 20
	maxDurationMinutes = 24 * 60
)

// placeBookingRequest は POST /bookings の本文である。
type placeBookingRequest struct {
	MemberID        string    `json:"member_id"`
	LessonID        string    `json:"lesson_id"`
	Start           time.Time `json:"start"`
	DurationMinutes int       `json:"duration_minutes"`
}

func (r placeBookingRequest) toInput() (
	application.PlaceBookingInput, error,
) {
	// 分から Duration への変換であふれないように先に止める
	if r.DurationMinutes > maxDurationMinutes {
		return application.PlaceBookingInput{},
			errors.New("duration_minutes is too large")
	}
	return application.PlaceBookingInput{
		MemberID: r.MemberID,
		LessonID: r.LessonID,
		Start:    r.Start,
		Duration: time.Duration(r.DurationMinutes) * time.Minute,
	}, nil
}

// bookingResponse は予約とキャンセルの結果である。
type bookingResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// bookingItem は一覧の1件である。
type bookingItem struct {
	ID       string    `json:"id"`
	LessonID string    `json:"lesson_id"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Status   string    `json:"status"`
	TotalYen int64     `json:"total_yen"`
}

// bookingListResponse は GET /members/{id}/bookings の本文である。
type bookingListResponse struct {
	Bookings []bookingItem `json:"bookings"`
}

func toListResponse(
	views []application.BookingView,
) bookingListResponse {
	items := make([]bookingItem, len(views))
	for i, v := range views {
		items[i] = bookingItem{
			ID:       v.ID,
			LessonID: v.LessonID,
			Start:    v.Start,
			End:      v.End,
			Status:   v.Status,
			TotalYen: v.TotalYen,
		}
	}
	return bookingListResponse{Bookings: items}
}

// errorResponse は失敗したときの本文である。
type errorResponse struct {
	Kind    application.Kind `json:"kind"`
	Message string           `json:"message"`
}

// decodeJSON は本文を1つの JSON の値として読む。
// 知らない項目と2つ目の値はエラーにする。
func decodeJSON(
	w http.ResponseWriter, r *http.Request, dst any,
) error {
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("body has more than one JSON value")
	}
	return nil
}

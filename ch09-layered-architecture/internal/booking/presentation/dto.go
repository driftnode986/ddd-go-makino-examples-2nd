package presentation

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/trainingbooking/internal/booking/application"
)

const maxBodyBytes = 1 << 20

// optionRequest は選んだオプションと数量である。
type optionRequest struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// #region place-request

// placeBookingRequest は POST /bookings の本文である。
// 時間帯と料金はレッスンから取るので受け取らない。
type placeBookingRequest struct {
	MemberID   string          `json:"member_id"`
	LessonID   string          `json:"lesson_id"`
	Options    []optionRequest `json:"options"`
	CouponCode string          `json:"coupon_code"`
}

// #endregion

func (r placeBookingRequest) input() application.PlaceBookingInput {
	options := make([]application.OptionInput, len(r.Options))
	for i, o := range r.Options {
		options[i] = application.OptionInput{
			Name:     o.Name,
			Quantity: o.Quantity,
		}
	}
	return application.PlaceBookingInput{
		MemberID:   r.MemberID,
		LessonID:   r.LessonID,
		Options:    options,
		CouponCode: r.CouponCode,
	}
}

// bookingResponse は予約とキャンセルの結果である。
type bookingResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// bookingOptionItem は一覧に載せる明細である。
type bookingOptionItem struct {
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	SubtotalYen int64  `json:"subtotal_yen"`
}

// bookingItem は一覧の1件である。
type bookingItem struct {
	ID       string              `json:"id"`
	LessonID string              `json:"lesson_id"`
	Start    time.Time           `json:"start"`
	End      time.Time           `json:"end"`
	Status   string              `json:"status"`
	TotalYen int64               `json:"total_yen"`
	Options  []bookingOptionItem `json:"options"`
}

// bookingListResponse は GET /members/{id}/bookings の本文である。
type bookingListResponse struct {
	Bookings []bookingItem `json:"bookings"`
}

func toListResponse(
	views []application.BookingView, loc *time.Location,
) bookingListResponse {
	items := make([]bookingItem, len(views))
	for i, v := range views {
		items[i] = bookingItem{
			ID:       v.ID,
			LessonID: v.LessonID,
			Start:    v.Start.In(loc),
			End:      v.End.In(loc),
			Status:   v.Status,
			TotalYen: v.TotalYen,
			Options:  toOptionItems(v.Options),
		}
	}
	return bookingListResponse{Bookings: items}
}

func toOptionItems(
	views []application.BookingOptionView,
) []bookingOptionItem {
	items := make([]bookingOptionItem, len(views))
	for i, v := range views {
		items[i] = bookingOptionItem{
			Name:        v.Name,
			Quantity:    v.Quantity,
			SubtotalYen: v.SubtotalYen,
		}
	}
	return items
}

// errorResponse は失敗したときの本文である。
type errorResponse struct {
	Kind    application.Kind `json:"kind"`
	Message string           `json:"message"`
}

// #region decode-json

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

// #endregion

package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/ids"
)

const (
	messageBadRequest = "リクエストの JSON が正しくありません"
	messageBadMember  = "会員 ID の形式が正しくありません"
	messageInternal   = "内部エラーが発生しました"
)

type handlers struct {
	place   placeBooking
	cancel  cancelBooking
	queries application.BookingQueries
	logger  *slog.Logger
}

func (h *handlers) postBooking(
	w http.ResponseWriter, r *http.Request,
) {
	var req placeBookingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeInvalid(w, r, messageBadRequest)
		return
	}
	out, err := h.place.Execute(r.Context(), req.input())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusCreated, bookingResponse{
		ID:     out.BookingID,
		Status: out.Status,
	})
}

func (h *handlers) postCancel(
	w http.ResponseWriter, r *http.Request,
) {
	out, err := h.cancel.Execute(r.Context(),
		application.CancelBookingInput{
			BookingID: r.PathValue("id"),
		})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, bookingResponse{
		ID:     out.BookingID,
		Status: out.Status,
	})
}

func (h *handlers) getBookings(
	w http.ResponseWriter, r *http.Request,
) {
	member, err := ids.ParseID[ids.MemberMarker](r.PathValue("id"))
	if err != nil {
		h.writeInvalid(w, r, messageBadMember)
		return
	}
	views, err := h.queries.ListByMember(r.Context(), member)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toListResponse(views))
}

func (h *handlers) writeJSON(
	w http.ResponseWriter, r *http.Request, status int, body any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.logger.WarnContext(r.Context(), "write response",
			"error", err)
	}
}

func (h *handlers) writeInvalid(
	w http.ResponseWriter, r *http.Request, message string,
) {
	h.writeJSON(w, r, http.StatusBadRequest, errorResponse{
		Kind:    application.KindInvalid,
		Message: message,
	})
}

// writeInternal は原因を含まない 500 を返す。
func (h *handlers) writeInternal(
	w http.ResponseWriter, r *http.Request,
) {
	h.writeJSON(w, r, http.StatusInternalServerError, errorResponse{
		Kind:    application.KindInternal,
		Message: messageInternal,
	})
}

// writeError は Kind をステータスに変える。5xx の本文に原因を
// 出さず、原因はログに記録する。
func (h *handlers) writeError(
	w http.ResponseWriter, r *http.Request, err error,
) {
	kind, message := application.KindInternal, messageInternal
	appErr, ok := errors.AsType[*application.ApplicationError](err)
	if ok {
		kind, message = appErr.Kind, appErr.Message
	}
	status := statusOf(kind)
	if status >= http.StatusInternalServerError {
		h.logger.Log(r.Context(), levelOf(err), "request failed",
			"error", err)
	}
	h.writeJSON(w, r, status, errorResponse{
		Kind:    kind,
		Message: message,
	})
}

func statusOf(kind application.Kind) int {
	switch kind {
	case application.KindInvalid:
		return http.StatusBadRequest
	case application.KindNotFound:
		return http.StatusNotFound
	case application.KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// levelOf は利用者の切断と期限切れの水準を下げる。
// サーバーの障害と分けて見るためである。
func levelOf(err error) slog.Level {
	if errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return slog.LevelWarn
	}
	return slog.LevelError
}

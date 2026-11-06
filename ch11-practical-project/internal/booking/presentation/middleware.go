package presentation

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
	"uuid"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/platform/logging"
)

// statusRecorder は書き込まれたステータスを覚える。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Unwrap は http.ResponseController が元の値を見つけるために使う。
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// withRequestID は request_id を採番して context と応答に入れる。
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			id := uuid.NewV7().String()
			w.Header().Set("X-Request-ID", id)
			ctx := logging.WithRequestID(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
}

// withAccessLog は1リクエストにつき1行を記録する。
func withAccessLog(
	logger *slog.Logger, next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			logger.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration", time.Since(start),
			)
		})
}

// withRecovery は panic を記録して 500 を返す。
// 本文に panic の値を出さない。
func withRecovery(h *handlers, next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w}
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				// net/http に打ち切りを伝える値はそのまま流す
				if v == http.ErrAbortHandler {
					panic(v)
				}
				h.logger.ErrorContext(r.Context(), "panic",
					"panic", v,
					"stack", string(debug.Stack()),
				)
				// 応答を書き始めていれば、ステータスは変えられない
				if rec.status == 0 {
					h.writeJSON(rec, r, statusOf(application.KindInternal),
						errorResponse{
							Kind:    application.KindInternal,
							Message: messageInternal,
						})
				}
			}()
			next.ServeHTTP(rec, r)
		})
}

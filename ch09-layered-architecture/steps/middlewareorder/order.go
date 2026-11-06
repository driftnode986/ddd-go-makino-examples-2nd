// Package middlewareorder はミドルウェアの順番で変わるログを見せる。
// 本体の middleware.go を順番だけ見比べられるように短くしたものである。
package middlewareorder

import (
	"log/slog"
	"net/http"
	"uuid"

	"example.com/trainingbooking/internal/platform/logging"
)

// WithRequestID は request_id を context に入れる。
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := logging.WithRequestID(
				r.Context(), uuid.NewV7().String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
}

// WithAccessLog は next が返った後に1行を記録する。
func WithAccessLog(
	logger *slog.Logger, next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			logger.InfoContext(r.Context(), "request")
		})
}

// WithRecovery は panic を記録して 500 を返す。
func WithRecovery(
	logger *slog.Logger, next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					logger.ErrorContext(r.Context(), "panic")
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
}

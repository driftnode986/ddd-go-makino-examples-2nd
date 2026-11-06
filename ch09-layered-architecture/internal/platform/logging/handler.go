// Package logging は slog の補助である。
package logging

import (
	"context"
	"log/slog"
)

type requestIDKey struct{}

// WithRequestID は request_id を context に入れる。
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID は context の request_id を返す。無ければ空文字列である。
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// #region context-handler

// contextHandler は context の値をログの属性に加える。
type contextHandler struct {
	slog.Handler
}

// NewContextHandler は request_id を全ログに付ける Handler を返す。
// 標準の Handler は context を読まないので、ここで読む。
func NewContextHandler(next slog.Handler) slog.Handler {
	return contextHandler{next}
}

// Handle は request_id を加えて次の Handler に渡す。
func (h contextHandler) Handle(
	ctx context.Context, r slog.Record,
) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// #endregion

// WithAttrs は context を読む動きを保ったまま属性を加える。
func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

// WithGroup は context を読む動きを保ったままグループを始める。
func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"example.com/trainingbooking/internal/platform/logging"
)

func newLogger(buf *bytes.Buffer, level slog.Level) *slog.Logger {
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(logging.NewContextHandler(inner))
}

func record(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("ログを読めない: %v: %s", err, buf)
	}
	return m
}

// 標準の JSONHandler は context を読まない。自前の Handler が付ける。
func TestLogging_AddsRequestIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, slog.LevelInfo)
	ctx := logging.WithRequestID(t.Context(), "req-1")

	logger.InfoContext(ctx, "hello", "key", "value")

	got := record(t, &buf)
	if got["request_id"] != "req-1" {
		t.Errorf("request_id = %v", got["request_id"])
	}
	if got["key"] != "value" || got["msg"] != "hello" {
		t.Errorf("record = %v", got)
	}
	if logging.RequestID(ctx) != "req-1" {
		t.Errorf("RequestID = %s", logging.RequestID(ctx))
	}
}

func TestLogging_WithoutRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, slog.LevelInfo)
	logger.InfoContext(t.Context(), "hello")
	if _, ok := record(t, &buf)["request_id"]; ok {
		t.Errorf("record = %s", buf.String())
	}
	if logging.RequestID(t.Context()) != "" {
		t.Error("RequestID が空でない")
	}
}

func TestLogging_KeepsContextHandlerThroughWith(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, slog.LevelInfo).With("app", "api")
	ctx := logging.WithRequestID(t.Context(), "req-2")
	logger.InfoContext(ctx, "hello")
	got := record(t, &buf)
	if got["app"] != "api" || got["request_id"] != "req-2" {
		t.Errorf("record = %v", got)
	}
}

// WithGroup の後の属性はグループの中に入る。Handler の約束どおりである。
func TestLogging_WithGroupKeepsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, slog.LevelInfo).WithGroup("g")
	ctx := logging.WithRequestID(t.Context(), "req-3")
	logger.InfoContext(ctx, "hello")
	group, ok := record(t, &buf)["g"].(map[string]any)
	if !ok || group["request_id"] != "req-3" {
		t.Errorf("record = %s", buf.String())
	}
}

func TestLogging_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, slog.LevelWarn)
	logger.InfoContext(t.Context(), "hidden")
	if strings.Contains(buf.String(), "hidden") {
		t.Errorf("記録された: %s", buf.String())
	}
}

package middlewareorder_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/trainingbooking/internal/platform/logging"
	"example.com/trainingbooking/steps/middlewareorder"
)

var panics = http.HandlerFunc(
	func(http.ResponseWriter, *http.Request) { panic("bug") })

// serve は1回呼び、ログの msg と request_id の有無を返す。
func serve(
	t *testing.T, wrap func(*slog.Logger, http.Handler) http.Handler,
) (int, []string) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(logging.NewContextHandler(
		slog.NewJSONHandler(&buf, nil)))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	wrap(logger, panics).ServeHTTP(rec, req)

	var lines []string
	for line := range strings.Lines(buf.String()) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		_, ok := m["request_id"]
		lines = append(lines, m["msg"].(string)+map[bool]string{
			true: " (request_id あり)", false: " (request_id なし)",
		}[ok])
	}
	return rec.Code, lines
}

// #region wrong

func TestRecoveryOutside(t *testing.T) {
	status, lines := serve(t, func(
		l *slog.Logger, h http.Handler,
	) http.Handler {
		return middlewareorder.WithRecovery(l,
			middlewareorder.WithRequestID(
				middlewareorder.WithAccessLog(l, h)))
	})
	t.Logf("Recovery が外側: %d %q", status, lines)
	if len(lines) != 1 {
		t.Fatalf("lines = %q", lines)
	}
}

// #endregion

// #region right

func TestRecoveryInside(t *testing.T) {
	status, lines := serve(t, func(
		l *slog.Logger, h http.Handler,
	) http.Handler {
		return middlewareorder.WithRequestID(
			middlewareorder.WithAccessLog(l,
				middlewareorder.WithRecovery(l, h)))
	})
	t.Logf("Recovery が内側: %d %q", status, lines)
	if len(lines) != 2 {
		t.Fatalf("lines = %q", lines)
	}
}

// #endregion

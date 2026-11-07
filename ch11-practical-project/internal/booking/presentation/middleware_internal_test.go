package presentation

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 応答を書き始めた後の panic は、接続を切って途中の本文を
// 正常な応答に見せない。
func TestRecovery_PanicAfterWriteAbortsResponse(t *testing.T) {
	var buf bytes.Buffer
	h := &handlers{logger: slog.New(slog.NewJSONHandler(&buf, nil))}
	ts := httptest.NewServer(withRecovery(h, http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"bookings":[`))
			_ = http.NewResponseController(w).Flush()
			panic("bug")
		})))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Error("本文が正常に終わった")
	}
	if !strings.Contains(buf.String(), `"msg":"panic"`) {
		t.Errorf("ログ = %s", buf.String())
	}
}

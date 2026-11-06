// Package shutdown は Close と Shutdown で処理中のリクエストがどうなるかを比べる。
package shutdown_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// result は処理中に止めたときのクライアントの結果である。
func result(
	t *testing.T, stop func(*http.Server),
) (int, error) {
	t.Helper()
	entered := make(chan struct{})
	release := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			close(entered)
			<-release
			w.WriteHeader(http.StatusCreated)
		}))
	defer ts.Close()

	type answer struct {
		status int
		err    error
	}
	got := make(chan answer, 1)
	go func() {
		resp, err := ts.Client().Get(ts.URL)
		if err != nil {
			got <- answer{0, err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		got <- answer{resp.StatusCode, nil}
	}()
	<-entered
	stop(ts.Config)
	time.Sleep(50 * time.Millisecond)
	close(release)
	a := <-got
	return a.status, a.err
}

// #region compare

func TestCloseDropsRunningRequest(t *testing.T) {
	status, err := result(t, func(s *http.Server) {
		_ = s.Close()
	})
	t.Logf("Close:    status = %d, EOF = %v",
		status, errors.Is(err, io.EOF))
	if err == nil {
		t.Fatal("応答が返った")
	}
}

func TestShutdownWaitsForRunningRequest(t *testing.T) {
	status, err := result(t, func(s *http.Server) {
		go func() { _ = s.Shutdown(context.Background()) }()
	})
	t.Logf("Shutdown: status = %d, EOF = %v",
		status, errors.Is(err, io.EOF))
	if err != nil || status != http.StatusCreated {
		t.Fatalf("status = %d, err = %v", status, err)
	}
}

// #endregion

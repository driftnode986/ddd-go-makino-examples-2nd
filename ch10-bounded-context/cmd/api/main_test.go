package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 期限までに終わらないリクエストは中断する。
func TestShutdown_CancelsRequestsAfterTimeout(t *testing.T) {
	entered := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	ts := httptest.NewServer(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			close(entered)
			select {
			case <-r.Context().Done():
				close(canceled)
			case <-release:
			}
		}))
	defer ts.Close()
	go func() {
		resp, err := ts.Client().Get(ts.URL)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-entered

	err := shutdown(ts.Config, time.Millisecond)
	if err == nil {
		t.Error("期限切れがエラーにならない")
	}
	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("リクエストが中断されない")
	}
}

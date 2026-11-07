package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 処理中のリクエストは、終わるまで待って応答を返す。
func TestShutdown_WaitsForRunningRequest(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			close(entered)
			<-release
			w.WriteHeader(http.StatusOK)
		}))
	defer ts.Close()
	got := make(chan int, 1)
	go func() {
		resp, err := ts.Client().Get(ts.URL)
		if err != nil {
			got <- 0
			return
		}
		defer func() { _ = resp.Body.Close() }()
		got <- resp.StatusCode
	}()
	<-entered

	done := make(chan error, 1)
	go func() { done <- shutdown(ts.Config, 5*time.Second) }()
	time.Sleep(50 * time.Millisecond)
	close(release)
	if err := <-done; err != nil {
		t.Errorf("shutdown: %v", err)
	}
	if status := <-got; status != http.StatusOK {
		t.Errorf("status = %d", status)
	}
}

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

// 本文の読み取りは処理の期限より先に止める。遅い本文を
// 期限切れの 500 にしないためである。
func TestNewServer_ReadsBodyBeforeRequestDeadline(t *testing.T) {
	s := newServer(nil)
	if s.ReadTimeout >= requestTimeout {
		t.Errorf("ReadTimeout = %v, requestTimeout = %v",
			s.ReadTimeout, requestTimeout)
	}
	if s.WriteTimeout <= requestTimeout {
		t.Errorf("WriteTimeout = %v, requestTimeout = %v",
			s.WriteTimeout, requestTimeout)
	}
	if requestTimeout >= shutdownTimeout {
		t.Errorf("requestTimeout = %v, shutdownTimeout = %v",
			requestTimeout, shutdownTimeout)
	}
}

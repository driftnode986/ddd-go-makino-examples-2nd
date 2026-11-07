// Package servertimeouts は http.Server のタイムアウトが止めるものを見せる。
package servertimeouts_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 本文を少しだけ送って止まる接続を作り、ハンドラの読み取りの結果を返す。
func slowBody(t *testing.T, cfg func(*http.Server)) (bool, error) {
	t.Helper()
	got := make(chan error, 1)
	ts := httptest.NewUnstartedServer(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			got <- err
		}))
	cfg(ts.Config)
	ts.Start()
	defer ts.Close()
	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_, err = fmt.Fprint(conn, "POST / HTTP/1.1\r\nHost: x\r\n"+
		"Content-Length: 100\r\n\r\nabc")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-got:
		return true, err
	case <-time.After(700 * time.Millisecond):
		return false, nil
	}
}

func TestReadHeaderTimeoutDoesNotLimitBody(t *testing.T) {
	done, _ := slowBody(t, func(s *http.Server) {
		s.ReadHeaderTimeout = 100 * time.Millisecond
	})
	t.Logf("ReadHeaderTimeout だけ: 読み取りが終わった = %v", done)
	if done {
		t.Fatal("読み取りが終わった")
	}
}

func TestReadTimeoutLimitsBody(t *testing.T) {
	done, err := slowBody(t, func(s *http.Server) {
		s.ReadHeaderTimeout = 100 * time.Millisecond
		s.ReadTimeout = 200 * time.Millisecond
	})
	t.Logf("ReadTimeout あり: 読み取りが終わった = %v", done)
	if !done || err == nil {
		t.Fatalf("done = %v, err = %v", done, err)
	}
}

func TestWriteTimeoutDoesNotCancelHandler(t *testing.T) {
	ctxErr := make(chan error, 1)
	ts := httptest.NewUnstartedServer(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			time.Sleep(300 * time.Millisecond)
			ctxErr <- r.Context().Err()
		}))
	ts.Config.WriteTimeout = 100 * time.Millisecond
	ts.Start()
	defer ts.Close()
	resp, err := ts.Client().Get(ts.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("応答が返った")
	}
	got := <-ctxErr
	t.Logf("WriteTimeout: 応答は届かない / ハンドラの ctx.Err = %v", got)
	if got != nil {
		t.Fatalf("ctx.Err = %v", got)
	}
}

// 大きいアップロードのルートは、ハンドラで期限を延ばせる。
func TestSetReadDeadlineExtendsForOneRoute(t *testing.T) {
	done, err := slowUpload(t)
	t.Logf("SetReadDeadline で延長: 読み取りのエラー = %v", err)
	if !done || err != nil {
		t.Fatalf("done = %v, err = %v", done, err)
	}
}

func slowUpload(t *testing.T) (bool, error) {
	t.Helper()
	got := make(chan error, 1)
	ts := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			err := http.NewResponseController(w).SetReadDeadline(
				time.Now().Add(2 * time.Second))
			if err != nil {
				got <- err
				return
			}
			_, err = io.ReadAll(r.Body)
			got <- err
		}))
	ts.Config.ReadTimeout = 200 * time.Millisecond
	ts.Start()
	defer ts.Close()
	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_, err = fmt.Fprint(conn, "POST / HTTP/1.1\r\nHost: x\r\n"+
		"Content-Length: 6\r\n\r\nabc")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if _, err = fmt.Fprint(conn, "def"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-got:
		return true, err
	case <-time.After(3 * time.Second):
		return false, nil
	}
}

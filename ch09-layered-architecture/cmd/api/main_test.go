package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
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

// 固定の条件は営業時間内で、起動した時刻より後でなければならない。
func TestLessonTerms_IsBookable(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	// 日本時間の深夜でも、翌日にずれない
	now := time.Date(2026, time.November, 6, 23, 30, 0, 0, jst)
	terms, err := lessonTerms(now)
	if err != nil {
		t.Fatalf("lessonTerms: %v", err)
	}
	want := time.Date(2026, time.December, 6, 10, 0, 0, 0, jst)
	if !terms.Start.Equal(want) || terms.Duration != time.Hour {
		t.Errorf("Start = %v, Duration = %v",
			terms.Start, terms.Duration)
	}
	_, err = domain.NewTimeSlot(
		terms.Start, terms.Duration, terms.Location)
	if err != nil {
		t.Errorf("NewTimeSlot: %v", err)
	}
	if terms.Fee.Amount() != 3000 || len(terms.Options) != 2 {
		t.Errorf("Fee = %v, Options = %v", terms.Fee, terms.Options)
	}
}

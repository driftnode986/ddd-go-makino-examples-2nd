package outbox_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"example.com/trainingbooking/internal/platform/outbox"
)

type event struct {
	Name string `json:"name"`
}

func TestOn_DecodesPayload(t *testing.T) {
	var gotID string
	var got event
	h := outbox.On(func(
		_ context.Context, eventID string, e event,
	) error {
		gotID, got = eventID, e
		return nil
	})
	err := h(t.Context(), outbox.Delivery{Message: outbox.Message{
		EventID: "e1",
		Payload: []byte(`{"name":"a","added_later":1}`),
	}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if gotID != "e1" || got.Name != "a" {
		t.Errorf("got = %s %+v", gotID, got)
	}
}

func TestOn_BrokenPayloadIsError(t *testing.T) {
	h := outbox.On(func(context.Context, string, event) error {
		t.Error("呼ばれた")
		return nil
	})
	err := h(t.Context(), outbox.Delivery{Message: outbox.Message{
		Payload: []byte(`{`),
	}})
	if err == nil {
		t.Fatal("err = nil")
	}
}

type emptyStore struct{ boom error }

func (s emptyStore) Deliverable(
	context.Context, int,
) ([]outbox.Delivery, error) {
	return nil, s.boom
}

func (emptyStore) MarkPublished(context.Context, int64) error {
	return nil
}

func (emptyStore) RecordFailure(
	context.Context, int64, string,
) error {
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// 読み出しの失敗で止まらず、ctx が終わるまで動く。
func TestRun_KeepsGoingUntilContextEnds(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		t.Context(), 50*time.Millisecond)
	defer cancel()
	relay := outbox.NewRelay(
		emptyStore{boom: errors.New("boom")}, discardLogger(),
		outbox.RelayOptions{Interval: time.Millisecond})
	err := relay.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunOnce_ReportsStoreFailure(t *testing.T) {
	boom := errors.New("boom")
	relay := outbox.NewRelay(emptyStore{boom: boom},
		discardLogger(), outbox.RelayOptions{})
	_, err := relay.RunOnce(t.Context())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

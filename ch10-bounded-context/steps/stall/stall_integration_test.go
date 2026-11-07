//go:build integration

// Package stall は配信の失敗が WARN から ERROR に変わり、原因が直れば
// 配信されることを確かめる。
package stall_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
	"uuid"

	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

const eventType = "booking.placed"

func TestWarnThenError(t *testing.T) {
	pool := dbtest.New(t, outbox.Migrations)
	store := outbox.NewStore(pool)
	ctx := t.Context()
	err := pgtx.NewTransactor(pool).Do(ctx,
		func(ctx context.Context) error {
			return store.Append(ctx, outbox.Message{
				EventID:       outbox.NewEventID(),
				AggregateType: "booking",
				AggregateID:   uuid.NewV7().String(),
				EventType:     eventType,
				Payload:       []byte(`{}`),
				OccurredAt:    time.Now(),
			})
		})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	cause := errors.New("connection refused")
	handler := func(context.Context, outbox.Delivery) error {
		return cause
	}
	run := func(stallAfter time.Duration) int {
		t.Helper()
		var buf bytes.Buffer
		relay := outbox.NewRelay(store,
			slog.New(slog.NewJSONHandler(&buf, nil)),
			outbox.RelayOptions{StallAfter: stallAfter})
		relay.Subscribe(eventType, "catalog", handler)
		n, err := relay.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
		logLine(t, buf.String())
		return n
	}
	run(5 * time.Minute) // 許容時間の内
	run(time.Nanosecond) // 許容時間を超えた
	cause = nil          // 原因が直る
	if n := run(5 * time.Minute); n != 1 {
		t.Fatalf("配信 = %d 件", n)
	}
	t.Log("原因が直った後: 配信 1 件, 未配信 0 件")
	head, err := store.Deliverable(ctx, 10)
	if err != nil || len(head) != 0 {
		t.Fatalf("head = %+v, err = %v", head, err)
	}
}

func logLine(t *testing.T, s string) {
	t.Helper()
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	var line struct {
		Level    string `json:"level"`
		Msg      string `json:"msg"`
		Attempts int    `json:"attempts"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal([]byte(s), &line); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	t.Logf("%s %s attempts=%d error=%q",
		line.Level, line.Msg, line.Attempts, line.Error)
}

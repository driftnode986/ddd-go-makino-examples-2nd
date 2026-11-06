//go:build integration

package postgres_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

func TestLoggingOutbox_RequiresTransaction(t *testing.T) {
	var buf bytes.Buffer
	outbox := postgres.NewLoggingOutbox(
		slog.New(slog.NewJSONHandler(&buf, nil)))
	event := domain.BookingPlaced{}

	err := outbox.Append(t.Context(), event)
	if !errors.Is(err, txn.ErrNoTransaction) {
		t.Fatalf("err = %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("記録された: %s", buf.String())
	}

	tx := pgtx.NewTransactor(newPool(t))
	err = tx.Do(t.Context(), func(ctx context.Context) error {
		return outbox.Append(ctx, event)
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if !strings.Contains(buf.String(), "booking.placed") {
		t.Errorf("記録されていない: %s", buf.String())
	}
}

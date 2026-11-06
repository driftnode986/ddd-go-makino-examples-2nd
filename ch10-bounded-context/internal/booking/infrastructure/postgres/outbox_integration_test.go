//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

func TestOutboxWriter_RequiresTransaction(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations, outbox.Migrations)
	store := outbox.NewStore(pool)
	writer := postgres.NewOutboxWriter(store)
	event := domain.BookingPlaced{
		BookingID:  ids.NewID[ids.BookingMarker](),
		MemberID:   ids.NewID[ids.MemberMarker](),
		LessonID:   ids.NewID[ids.LessonMarker](),
		OccurredAt: time.Now(),
	}

	err := writer.Append(t.Context(), event)
	if !errors.Is(err, txn.ErrNoTransaction) {
		t.Fatalf("err = %v", err)
	}

	tx := pgtx.NewTransactor(pool)
	err = tx.Do(t.Context(), func(ctx context.Context) error {
		return writer.Append(ctx, event)
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	got, err := store.Deliverable(t.Context(), 10)
	if err != nil {
		t.Fatalf("Deliverable: %v", err)
	}
	if len(got) != 1 ||
		got[0].EventType != event.EventName() ||
		got[0].AggregateID != event.AggregateID() {
		t.Fatalf("got = %+v", got)
	}
}

package outbox_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

type processedEvents interface {
	MarkProcessed(
		ctx context.Context, subscriber, eventID string,
	) (bool, error)
}

type transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// testProcessedEvents は実装に共通の動きを確かめる。
func testProcessedEvents(
	t *testing.T, p processedEvents, tx transactor,
) {
	t.Helper()
	mark := func(subscriber, id string) (first bool, err error) {
		err = tx.Do(t.Context(), func(ctx context.Context) error {
			var err error
			first, err = p.MarkProcessed(ctx, subscriber, id)
			return err
		})
		return first, err
	}
	id := uuid.New().String()
	first, err := mark("a", id)
	if !first || err != nil {
		t.Errorf("1回目: %v, %v", first, err)
	}
	first, err = mark("a", id)
	if first || err != nil {
		t.Errorf("2回目: %v, %v", first, err)
	}
	first, err = mark("b", id)
	if !first || err != nil {
		t.Errorf("別の購読者: %v, %v", first, err)
	}

	_, err = p.MarkProcessed(t.Context(), "a", uuid.New().String())
	if !errors.Is(err, txn.ErrNoTransaction) {
		t.Errorf("Tx なし: %v", err)
	}

	// Tx が戻れば記録も戻り、再配信で処理できる
	again := uuid.New().String()
	boom := errors.New("boom")
	err = tx.Do(t.Context(), func(ctx context.Context) error {
		if _, err := p.MarkProcessed(ctx, "a", again); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Do: %v", err)
	}
	first, err = mark("a", again)
	if !first || err != nil {
		t.Errorf("戻した後: %v, %v", first, err)
	}
}

func TestMemoryProcessedEvents(t *testing.T) {
	testProcessedEvents(t,
		outbox.NewMemoryProcessedEvents(), memtx.NewTransactor())
}

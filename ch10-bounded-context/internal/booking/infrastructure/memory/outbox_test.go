package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/memory"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

// A の Tx が開いているあいだに B が確定しようとしても、A の失敗で
// B のイベントは消えない。
func TestOutboxWriter_RollbackKeepsOtherCommit(t *testing.T) {
	tx := memtx.NewTransactor()
	outbox := memory.NewOutboxWriter()
	event := domain.BookingCancelled{
		BookingID: ids.NewID[ids.BookingMarker](),
	}
	appended := make(chan struct{})
	committed := make(chan error, 1)
	go func() {
		<-appended
		committed <- tx.Do(t.Context(),
			func(ctx context.Context) error {
				return outbox.Append(ctx, event)
			})
	}()

	boom := errors.New("boom")
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		if err := outbox.Append(ctx, event); err != nil {
			return err
		}
		close(appended)
		// 直列化されていなければ、ここで B が先に確定する
		select {
		case err := <-committed:
			committed <- err
		case <-time.After(50 * time.Millisecond):
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("A: %v", err)
	}
	if err := <-committed; err != nil {
		t.Fatalf("B: %v", err)
	}
	if got := outbox.Events(); len(got) != 1 {
		t.Fatalf("events = %v", got)
	}
}

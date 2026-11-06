package memtx_test

import (
	"context"
	"errors"
	"testing"

	"example.com/trainingbooking/internal/platform/txn/memtx"
)

func TestDo_RollsBackInReverseOrder(t *testing.T) {
	var got []int
	boom := errors.New("boom")
	err := memtx.NewTransactor().Do(t.Context(),
		func(ctx context.Context) error {
			cur, ok := memtx.From(ctx)
			if !ok {
				t.Fatal("ctx に Tx が無い")
			}
			cur.OnRollback(func() { got = append(got, 1) })
			cur.OnRollback(func() { got = append(got, 2) })
			return boom
		})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if len(got) != 2 || got[0] != 2 || got[1] != 1 {
		t.Fatalf("got = %v", got)
	}
}

func TestDo_NestedJoinsOuter(t *testing.T) {
	tx := memtx.NewTransactor()
	rolledBack := false
	boom := errors.New("boom")
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		err := tx.Do(ctx, func(ctx context.Context) error {
			cur, _ := memtx.From(ctx)
			cur.OnRollback(func() { rolledBack = true })
			return nil
		})
		if err != nil {
			return err
		}
		if rolledBack {
			t.Error("内側の Do で戻った")
		}
		return boom
	})
	if !errors.Is(err, boom) || !rolledBack {
		t.Fatalf("err = %v, rolledBack = %v", err, rolledBack)
	}
}

func TestDo_CommitKeepsWrites(t *testing.T) {
	rolledBack := false
	err := memtx.NewTransactor().Do(t.Context(),
		func(ctx context.Context) error {
			cur, _ := memtx.From(ctx)
			cur.OnRollback(func() { rolledBack = true })
			return nil
		})
	if err != nil || rolledBack {
		t.Fatalf("err = %v, rolledBack = %v", err, rolledBack)
	}
}

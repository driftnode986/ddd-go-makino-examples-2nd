// Package repositorytest は MemberRepository の契約テストである。
// in-memory と PostgreSQL の両方に同じテストを流す。
package repositorytest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"example.com/trainingbooking/internal/identity/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
	"github.com/google/go-cmp/cmp"
)

// Transactor は契約テストが使う Tx の境界である。
type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// Fixture は契約テストに渡す部品である。
type Fixture struct {
	Repo domain.MemberRepository
	Tx   Transactor
}

func (f Fixture) add(t *testing.T, m *domain.Member) error {
	t.Helper()
	return f.Tx.Do(t.Context(), func(ctx context.Context) error {
		return f.Repo.Add(ctx, m)
	})
}

// Run は契約テストを実行する。fixture はテストごとに空の部品を返す。
func Run(t *testing.T, fixture func(t *testing.T) Fixture) {
	t.Helper()
	cases := []struct {
		name string
		run  func(*testing.T, Fixture)
	}{
		{"往復で全項目が一致する", testRoundTrip},
		{"重複した ID は ErrDuplicate", testDuplicateID},
		{"重複した Email は ErrDuplicate", testDuplicateEmail},
		{"無い会員の FindByID は ErrNotFound", testFindMissing},
		{"Tx の外の書き込みは ErrNoTransaction", testNoTransaction},
		{"Tx の失敗で書き込みが戻る", testRollback},
		{"Tx の中で追加した値を読める", testReadInTransaction},
		{"ゼロ値の集約は保存しない", testZeroMember},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.run(t, fixture(t)) })
	}
}

func newMember(t *testing.T, address string) *domain.Member {
	t.Helper()
	email, err := domain.NewEmail(address)
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	m, err := domain.NewMember(
		ids.NewID[ids.MemberMarker](), email, "花子")
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	return m
}

func testRoundTrip(t *testing.T, f Fixture) {
	m := newMember(t, "hanako@example.com")
	if err := f.add(t, m); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := f.Repo.FindByID(t.Context(), m.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if diff := cmp.Diff(m.State(), got.State()); diff != "" {
		t.Errorf("往復の差分 (-want +got):\n%s", diff)
	}
}

func testDuplicateID(t *testing.T, f Fixture) {
	m := newMember(t, "hanako@example.com")
	if err := f.add(t, m); err != nil {
		t.Fatalf("Add: %v", err)
	}
	err := f.add(t, m)
	if !errors.Is(err, domain.ErrDuplicate) {
		t.Fatalf("err = %v", err)
	}
}

func testDuplicateEmail(t *testing.T, f Fixture) {
	err := f.add(t, newMember(t, "hanako@example.com"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	err = f.add(t, newMember(t, "hanako@EXAMPLE.com"))
	if !errors.Is(err, domain.ErrDuplicate) {
		t.Fatalf("err = %v", err)
	}
}

func testFindMissing(t *testing.T, f Fixture) {
	_, err := f.Repo.FindByID(
		t.Context(), ids.NewID[ids.MemberMarker]())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func testNoTransaction(t *testing.T, f Fixture) {
	err := f.Repo.Add(t.Context(), newMember(t, "a@example.com"))
	if !errors.Is(err, txn.ErrNoTransaction) {
		t.Fatalf("err = %v", err)
	}
}

func testRollback(t *testing.T, f Fixture) {
	m := newMember(t, "hanako@example.com")
	boom := fmt.Errorf("boom")
	err := f.Tx.Do(t.Context(), func(ctx context.Context) error {
		if err := f.Repo.Add(ctx, m); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Do: %v", err)
	}
	_, err = f.Repo.FindByID(t.Context(), m.ID())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Add が残った: %v", err)
	}
}

func testReadInTransaction(t *testing.T, f Fixture) {
	m := newMember(t, "hanako@example.com")
	err := f.Tx.Do(t.Context(), func(ctx context.Context) error {
		if err := f.Repo.Add(ctx, m); err != nil {
			return err
		}
		_, err := f.Repo.FindByID(ctx, m.ID())
		return err
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func testZeroMember(t *testing.T, f Fixture) {
	err := f.add(t, &domain.Member{})
	if !errors.Is(err, domain.ErrInvalidMember) {
		t.Fatalf("err = %v", err)
	}
}

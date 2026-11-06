package application_test

import (
	"context"
	"testing"

	"example.com/trainingbooking/internal/identity/application"
	"example.com/trainingbooking/internal/identity/domain"
	"example.com/trainingbooking/internal/identity/infrastructure/memory"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

func TestMemberReader_MemberExists(t *testing.T) {
	repo := memory.NewMemberRepository()
	email, err := domain.NewEmail("hanako@example.com")
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	m, err := domain.NewMember(
		ids.NewID[ids.MemberMarker](), email, "花子")
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	err = memtx.NewTransactor().Do(t.Context(),
		func(ctx context.Context) error {
			return repo.Add(ctx, m)
		})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	reader := application.NewMemberReader(repo)
	got, err := reader.MemberExists(t.Context(), m.ID())
	if err != nil || !got {
		t.Errorf("登録済み: %v, %v", got, err)
	}
	got, err = reader.MemberExists(
		t.Context(), ids.NewID[ids.MemberMarker]())
	if err != nil || got {
		t.Errorf("未登録: %v, %v", got, err)
	}
}

func TestMemberReader_FailureIsError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reader := application.NewMemberReader(
		memory.NewMemberRepository())
	_, err := reader.MemberExists(
		ctx, ids.NewID[ids.MemberMarker]())
	if err == nil {
		t.Fatal("err = nil")
	}
}

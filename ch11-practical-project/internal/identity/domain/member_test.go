package domain_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/internal/identity/domain"
	"example.com/trainingbooking/internal/ids"
)

func newEmail(t *testing.T) domain.Email {
	t.Helper()
	e, err := domain.NewEmail("hanako@example.com")
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	return e
}

func TestNewMember_KeepsValues(t *testing.T) {
	id := ids.NewID[ids.MemberMarker]()
	m, err := domain.NewMember(id, newEmail(t), "花子")
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	s := m.State()
	if !m.ID().Equal(id) || s.Email != "hanako@example.com" ||
		s.Name != "花子" {
		t.Errorf("state = %+v", s)
	}
}

func TestNewMember_RejectsZeroValues(t *testing.T) {
	id := ids.NewID[ids.MemberMarker]()
	email := newEmail(t)
	var zero domain.Email
	cases := []struct {
		name  string
		id    ids.MemberID
		email domain.Email
		nick  string
	}{
		{"ゼロの ID", ids.MemberID{}, email, "花子"},
		{"ゼロの Email", id, zero, "花子"},
		{"空の名前", id, email, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := domain.NewMember(c.id, c.email, c.nick)
			if !errors.Is(err, domain.ErrInvalidMember) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestReconstruct_RejectsBrokenState(t *testing.T) {
	valid := domain.MemberState{
		ID:    ids.NewID[ids.MemberMarker](),
		Email: "hanako@example.com",
		Name:  "花子",
	}
	if _, err := domain.ReconstructMember(valid); err != nil {
		t.Fatalf("正しい中身: %v", err)
	}
	broken := valid
	broken.Email = "not an address"
	_, err := domain.ReconstructMember(broken)
	if !errors.Is(err, domain.ErrInvalidMember) {
		t.Errorf("不正な Email: %v", err)
	}
	broken = valid
	broken.ID = ids.MemberID{}
	_, err = domain.ReconstructMember(broken)
	if !errors.Is(err, domain.ErrInvalidMember) {
		t.Errorf("ゼロの ID: %v", err)
	}
}

package domain_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/internal/identity/domain"
)

func TestEmail_Valid(t *testing.T) {
	e, err := domain.NewEmail("taro@example.com")
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	if e.String() != "taro@example.com" {
		t.Errorf("String = %q", e.String())
	}
}

func TestEmail_RejectsDisplayName(t *testing.T) {
	inputs := []string{
		"Taro <taro@example.com>",
		"<taro@example.com>",
		"taro@example.com (Taro)",
	}
	for _, in := range inputs {
		_, err := domain.NewEmail(in)
		if !errors.Is(err, domain.ErrInvalidEmail) {
			t.Errorf("NewEmail(%q) = %v", in, err)
		}
	}
}

func TestEmail_RejectsMalformed(t *testing.T) {
	inputs := []string{
		"",
		"taro",
		"@example.com",
		"taro@",
		" taro@example.com ",
	}
	for _, in := range inputs {
		_, err := domain.NewEmail(in)
		if !errors.Is(err, domain.ErrInvalidEmail) {
			t.Errorf("NewEmail(%q) = %v", in, err)
		}
	}
}

func TestEmail_NormalizesDomain(t *testing.T) {
	a, err := domain.NewEmail("Taro@EXAMPLE.com")
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	// ローカル部の大文字と小文字は区別する（RFC 5321）
	if a.String() != "Taro@example.com" {
		t.Errorf("String = %q", a.String())
	}
	b, err := domain.NewEmail("Taro@example.com")
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	if !a.Equal(b) {
		t.Error("ドメイン部の大文字と小文字で等価性が変わった")
	}
	c, err := domain.NewEmail("taro@example.com")
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	if a.Equal(c) {
		t.Error("ローカル部の違いは別のアドレスである")
	}
}

func TestZeroValues_AreDetected(t *testing.T) {
	var e domain.Email
	if !e.IsZero() {
		t.Error("var で宣言した Email は IsZero が true")
	}
	valid, err := domain.NewEmail("taro@example.com")
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	if valid.IsZero() {
		t.Error("NewEmail の結果は IsZero が false")
	}
}

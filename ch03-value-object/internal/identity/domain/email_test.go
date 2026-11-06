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
		`"taro"@example.com`,
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

func mustEmail(t *testing.T, s string) domain.Email {
	t.Helper()
	e, err := domain.NewEmail(s)
	if err != nil {
		t.Fatalf("NewEmail(%q): %v", s, err)
	}
	return e
}

// #region normalizes-domain

func TestEmail_NormalizesDomain(t *testing.T) {
	a := mustEmail(t, "Taro@EXAMPLE.com")
	if a.String() != "Taro@example.com" {
		t.Errorf("String = %q", a.String())
	}
	// ドメイン部の大文字と小文字は同じものとして扱う
	if !a.Equal(mustEmail(t, "Taro@example.com")) {
		t.Error("ドメイン部の違いで等価性が変わった")
	}
	// ローカル部の大文字と小文字は別のものとして扱う
	if a.Equal(mustEmail(t, "taro@example.com")) {
		t.Error("ローカル部の違いは別のアドレスである")
	}
}

// #endregion

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

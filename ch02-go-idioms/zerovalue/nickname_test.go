package zerovalue_test

import (
	"bytes"
	"errors"
	"testing"

	"example.com/trainingbooking/zerovalue"
)

// 公開フィールドは型の外から不正な値を入れられる。
func TestPublicField_AcceptsNegative(t *testing.T) {
	p := zerovalue.PublicPoints{Balance: -100}
	if p.Balance >= 0 {
		t.Fatalf("Balance = %d", p.Balance)
	}
}

// 非公開フィールドでも、var でゼロ値は作れる。
func TestNickname_ZeroValueCanBeDeclared(t *testing.T) {
	var n zerovalue.Nickname
	if !n.IsZero() {
		t.Fatal("var で宣言した Nickname は IsZero が true")
	}
}

// ゼロ値を止めるのは、受け取る側のコンストラクタである。
func TestNewProfile_RejectsZeroValue(t *testing.T) {
	_, err := zerovalue.NewProfile(zerovalue.Nickname{})
	if !errors.Is(err, zerovalue.ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}

// Effective Go が勧める形はゼロ値のまま使える型である。
// bytes.Buffer はその例で、生成時の不変条件を持たない。
func TestBuffer_ZeroValueIsUsable(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("ok")
	if b.String() != "ok" {
		t.Fatalf("String = %q", b.String())
	}
}

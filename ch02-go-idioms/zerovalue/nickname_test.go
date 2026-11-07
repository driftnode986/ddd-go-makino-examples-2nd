package zerovalue_test

import (
	"bytes"
	"errors"
	"testing"

	"example.com/trainingbooking/zerovalue"
)

// #region public

func TestPublicField_AcceptsEmpty(t *testing.T) {
	n := zerovalue.PublicNickname{Value: ""}
	t.Logf("Value = %q", n.Value)
}

// #endregion

// #region zero

// どちらも var で宣言できる。違うのは、そのまま使えるかである。
func TestZeroValue_BufferAndNickname(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("ok")
	t.Logf("Buffer: %q", b.String())

	var n zerovalue.Nickname
	t.Logf("Nickname: IsZero = %v", n.IsZero())
}

// #endregion

// #region reject

func TestNewProfile_RejectsZeroValue(t *testing.T) {
	var n zerovalue.Nickname

	_, err := zerovalue.NewProfile(n)

	if !errors.Is(err, zerovalue.ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}

// #endregion

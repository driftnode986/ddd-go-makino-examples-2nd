package stringemail_test

import (
	"testing"

	"example.com/trainingbooking/steps/stringemail"
)

// #region bypass
func TestConstructorIsBypassed(t *testing.T) {
	a := stringemail.Email("not-an-email") // 型変換
	var b stringemail.Email = "山田太郎"       // 型なし定数の代入
	t.Logf("a=%q b=%q", a, b)

	if _, err := stringemail.NewEmail(string(a)); err == nil {
		t.Fatal("NewEmail を通せば拒否される値のはず")
	}
}

// #endregion

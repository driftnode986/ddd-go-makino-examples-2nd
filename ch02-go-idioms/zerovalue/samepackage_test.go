package zerovalue

import "testing"

// #region same-package

// 同じパッケージのコードは、別の型の非公開フィールドにも書ける。
func TestSamePackage_CanWriteUnexported(t *testing.T) {
	p, err := NewProfile(Nickname{value: "taro"})
	if err != nil {
		t.Fatalf("NewProfile: %v", err)
	}

	p.nickname = Nickname{}

	t.Logf("IsZero = %v", p.nickname.IsZero())
}

// #endregion

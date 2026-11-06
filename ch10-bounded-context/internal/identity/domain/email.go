// Package domain は Identity コンテキストのドメインモデルである。
package domain

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

// #region email-type

// ErrInvalidEmail はメールアドレスの形式が不正なことを表す。
var ErrInvalidEmail = errors.New("invalid email")

// Email は会員のメールアドレスを表す値オブジェクトである。
type Email struct {
	value string
}

// #endregion

// #region new-email

// NewEmail はアドレスだけの文字列から Email を作る。
// 表示名や前後の空白を含む文字列は拒否し、
// ドメイン部を小文字にそろえる。
func NewEmail(s string) (Email, error) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return Email{}, fmt.Errorf("%w: %v", ErrInvalidEmail, err)
	}
	// ParseAddress は表示名や山括弧を取り除いて返す。
	// 入力と一致しなければ、アドレス以外を含んでいる。
	if addr.Name != "" || addr.Address != s {
		return Email{}, fmt.Errorf(
			"%w: address only: %q", ErrInvalidEmail, s)
	}
	at := strings.LastIndex(s, "@")
	return Email{value: s[:at] + strings.ToLower(s[at:])}, nil
}

// #endregion

// #region email-methods

// String はアドレスの文字列を返す。
func (e Email) String() string { return e.value }

// Equal は同じアドレスのとき true を返す。
func (e Email) Equal(other Email) bool {
	return e.value == other.value
}

// IsZero は NewEmail を通していないとき true を返す。
func (e Email) IsZero() bool { return e.value == "" }

// #endregion

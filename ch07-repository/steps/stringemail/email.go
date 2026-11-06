// Package stringemail は、string を元にした型で Email を作る例である。
// 本書の Email と比べるために置く。
package stringemail

import (
	"errors"
	"net/mail"
)

// ErrInvalidEmail はメールアドレスの形式が不正なことを表す。
var ErrInvalidEmail = errors.New("invalid email")

// #region type
type Email string

func NewEmail(s string) (Email, error) {
	if _, err := mail.ParseAddress(s); err != nil {
		return "", ErrInvalidEmail
	}
	return Email(s), nil
}

// #endregion

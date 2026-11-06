// Package registermember は、ゼロ値の Email を受け取る側で止める例である。
// 実際の会員は第11章の internal/identity/domain で作る。
package registermember

import (
	"errors"

	"example.com/trainingbooking/internal/identity/domain"
)

// Member は会員である。
type Member struct {
	name  string
	email domain.Email
}

// #region register
var ErrMissingEmail = errors.New("email is required")

func RegisterMember(
	name string, email domain.Email,
) (Member, error) {
	if email.IsZero() {
		return Member{}, ErrMissingEmail
	}
	return Member{name: name, email: email}, nil
}

// #endregion

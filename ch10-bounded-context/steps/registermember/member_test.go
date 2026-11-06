package registermember_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/internal/identity/domain"
	"example.com/trainingbooking/steps/registermember"
)

// #region stop
func TestZeroEmailIsRejected(t *testing.T) {
	var e domain.Email
	_, err := registermember.RegisterMember("山田太郎", e)
	t.Logf("err=%v", err)
	if !errors.Is(err, registermember.ErrMissingEmail) {
		t.Fatalf("got %v", err)
	}
}

// #endregion

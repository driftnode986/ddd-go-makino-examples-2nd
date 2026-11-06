package rawerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/steps/rawerror"
)

func TestRawErrorsReachTheClient(t *testing.T) {
	const internal = http.StatusInternalServerError
	db := errors.New("connection refused: secret-host:5432")
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"DB の障害", fmt.Errorf("add: %w", db), internal},
		{"期限切れ", domain.ErrCancelTooLate, internal},
		{"明細の上限", domain.ErrOptionLimit, internal},
	}
	for _, c := range cases {
		status, body := rawerror.Respond(c.err)
		t.Logf("%s: %d %s", c.name, status, body)
		if status != c.want {
			t.Fatalf("status = %d", status)
		}
	}
}

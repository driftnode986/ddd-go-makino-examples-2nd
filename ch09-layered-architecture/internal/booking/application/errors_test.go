package application_test

import (
	"errors"
	"strings"
	"testing"

	"example.com/trainingbooking/internal/booking/application"
)

func TestApplicationError_ErrorKeepsCauseForLogs(t *testing.T) {
	cause := errors.New("boom")
	err := &application.ApplicationError{
		Kind:    application.KindInternal,
		Message: "msg",
		Cause:   cause,
	}
	if !errors.Is(err, cause) {
		t.Error("Unwrap で原因をたどれない")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("Error = %s", err.Error())
	}
}

func TestApplicationError_WithoutCause(t *testing.T) {
	err := &application.ApplicationError{
		Kind:    application.KindInvalid,
		Message: "msg",
	}
	if err.Unwrap() != nil {
		t.Error("Unwrap が nil でない")
	}
	if !strings.Contains(err.Error(), "msg") {
		t.Errorf("Error = %s", err.Error())
	}
}

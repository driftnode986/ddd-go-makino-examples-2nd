package reminder_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/smallinterface/mailer"
	"example.com/trainingbooking/smallinterface/reminder"
)

// #region fake

// failing は Send だけを持つテスト用の型である。
type failing struct{ err error }

func (f failing) Send(string, string) error { return f.err }

// #endregion

func TestReminder_WrapsSendError(t *testing.T) {
	boom := errors.New("boom")
	r := reminder.NewReminder(failing{boom})

	err := r.Remind("a@example.com")

	t.Logf("err = %v", err)
	if !errors.Is(err, boom) {
		t.Fatalf("errors.Is(err, boom) = false")
	}
}

// NewMailer が具体型を返すので、Sent も読める。
func TestReminder_WithMailer(t *testing.T) {
	m := mailer.NewMailer()
	r := reminder.NewReminder(m)

	if err := r.Remind("a@example.com"); err != nil {
		t.Fatalf("Remind: %v", err)
	}

	t.Logf("Sent = %q", m.Sent())
	if len(m.Sent()) != 1 {
		t.Fatalf("len(Sent) = %d", len(m.Sent()))
	}
}

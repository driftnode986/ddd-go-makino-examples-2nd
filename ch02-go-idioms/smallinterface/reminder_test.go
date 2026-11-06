package smallinterface_test

import (
	"errors"
	"testing"

	"example.com/trainingbooking/smallinterface"
)

// 実装のメソッドが多くても、使う側は1メソッドだけを見る。
func TestReminder_UsesOnlySend(t *testing.T) {
	m := &smallinterface.Mailer{}
	r := smallinterface.NewReminder(m)
	if err := r.Remind("a@example.com"); err != nil {
		t.Fatalf("Remind: %v", err)
	}
	if len(m.Sent()) != 1 {
		t.Fatalf("Sent = %v", m.Sent())
	}
}

type failing struct{ err error }

func (f failing) Send(string, string) error { return f.err }

// テストでは1メソッドの型を書くだけで差し替えられる。
func TestReminder_WrapsSendError(t *testing.T) {
	boom := errors.New("boom")
	r := smallinterface.NewReminder(failing{boom})
	if err := r.Remind("a@example.com"); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

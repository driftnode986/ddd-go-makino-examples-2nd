// Package smallinterface は、小さい interface が interface の
// 大きさの話で、struct のメソッド数の話ではないことを示す。
package smallinterface

import "fmt"

// Mailer はメールを送る実装である。メソッドは多くてもよい。
type Mailer struct {
	sent []string
}

// Send はメールを送る。ここでは本文を記録するだけである。
func (m *Mailer) Send(to, body string) error {
	m.sent = append(m.sent, to+": "+body)
	return nil
}

// Sent は送った内容を返す。
func (m *Mailer) Sent() []string { return m.sent }

// Reset は記録を消す。
func (m *Mailer) Reset() { m.sent = nil }

// Notifier は使う側が必要なメソッドだけを宣言する。
// Mailer は宣言なしでこの interface を満たす。
type Notifier interface {
	Send(to, body string) error
}

// Reminder は Notifier だけに依存する。
type Reminder struct {
	notifier Notifier
}

// NewReminder は依存をコンストラクタで受け取る。
func NewReminder(n Notifier) *Reminder {
	return &Reminder{notifier: n}
}

// Remind はお知らせを送る。
func (r *Reminder) Remind(to string) error {
	if err := r.notifier.Send(to, "明日の10時です"); err != nil {
		return fmt.Errorf("remind %s: %w", to, err)
	}
	return nil
}

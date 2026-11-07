// Package reminder はお知らせを送る。mailer を import しない。
package reminder

import "fmt"

// #region notifier

// Notifier は、Reminder が使うメソッドだけを宣言する。
type Notifier interface {
	Send(to, body string) error
}

// #endregion

// #region reminder

// Reminder は Notifier を通してお知らせを送る。
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

// #endregion

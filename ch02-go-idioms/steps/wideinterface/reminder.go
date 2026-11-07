// Package wideinterface は、実装のメソッドをすべて写した
// インターフェースを使う側に要求する形を示す。
package wideinterface

// #region wide

// Notifier は Mailer のメソッドをすべて写している。
type Notifier interface {
	Send(to, body string) error
	Sent() []string
	Reset()
}

// Reminder が呼ぶのは Send だけである。
type Reminder struct {
	notifier Notifier
}

// #endregion

// NewReminder は依存をコンストラクタで受け取る。
func NewReminder(n Notifier) *Reminder {
	return &Reminder{notifier: n}
}

// Remind はお知らせを送る。
func (r *Reminder) Remind(to string) error {
	return r.notifier.Send(to, "明日の10時です")
}

// Package mailer はメールを送る実装である。Notifier を知らない。
package mailer

// #region mailer

// Mailer はメールを送る。メソッドの数に上限はない。
type Mailer struct {
	sent []string
}

// NewMailer は具体型の *Mailer を返す。
func NewMailer() *Mailer { return &Mailer{} }

// Send はメールを送る。ここでは本文を記録するだけである。
func (m *Mailer) Send(to, body string) error {
	m.sent = append(m.sent, to+": "+body)
	return nil
}

// Sent は送った内容を返す。
func (m *Mailer) Sent() []string { return m.sent }

// Reset は記録を消す。
func (m *Mailer) Reset() { m.sent = nil }

// #endregion

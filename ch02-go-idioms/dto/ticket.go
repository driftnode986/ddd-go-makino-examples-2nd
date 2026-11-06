// Package dto は、同じ struct で境界を越えない理由を示す。
package dto

// Ticket はドメインの型である。JSON タグを持たない。
type Ticket struct {
	code string
	used bool
}

// NewTicket は Ticket を作る。
func NewTicket(code string) *Ticket { return &Ticket{code: code} }

// Code はコードを返す。
func (t *Ticket) Code() string { return t.code }

// Used は使用済みのとき true を返す。
func (t *Ticket) Used() bool { return t.used }

// TicketResponse は HTTP の DTO である。JSON の形はここで決める。
type TicketResponse struct {
	Code string `json:"code"`
	Used bool   `json:"used"`
}

// ToResponse はドメインの型を DTO に変える。
func ToResponse(t *Ticket) TicketResponse {
	return TicketResponse{Code: t.Code(), Used: t.Used()}
}

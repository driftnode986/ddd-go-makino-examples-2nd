// Package leakyoptions は、内部のスライスをそのまま返す例である。
package leakyoptions

// Option は明細である。
type Option struct {
	Name     string
	PriceYen int64
}

// Booking は明細と合計を持つ予約である。
type Booking struct {
	options  []Option
	totalYen int64
}

// New は料金を受け取って予約を作る。
func New(feeYen int64) *Booking {
	return &Booking{totalYen: feeYen}
}

// AddOption は明細を足して合計を更新する。
func (b *Booking) AddOption(o Option) {
	b.options = append(b.options, o)
	b.totalYen += o.PriceYen
}

// Total は合計を返す。
func (b *Booking) Total() int64 { return b.totalYen }

// #region leak

// Options は内部のスライスをそのまま返す。
func (b *Booking) Options() []Option { return b.options }

// #endregion

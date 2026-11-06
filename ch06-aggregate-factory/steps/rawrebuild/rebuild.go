// Package rawrebuild は、再構築で検証をすべて省く例である。
// 別のパッケージから Booking の非公開フィールドに代入できないので、
// 同じ形の小さな型で試す。
package rawrebuild

// #region raw

// State は保存された中身である。
type State struct {
	FeeYen      int64
	DiscountYen int64
	Status      string
}

// Booking は検証せずに組み立てた予約である。
type Booking struct{ state State }

// Rebuild は読んだ値をそのまま詰める。
func Rebuild(s State) *Booking { return &Booking{state: s} }

// Total は料金から割引を引く。
func (b *Booking) Total() int64 {
	return b.state.FeeYen - b.state.DiscountYen
}

// Status は状態を返す。
func (b *Booking) Status() string { return b.state.Status }

// #endregion

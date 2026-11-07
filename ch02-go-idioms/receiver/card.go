// Package receiver はレシーバの選び方を示す。値を変えない型は
// 値レシーバ、状態を持つ型はポインタレシーバにそろえる。
package receiver

// #region points

// Points は作った後に変わらない値である。計算は新しい値を返す。
type Points struct {
	n int
}

// NewPoints は Points を作る。
func NewPoints(n int) Points { return Points{n: n} }

// Add は合計を新しい値で返す。元の値は変わらない。
func (p Points) Add(other Points) Points {
	return Points{n: p.n + other.n}
}

// Count は値を返す。getter の名前に Get を付けない。
func (p Points) Count() int { return p.n }

// #endregion

// #region card

// Card は状態が変わっていく型である。
type Card struct {
	balance Points
}

// Charge は残高を増やす。呼び出し元の Card が変わる。
func (c *Card) Charge(p Points) { c.balance = c.balance.Add(p) }

// Balance は残高を返す。読むだけでもレシーバをそろえる。
func (c *Card) Balance() Points { return c.balance }

// #endregion

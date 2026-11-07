// Package setterstock は、ルールを型の外に置くと setter が要り、
// setter を呼ぶ処理にはルールが効かないことを示す。
package setterstock

// #region stock

// Stock はデータだけを持つ。
type Stock struct {
	count int
}

// NewStock は在庫を作る。
func NewStock(count int) *Stock { return &Stock{count: count} }

// Count は在庫の数を返す。
func (s *Stock) Count() int { return s.count }

// SetCount は在庫の数を書き換える。外のパッケージのために要る。
func (s *Stock) SetCount(n int) { s.count = n }

// #endregion

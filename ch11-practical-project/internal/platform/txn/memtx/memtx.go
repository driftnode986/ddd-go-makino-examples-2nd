// Package memtx は in-memory のテストダブルが使う Tx である。
package memtx

import "context"

type key struct{}

// Tx は Do のあいだの書き込みの取り消し方を持つ。
type Tx struct {
	undo []func()
}

// OnRollback は Tx が戻るときに実行する関数を登録する。
// 登録の逆順に実行する。
func (t *Tx) OnRollback(fn func()) {
	t.undo = append(t.undo, fn)
}

func (t *Tx) rollback() {
	for i := len(t.undo) - 1; i >= 0; i-- {
		t.undo[i]()
	}
}

// From は ctx の Tx を返す。
func From(ctx context.Context) (*Tx, bool) {
	cur, ok := ctx.Value(key{}).(*Tx)
	return cur, ok
}

// Transactor は PostgreSQL の Transactor のテストダブルである。
// 失敗したら書き込みを戻す。テストダブルなので隔離はしない。
// コミット前の値は他の Do と Tx の外からも見える。
type Transactor struct{}

// NewTransactor は Transactor を返す。
func NewTransactor() *Transactor { return &Transactor{} }

// Do は fn を1つの Tx として実行する。fn がエラーを返すか panic
// すれば戻す。fn は渡された ctx を使う。
// 入れ子の Do は外側の Tx に参加する。
func (t *Transactor) Do(
	ctx context.Context, fn func(context.Context) error,
) error {
	if _, ok := From(ctx); ok {
		return fn(ctx)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cur := &Tx{}
	committed := false
	defer func() {
		if !committed {
			cur.rollback()
		}
	}()
	err := fn(context.WithValue(ctx, key{}, cur))
	if err != nil {
		return err
	}
	// キャンセルされた Tx はコミットしない。pgx と同じ扱いにする
	if err := ctx.Err(); err != nil {
		return err
	}
	committed = true
	return nil
}

package memory

import "context"

type txKey struct{}

// tx は Do のあいだの書き込みと、その取り消し方を持つ。
type tx struct {
	undo []func()
}

func txFrom(ctx context.Context) (*tx, bool) {
	cur, ok := ctx.Value(txKey{}).(*tx)
	return cur, ok
}

func (t *tx) rollback() {
	for i := len(t.undo) - 1; i >= 0; i-- {
		t.undo[i]()
	}
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
	if _, ok := txFrom(ctx); ok {
		return fn(ctx)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cur := &tx{}
	committed := false
	defer func() {
		if !committed {
			cur.rollback()
		}
	}()
	err := fn(context.WithValue(ctx, txKey{}, cur))
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

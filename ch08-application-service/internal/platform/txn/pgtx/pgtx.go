// Package pgtx は pgx の Tx を context に乗せる。
package pgtx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type key struct{}

// Transactor は pgx の Tx を context に乗せて関数を実行する。
type Transactor struct {
	pool *pgxpool.Pool
}

// NewTransactor は Transactor を返す。
func NewTransactor(pool *pgxpool.Pool) *Transactor {
	return &Transactor{pool: pool}
}

// #region do

// Do は fn を1つの Tx で実行する。fn がエラーを返せば戻す。
// fn は渡された ctx を使う。入れ子の Do は外側の Tx に参加する。
func (t *Transactor) Do(
	ctx context.Context, fn func(context.Context) error,
) error {
	if _, ok := From(ctx); ok {
		return fn(ctx)
	}
	return pgx.BeginTxFunc(ctx, t.pool, pgx.TxOptions{},
		func(tx pgx.Tx) error {
			return fn(context.WithValue(ctx, key{}, tx))
		})
}

// #endregion

// #region from

// From は ctx の Tx を返す。
func From(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(key{}).(pgx.Tx)
	return tx, ok
}

// #endregion

// Package postgres は Booking の PostgreSQL 実装である。
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

// Transactor は pgx の Tx を context に乗せて関数を実行する。
type Transactor struct {
	pool *pgxpool.Pool
}

// NewTransactor は Transactor を返す。
func NewTransactor(pool *pgxpool.Pool) *Transactor {
	return &Transactor{pool: pool}
}

// Do は fn を1つの Tx で実行する。fn がエラーを返せば戻す。
// fn は渡された ctx を使う。入れ子の Do は外側の Tx に参加する。
func (t *Transactor) Do(
	ctx context.Context, fn func(context.Context) error,
) error {
	if _, ok := txFrom(ctx); ok {
		return fn(ctx)
	}
	return pgx.BeginTxFunc(ctx, t.pool, pgx.TxOptions{},
		func(tx pgx.Tx) error {
			return fn(context.WithValue(ctx, txKey{}, tx))
		})
}

func txFrom(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// Package txn はコンテキストをまたいで共有する Tx の取り決めである。
// 実装は pgtx（PostgreSQL）と memtx（テストダブル）にある。
// 同じ Transactor の Tx に複数のコンテキストの書き込みが参加できる。
// domain からは import しない。契約テストと infrastructure が使う。
package txn

import "errors"

// ErrNoTransaction は Tx の無い ctx で書き込もうとしたことを表す。
var ErrNoTransaction = errors.New("no transaction in context")

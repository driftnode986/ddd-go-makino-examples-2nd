package application

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
)

// #region transactor

// Transactor は Tx の境界である。fn は渡された ctx を使う。
// fn がエラーを返せば戻す。入れ子の Do は外側の Tx に参加するので、
// 内側のエラーは一番外側まで返さないと戻らない。
type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// #endregion

// #region outbox-writer

// OutboxWriter はイベントを集約と同じ Tx で書く。
// ctx に Tx が無ければ実装はエラー（txn.ErrNoTransaction）を返す。
type OutboxWriter interface {
	Append(ctx context.Context, events ...domain.DomainEvent) error
}

// #endregion

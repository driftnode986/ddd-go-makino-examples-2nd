package application

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
)

// Transactor は Tx の境界である。fn は渡された ctx を使う。
// fn がエラーを返せば戻す。入れ子の Do は外側の Tx に参加する。
type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// OutboxWriter はイベントを集約と同じ Tx で書く。
// ctx に Tx が無ければ domain.ErrNoTransaction を返す。
type OutboxWriter interface {
	Append(ctx context.Context, events ...domain.DomainEvent) error
}

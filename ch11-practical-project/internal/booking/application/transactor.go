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
// ctx に Tx が無ければ実装はエラー（txn.ErrNoTransaction）を返す。
type OutboxWriter interface {
	Append(ctx context.Context, events ...domain.DomainEvent) error
}

// ProcessedEvents は処理したイベントの記録である。集約の保存と
// 同じ Tx で書く。初めてのイベントなら true を返す。
type ProcessedEvents interface {
	MarkProcessed(
		ctx context.Context, subscriber, eventID string,
	) (bool, error)
}

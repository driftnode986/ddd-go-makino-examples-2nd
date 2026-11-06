package booking

import (
	"context"
	"fmt"
)

// Transactor は Tx の境界である。@Transactional の代わりに、
// 範囲を関数で明示する。fn は渡された ctx を使う。
type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// EventWriter は @TransactionalEventListener の代わりに、
// イベントを集約と同じ Tx で書く。
type EventWriter interface {
	Append(ctx context.Context, events ...Event) error
}

// CancelBooking はユースケースである。
//
//	@Service class CancelBooking(
//	    private val repo: BookingRepository,
//	) {
//	    @Transactional fun execute(id: UUID) { ... }
//	}
//
// DI コンテナは使わず、main で組み立てて渡す。
type CancelBooking struct {
	repo   Repository
	writer EventWriter
	tx     Transactor
	clock  Clock
}

// NewCancelBooking は依存をコンストラクタで受け取る。
func NewCancelBooking(
	repo Repository, writer EventWriter, tx Transactor, clock Clock,
) *CancelBooking {
	return &CancelBooking{repo, writer, tx, clock}
}

// Execute は1つの Tx で読み、変更し、保存する。
func (u *CancelBooking) Execute(
	ctx context.Context, id BookingID,
) error {
	return u.tx.Do(ctx, func(ctx context.Context) error {
		b, err := u.repo.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("find: %w", err)
		}
		if err := b.Cancel(u.clock); err != nil {
			return err
		}
		if err := u.repo.Save(ctx, b); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		return u.writer.Append(ctx, b.PullEvents()...)
	})
}

package outbox

import (
	"context"
	"sync"

	"example.com/trainingbooking/internal/platform/outbox/sqlcgen"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/memtx"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProcessedEvents は受け手が処理したイベントの記録である。
// 集約の保存と同じ Tx で書く。確かめてから処理して記録する順に
// 分けると、途中で止まったときの再配信で二重に処理する。
type ProcessedEvents struct {
	pool *pgxpool.Pool
}

// NewProcessedEvents は ProcessedEvents を返す。
func NewProcessedEvents(pool *pgxpool.Pool) *ProcessedEvents {
	return &ProcessedEvents{pool: pool}
}

// MarkProcessed は処理済みとして記録する。初めてのイベントなら
// true、処理済みなら false を返す。一意キーは購読者とイベントの ID
// である。ctx に Tx が無ければ txn.ErrNoTransaction を返す。
func (p *ProcessedEvents) MarkProcessed(
	ctx context.Context, subscriber, eventID string,
) (bool, error) {
	tx, ok := pgtx.From(ctx)
	if !ok {
		return false, txn.ErrNoTransaction
	}
	rows, err := sqlcgen.New(tx).MarkProcessed(ctx,
		sqlcgen.MarkProcessedParams{
			Subscriber: subscriber,
			EventID:    eventID,
		})
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

// MemoryProcessedEvents は ProcessedEvents のテストダブルである。
type MemoryProcessedEvents struct {
	mu   sync.Mutex
	seen map[string]bool
}

// NewMemoryProcessedEvents は空の記録を返す。
func NewMemoryProcessedEvents() *MemoryProcessedEvents {
	return &MemoryProcessedEvents{seen: make(map[string]bool)}
}

// MarkProcessed は ProcessedEvents と同じ判定を行う。
func (p *MemoryProcessedEvents) MarkProcessed(
	ctx context.Context, subscriber, eventID string,
) (bool, error) {
	cur, ok := memtx.From(ctx)
	if !ok {
		return false, txn.ErrNoTransaction
	}
	key := subscriber + ":" + eventID
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.seen[key] {
		return false, nil
	}
	p.seen[key] = true
	cur.OnRollback(func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		delete(p.seen, key)
	})
	return true, nil
}

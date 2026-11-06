//go:build integration

package application_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"example.com/trainingbooking/internal/catalog/application"
	"example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/catalog/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

// racingRepo は2本の処理が同じ版を読むまで読み込みを止める。
// 残席1の競合を偶然に頼らず必ず起こすためである。
type racingRepo struct {
	domain.LessonRepository
	arrived   atomic.Int32
	bothRead  chan struct{}
	conflicts atomic.Int32
}

func newRacingRepo(inner domain.LessonRepository) *racingRepo {
	return &racingRepo{
		LessonRepository: inner,
		bothRead:         make(chan struct{}),
	}
}

func (r *racingRepo) FindByID(
	ctx context.Context, id ids.LessonID,
) (*domain.Lesson, error) {
	l, err := r.LessonRepository.FindByID(ctx, id)
	if r.arrived.Add(1) == 2 {
		close(r.bothRead)
	}
	select {
	case <-r.bothRead:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return l, err
}

func (r *racingRepo) Save(
	ctx context.Context, l *domain.Lesson,
) error {
	err := r.LessonRepository.Save(ctx, l)
	if errors.Is(err, domain.ErrVersionConflict) {
		r.conflicts.Add(1)
	}
	return err
}

func TestReserveSeat_ConcurrentLastSeat(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations, outbox.Migrations)
	repo := newRacingRepo(postgres.NewLessonRepository(pool))
	tx := pgtx.NewTransactor(pool)
	lesson := addLesson(t, repo.LessonRepository, tx, 1)
	reserve := application.NewReserveSeat(repo, discardOutbox{},
		outbox.NewProcessedEvents(pool), tx, fixedClock{now})

	errs := make([]error, 2)
	var done sync.WaitGroup
	for i := range errs {
		done.Go(func() {
			errs[i] = reserve.Execute(
				t.Context(), reserveInput(lesson))
		})
	}
	done.Wait()

	reserved, full := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			reserved++
		case errors.Is(err, domain.ErrLessonFull):
			full++
		default:
			t.Errorf("想定外のエラー: %v", err)
		}
	}
	if reserved != 1 || full != 1 {
		t.Fatalf("確保 %d 件, 満席 %d 件: %v", reserved, full, errs)
	}
	// 版の競合を経たことを確かめる。競合が無ければ検査になっていない
	if n := repo.conflicts.Load(); n != 1 {
		t.Errorf("版の競合 = %d 回", n)
	}
	got := seats(t, repo.LessonRepository, lesson)
	if got != 1 {
		t.Errorf("seats = %d", got)
	}
}

// 定員より多い予約が同時に来ても、定員ちょうどが確保される。
func TestReserveSeat_ConcurrentUpToCapacity(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations, outbox.Migrations)
	repo := postgres.NewLessonRepository(pool)
	tx := pgtx.NewTransactor(pool)
	const capacity, requests = 3, 8
	lesson := addLesson(t, repo, tx, capacity)
	reserve := application.NewReserveSeat(repo, discardOutbox{},
		outbox.NewProcessedEvents(pool), tx, fixedClock{now})

	errs := make([]error, requests)
	var done sync.WaitGroup
	for i := range errs {
		done.Go(func() {
			errs[i] = reserve.Execute(
				t.Context(), reserveInput(lesson))
		})
	}
	done.Wait()

	reserved := 0
	for _, err := range errs {
		switch {
		case err == nil:
			reserved++
		case errors.Is(err, domain.ErrLessonFull):
		default:
			t.Errorf("想定外のエラー: %v", err)
		}
	}
	if reserved != capacity {
		t.Errorf("確保 = %d", reserved)
	}
	if n := seats(t, repo, lesson); n != capacity {
		t.Errorf("seats = %d", n)
	}
}

// failsOnceOutbox は最初の Append を失敗させる。席の保存の後で
// 止まった状況を作る。
type failsOnceOutbox struct {
	application.OutboxWriter
	failed atomic.Bool
}

func (o *failsOnceOutbox) Append(
	ctx context.Context, events ...domain.DomainEvent,
) error {
	if o.failed.CompareAndSwap(false, true) {
		return errors.New("outbox")
	}
	return o.OutboxWriter.Append(ctx, events...)
}

// 処理の途中で止まれば席も処理済みの記録も戻る。再配信で1席だけ動く。
func TestReserveSeat_RedeliveryAfterFailureOnPostgres(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations, outbox.Migrations)
	repo := postgres.NewLessonRepository(pool)
	tx := pgtx.NewTransactor(pool)
	lesson := addLesson(t, repo, tx, 2)
	writer := postgres.NewOutboxWriter(outbox.NewStore(pool))
	reserve := application.NewReserveSeat(repo,
		&failsOnceOutbox{OutboxWriter: writer},
		outbox.NewProcessedEvents(pool), tx, fixedClock{now})
	in := reserveInput(lesson)

	err := reserve.Execute(t.Context(), in)
	if err == nil || errors.Is(err, domain.ErrLessonFull) {
		t.Fatalf("1回目: %v", err)
	}
	if n := seats(t, repo, lesson); n != 0 {
		t.Fatalf("途中で止まった席が残った: %d", n)
	}
	for range 2 {
		if err := reserve.Execute(t.Context(), in); err != nil {
			t.Fatalf("再配信: %v", err)
		}
	}
	if n := seats(t, repo, lesson); n != 1 {
		t.Errorf("seats = %d", n)
	}
	var events int
	err = pool.QueryRow(t.Context(),
		`SELECT count(*) FROM outbox WHERE event_type = $1`,
		contract.EventSeatReserved).Scan(&events)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if events != 1 {
		t.Errorf("結果のイベント = %d 件", events)
	}
}

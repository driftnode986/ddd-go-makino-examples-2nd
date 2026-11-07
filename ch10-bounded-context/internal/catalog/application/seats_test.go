package application_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"example.com/trainingbooking/internal/catalog/application"
	"example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/catalog/infrastructure/memory"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

type memoryDeps struct {
	repo      *memory.LessonRepository
	outbox    *memory.OutboxWriter
	processed *outbox.MemoryProcessedEvents
	tx        *memtx.Transactor
}

func newDeps() memoryDeps {
	return memoryDeps{
		repo:      memory.NewLessonRepository(),
		outbox:    memory.NewOutboxWriter(),
		processed: outbox.NewMemoryProcessedEvents(),
		tx:        memtx.NewTransactor(),
	}
}

func (d memoryDeps) reserve() *application.ReserveSeat {
	return application.NewReserveSeat(
		d.repo, d.outbox, d.processed, d.tx, fixedClock{now})
}

func (d memoryDeps) release() *application.ReleaseSeat {
	return application.NewReleaseSeat(d.repo, d.processed, d.tx)
}

func reserveInput(lesson ids.LessonID) application.ReserveSeatInput {
	return application.ReserveSeatInput{
		EventID:   uuid.New().String(),
		LessonID:  lesson,
		BookingID: ids.NewID[ids.BookingMarker](),
	}
}

func TestReserveSeat_SavesSeatAndRecordsEvent(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 1)
	in := reserveInput(lesson)
	if _, err := d.reserve().Execute(t.Context(), in); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if n := seats(t, d.repo, lesson); n != 1 {
		t.Errorf("seats = %d", n)
	}
	events := d.outbox.Events()
	if len(events) != 1 {
		t.Fatalf("events = %v", events)
	}
	got, ok := events[0].(domain.SeatReserved)
	if !ok || !got.BookingID.Equal(in.BookingID) {
		t.Errorf("event = %#v", events[0])
	}
}

// 満席は Tx を戻さず、SeatRejected を残して rejected を返す。
func TestReserveSeat_FullRecordsRejection(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 1)
	_, err := d.reserve().Execute(t.Context(), reserveInput(lesson))
	if err != nil {
		t.Fatalf("1席目: %v", err)
	}
	in := reserveInput(lesson)
	rejected, err := d.reserve().Execute(t.Context(), in)
	if !rejected || err != nil {
		t.Fatalf("rejected = %v, err = %v", rejected, err)
	}
	if n := seats(t, d.repo, lesson); n != 1 {
		t.Errorf("seats = %d", n)
	}
	events := d.outbox.Events()
	if len(events) != 2 {
		t.Fatalf("events = %v", events)
	}
	got, ok := events[1].(domain.SeatRejected)
	if !ok || !got.BookingID.Equal(in.BookingID) ||
		got.Reason != domain.RejectLessonFull {
		t.Errorf("event = %#v", events[1])
	}
}

// レッスンが無いときも拒否を伝え、予約を pending のまま残さない。
func TestReserveSeat_MissingLessonRecordsRejection(t *testing.T) {
	d := newDeps()
	in := reserveInput(ids.NewID[ids.LessonMarker]())
	rejected, err := d.reserve().Execute(t.Context(), in)
	if !rejected || err != nil {
		t.Fatalf("rejected = %v, err = %v", rejected, err)
	}
	events := d.outbox.Events()
	if len(events) != 1 {
		t.Fatalf("events = %v", events)
	}
	got, ok := events[0].(domain.SeatRejected)
	if !ok || got.Reason != domain.RejectLessonNotFound ||
		!got.OccurredAt.Equal(now) {
		t.Errorf("event = %#v", events[0])
	}
}

// conflictOnce は最初の Save だけ版の競合を返す。
type conflictOnce struct {
	domain.LessonRepository
	saves int
}

func (r *conflictOnce) Save(
	ctx context.Context, l *domain.Lesson,
) error {
	r.saves++
	if r.saves == 1 {
		return domain.ErrVersionConflict
	}
	return r.LessonRepository.Save(ctx, l)
}

func TestReserveSeat_RetriesVersionConflict(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 1)
	repo := &conflictOnce{LessonRepository: d.repo}
	reserve := application.NewReserveSeat(
		repo, d.outbox, d.processed, d.tx, fixedClock{now})
	_, err := reserve.Execute(t.Context(), reserveInput(lesson))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repo.saves != 2 {
		t.Errorf("saves = %d", repo.saves)
	}
	if len(d.outbox.Events()) != 1 {
		t.Errorf("events = %v", d.outbox.Events())
	}
}

// alwaysConflict は常に版の競合を返す。
type alwaysConflict struct {
	domain.LessonRepository
	saves int
}

func (r *alwaysConflict) Save(
	context.Context, *domain.Lesson,
) error {
	r.saves++
	return domain.ErrVersionConflict
}

// 再試行には上限がある。満席に変えず競合のまま返す。
func TestReserveSeat_GivesUpWithConflict(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 1)
	repo := &alwaysConflict{LessonRepository: d.repo}
	reserve := application.NewReserveSeat(
		repo, d.outbox, d.processed, d.tx, fixedClock{now})
	rejected, err := reserve.Execute(
		t.Context(), reserveInput(lesson))
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("err = %v", err)
	}
	if rejected {
		t.Error("満席に変わった")
	}
	if repo.saves < 2 {
		t.Errorf("再試行していない: %d", repo.saves)
	}
	if len(d.outbox.Events()) != 0 {
		t.Errorf("events = %v", d.outbox.Events())
	}
}

func TestReleaseSeat_FreesTheSeat(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 1)
	in := reserveInput(lesson)
	if _, err := d.reserve().Execute(t.Context(), in); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	release := application.ReleaseSeatInput{
		EventID:   uuid.New().String(),
		LessonID:  lesson,
		BookingID: in.BookingID,
	}
	for range 2 {
		err := d.release().Execute(t.Context(), release)
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
		if n := seats(t, d.repo, lesson); n != 0 {
			t.Fatalf("seats = %d", n)
		}
	}
}

// 無いレッスンには戻す席も無い。再配信を止めないように成功にする。
func TestReleaseSeat_MissingLessonIsNoOp(t *testing.T) {
	d := newDeps()
	err := d.release().Execute(t.Context(),
		application.ReleaseSeatInput{
			EventID:   uuid.New().String(),
			LessonID:  ids.NewID[ids.LessonMarker](),
			BookingID: ids.NewID[ids.BookingMarker](),
		})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestLessonReader_ReturnsContractLesson(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 2)
	got, err := application.NewLessonReader(d.repo).FindLesson(
		t.Context(), lesson)
	if err != nil {
		t.Fatalf("FindLesson: %v", err)
	}
	if !got.ID.Equal(lesson) || got.FeeYen != 3000 ||
		!got.Start.Equal(start) || got.Duration.Hours() != 1 ||
		len(got.Options) != 1 || got.Options[0].UnitPriceYen != 300 {
		t.Errorf("got = %+v", got)
	}
}

func TestLessonReader_MissingIsContractError(t *testing.T) {
	d := newDeps()
	_, err := application.NewLessonReader(d.repo).FindLesson(
		t.Context(), ids.NewID[ids.LessonMarker]())
	if !errors.Is(err, contract.ErrLessonNotFound) {
		t.Fatalf("err = %v", err)
	}
}

// 処理済みの記録で止めるのは同じイベントである。別のイベントなら
// 処理して、席を2重に取らないのは Lesson の役割である。
func TestReserveSeat_SameEventRunsOnce(t *testing.T) {
	d := newDeps()
	lesson := addLesson(t, d.repo, d.tx, 2)
	in := reserveInput(lesson)
	for range 2 {
		_, err := d.reserve().Execute(t.Context(), in)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
	}
	if len(d.outbox.Events()) != 1 {
		t.Errorf("events = %v", d.outbox.Events())
	}
	in.EventID = uuid.New().String()
	if _, err := d.reserve().Execute(t.Context(), in); err != nil {
		t.Fatalf("別のイベント: %v", err)
	}
	if n := seats(t, d.repo, lesson); n != 1 {
		t.Errorf("seats = %d", n)
	}
}

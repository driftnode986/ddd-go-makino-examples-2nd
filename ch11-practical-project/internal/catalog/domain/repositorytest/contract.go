// Package repositorytest は LessonRepository の契約テストである。
// in-memory と PostgreSQL の両方に同じテストを流し、
// errors.Is の結果まで一致させる。
package repositorytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
	"github.com/google/go-cmp/cmp"
)

var jst = time.FixedZone("JST", 9*60*60)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, jst),
}

// Transactor は契約テストが使う Tx の境界である。
type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// Fixture は契約テストに渡す部品である。
type Fixture struct {
	Repo domain.LessonRepository
	Tx   Transactor
}

func (f Fixture) add(t *testing.T, l *domain.Lesson) error {
	t.Helper()
	return f.Tx.Do(t.Context(), func(ctx context.Context) error {
		return f.Repo.Add(ctx, l)
	})
}

func (f Fixture) save(t *testing.T, l *domain.Lesson) error {
	t.Helper()
	return f.Tx.Do(t.Context(), func(ctx context.Context) error {
		return f.Repo.Save(ctx, l)
	})
}

// Run は契約テストを実行する。fixture はテストごとに空の部品を返す。
func Run(t *testing.T, fixture func(t *testing.T) Fixture) {
	t.Helper()
	cases := []struct {
		name string
		run  func(*testing.T, Fixture)
	}{
		{"往復で全項目が一致する", testRoundTrip},
		{"席とオプションなしでも往復できる", testRoundTripEmpty},
		{"重複した Add は ErrDuplicate", testDuplicateAdd},
		{"無いレッスンの FindByID は ErrNotFound", testFindMissing},
		{"無いレッスンの Save は ErrNotFound", testSaveMissing},
		{"2者の競合は後者が ErrVersionConflict", testConflict},
		{"席の変更が保存される", testSeatsSaved},
		{"変更なしの Save は成功する", testNoChangeSave},
		{"同じオブジェクトの2度目の Save は競合", testSaveTwice},
		{"読み出した集約を変えても保存値は変わらない", testNoSharing},
		{"Tx の外の書き込みは ErrNoTransaction", testNoTransaction},
		{"保存できない集約は ErrInvalidLesson", testInvalid},
		{"Tx の失敗で書き込みが戻る", testRollback},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.run(t, fixture(t)) })
	}
}

func newLesson(t *testing.T, capacity int) *domain.Lesson {
	t.Helper()
	l, err := domain.NewLesson(domain.LessonSpec{
		ID:       ids.NewID[ids.LessonMarker](),
		Title:    "朝のヨガ",
		Start:    time.Date(2030, time.June, 1, 10, 0, 0, 0, jst),
		Duration: time.Hour,
		FeeYen:   3000,
		Capacity: capacity,
		Options: []domain.LessonOption{
			{Name: "シューズ", UnitPriceYen: 300},
			{Name: "タオル", UnitPriceYen: 100},
		},
	})
	if err != nil {
		t.Fatalf("NewLesson: %v", err)
	}
	return l
}

func reserve(t *testing.T, l *domain.Lesson) ids.BookingID {
	t.Helper()
	id := ids.NewID[ids.BookingMarker]()
	if err := l.ReserveSeat(id, clock); err != nil {
		t.Fatalf("ReserveSeat: %v", err)
	}
	return id
}

func find(t *testing.T, f Fixture, id ids.LessonID) *domain.Lesson {
	t.Helper()
	l, err := f.Repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	return l
}

func testRoundTrip(t *testing.T, f Fixture) {
	l := newLesson(t, 3)
	reserve(t, l)
	reserve(t, l)
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got := find(t, f, l.ID())
	if !got.Equal(l) {
		t.Error("Equal が false")
	}
	if diff := cmp.Diff(l.State(), got.State()); diff != "" {
		t.Errorf("往復の差分 (-want +got):\n%s", diff)
	}
	if len(got.PullEvents()) != 0 {
		t.Error("読み出しでイベントが記録された")
	}
}

func testRoundTripEmpty(t *testing.T, f Fixture) {
	l, err := domain.NewLesson(domain.LessonSpec{
		ID:       ids.NewID[ids.LessonMarker](),
		Title:    "夜のヨガ",
		Start:    time.Date(2030, time.June, 1, 19, 0, 0, 0, jst),
		Duration: time.Hour,
		FeeYen:   2000,
		Capacity: 1,
	})
	if err != nil {
		t.Fatalf("NewLesson: %v", err)
	}
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got := find(t, f, l.ID()).State()
	if len(got.Seats) != 0 || len(got.Options) != 0 {
		t.Errorf("got = %+v", got)
	}
}

func testDuplicateAdd(t *testing.T, f Fixture) {
	l := newLesson(t, 1)
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	err := f.add(t, l)
	if !errors.Is(err, domain.ErrDuplicate) {
		t.Fatalf("err = %v", err)
	}
}

func testFindMissing(t *testing.T, f Fixture) {
	_, err := f.Repo.FindByID(
		t.Context(), ids.NewID[ids.LessonMarker]())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func testSaveMissing(t *testing.T, f Fixture) {
	err := f.save(t, newLesson(t, 1))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func testConflict(t *testing.T, f Fixture) {
	l := newLesson(t, 2)
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	first, second := find(t, f, l.ID()), find(t, f, l.ID())
	reserve(t, first)
	reserve(t, second)
	if err := f.save(t, first); err != nil {
		t.Fatalf("先の Save: %v", err)
	}
	err := f.save(t, second)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("err = %v", err)
	}
	if got := find(t, f, l.ID()); got.ReservedSeats() != 1 {
		t.Errorf("ReservedSeats = %d", got.ReservedSeats())
	}
}

func testSeatsSaved(t *testing.T, f Fixture) {
	l := newLesson(t, 2)
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	loaded := find(t, f, l.ID())
	first := reserve(t, loaded)
	second := reserve(t, loaded)
	if err := f.save(t, loaded); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := find(t, f, l.ID())
	if got.Version() != 1 {
		t.Errorf("Version = %d", got.Version())
	}
	want := []ids.BookingID{first, second}
	if diff := cmp.Diff(want, got.State().Seats); diff != "" {
		t.Errorf("席の差分 (-want +got):\n%s", diff)
	}
	if err := got.ReleaseSeat(first); err != nil {
		t.Fatalf("ReleaseSeat: %v", err)
	}
	if err := f.save(t, got); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if n := find(t, f, l.ID()).ReservedSeats(); n != 1 {
		t.Errorf("戻した後の席 = %d", n)
	}
}

func testNoChangeSave(t *testing.T, f Fixture) {
	l := newLesson(t, 1)
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := f.save(t, find(t, f, l.ID())); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func testSaveTwice(t *testing.T, f Fixture) {
	l := newLesson(t, 1)
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	loaded := find(t, f, l.ID())
	if err := f.save(t, loaded); err != nil {
		t.Fatalf("Save: %v", err)
	}
	err := f.save(t, loaded)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("err = %v", err)
	}
}

func testNoSharing(t *testing.T, f Fixture) {
	l := newLesson(t, 2)
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	reserve(t, l)
	loaded := find(t, f, l.ID())
	reserve(t, loaded)
	if n := find(t, f, l.ID()).ReservedSeats(); n != 0 {
		t.Errorf("保存値が変わった: %d", n)
	}
}

func testNoTransaction(t *testing.T, f Fixture) {
	l := newLesson(t, 1)
	err := f.Repo.Add(t.Context(), l)
	if !errors.Is(err, txn.ErrNoTransaction) {
		t.Fatalf("Add: %v", err)
	}
	err = f.Repo.Save(t.Context(), l)
	if !errors.Is(err, txn.ErrNoTransaction) {
		t.Fatalf("Save: %v", err)
	}
}

func testInvalid(t *testing.T, f Fixture) {
	err := f.add(t, &domain.Lesson{})
	if !errors.Is(err, domain.ErrInvalidLesson) {
		t.Fatalf("Add: %v", err)
	}
	err = f.save(t, &domain.Lesson{})
	if !errors.Is(err, domain.ErrInvalidLesson) {
		t.Fatalf("Save: %v", err)
	}
}

func testRollback(t *testing.T, f Fixture) {
	l := newLesson(t, 2)
	if err := f.add(t, l); err != nil {
		t.Fatalf("Add: %v", err)
	}
	added := newLesson(t, 1)
	boom := errors.New("boom")
	err := f.Tx.Do(t.Context(), func(ctx context.Context) error {
		if err := f.Repo.Add(ctx, added); err != nil {
			return err
		}
		loaded, err := f.Repo.FindByID(ctx, l.ID())
		if err != nil {
			return err
		}
		if err := loaded.ReserveSeat(
			ids.NewID[ids.BookingMarker](), clock); err != nil {
			return err
		}
		if err := f.Repo.Save(ctx, loaded); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Do: %v", err)
	}
	_, err = f.Repo.FindByID(t.Context(), added.ID())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Add が残った: %v", err)
	}
	got := find(t, f, l.ID())
	if got.ReservedSeats() != 0 || got.Version() != 0 {
		t.Errorf("Save が残った: %+v", got.State())
	}
}

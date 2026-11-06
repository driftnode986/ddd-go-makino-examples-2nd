//go:build integration

// Package pitfalls_test は第7章の失敗の例を動かす。
package pitfalls_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/memory"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres/sqlcgen"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/memtx"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var jst = time.FixedZone("JST", 9*60*60)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func newBooking(
	t *testing.T, member ids.MemberID,
) *domain.Booking {
	t.Helper()
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](), member,
		ids.NewID[ids.LessonMarker](), slot, fee,
		fixedClock{now: start.AddDate(0, -1, 0)})
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

// insert はエラーを変換せずに親を保存する。
func insert(
	ctx context.Context, db sqlcgen.DBTX, b *domain.Booking,
) error {
	s := b.State()
	return sqlcgen.New(db).InsertBooking(ctx,
		sqlcgen.InsertBookingParams{
			ID:          s.ID.String(),
			MemberID:    s.MemberID.String(),
			LessonID:    s.LessonID.String(),
			StartsAt:    s.Start,
			EndsAt:      s.End,
			FeeYen:      s.FeeYen,
			DiscountYen: s.DiscountYen,
			Status:      string(s.Status),
			Version:     int64(s.Version),
		})
}

// #region raw-error

func TestUntranslatedErrorDiffersFromMemory(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	ctx := t.Context()
	b := newBooking(t, ids.NewID[ids.MemberMarker]())

	mem := memory.NewBookingRepository()
	tx := memtx.NewTransactor()
	add := func(ctx context.Context) error { return mem.Add(ctx, b) }
	_ = tx.Do(ctx, add)
	err := tx.Do(ctx, add)
	t.Logf("in-memory: %v", err)
	t.Logf("  ErrDuplicate か: %v",
		errors.Is(err, domain.ErrDuplicate))

	_ = insert(ctx, pool, b)
	err = insert(ctx, pool, b)
	t.Logf("変換なし: %v", err)
	t.Logf("  ErrDuplicate か: %v",
		errors.Is(err, domain.ErrDuplicate))
}

// #endregion

// #region no-btree-gist

func TestExclusionNeedsBtreeGist(t *testing.T) {
	pool := dbtest.New(t)
	_, err := pool.Exec(t.Context(), `CREATE TABLE t (
		member_id uuid NOT NULL,
		starts_at timestamptz NOT NULL,
		ends_at   timestamptz NOT NULL,
		EXCLUDE USING gist (
			member_id WITH =,
			tstzrange(starts_at, ends_at) WITH &&
		))`)
	t.Logf("%v", err)
}

// #endregion

// activeSlots は会員の有効な予約の時間帯を読む。
func activeSlots(
	ctx context.Context, tx pgx.Tx, member ids.MemberID,
) ([]domain.TimeSlot, error) {
	rows, err := tx.Query(ctx, `SELECT starts_at, ends_at
		FROM bookings WHERE member_id = $1
		AND status IN ('pending', 'confirmed')`, member.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var slots []domain.TimeSlot
	for rows.Next() {
		var start, end time.Time
		if err := rows.Scan(&start, &end); err != nil {
			return nil, err
		}
		slot, err := domain.NewTimeSlot(start, end.Sub(start), jst)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

// #region check-then-save

func TestCheckThenSaveLetsBothIn(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	ctx := t.Context()
	_, err := pool.Exec(ctx, `ALTER TABLE bookings
		DROP CONSTRAINT bookings_no_overlap`) // 制約を外す
	if err != nil {
		t.Fatalf("DROP: %v", err)
	}
	member := ids.NewID[ids.MemberMarker]()
	a, b := newBooking(t, member), newBooking(t, member)
	txA, _ := pool.Begin(ctx)
	defer func() { _ = txA.Rollback(ctx) }()
	txB, _ := pool.Begin(ctx)
	defer func() { _ = txB.Rollback(ctx) }()

	// 2つの処理が、どちらも保存の前に重なりを判定する
	slotsA, _ := activeSlots(ctx, txA, member)
	slotsB, _ := activeSlots(ctx, txB, member)
	t.Logf("A の判定: %v", domain.DetectOverlap(a.Slot(), slotsA))
	t.Logf("B の判定: %v", domain.DetectOverlap(b.Slot(), slotsB))

	t.Logf("A の保存: %v", insert(ctx, txA, a))
	t.Logf("A のコミット: %v", txA.Commit(ctx))
	t.Logf("B の保存: %v", insert(ctx, txB, b))
	t.Logf("B のコミット: %v", txB.Commit(ctx))
	t.Logf("有効な予約: %d 件", countActive(t, pool, member))
}

// #endregion

func countActive(
	t *testing.T, pool *pgxpool.Pool, member ids.MemberID,
) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(), `SELECT count(*) FROM bookings
		WHERE member_id = $1
		AND status IN ('pending', 'confirmed')`,
		member.String()).Scan(&n)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// #region with-constraint

func TestConstraintStopsTheSecond(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	ctx := t.Context()
	member := ids.NewID[ids.MemberMarker]()
	add := func(b *domain.Booking) error {
		return tx.Do(ctx, func(ctx context.Context) error {
			return repo.Add(ctx, b)
		})
	}
	t.Logf("A: %v", add(newBooking(t, member)))
	err := add(newBooking(t, member))
	t.Logf("B: %v", err)
	t.Logf("B は ErrOverlap か: %v",
		errors.Is(err, domain.ErrOverlap))
}

// #endregion

// #region time-zone

func TestTimeZoneAfterRoundTrip(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	b := newBooking(t, ids.NewID[ids.MemberMarker]())
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		return repo.Add(ctx, b)
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := repo.FindByID(t.Context(), b.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	saved, read := b.State().Start, got.State().Start
	t.Logf("保存: %s", saved)
	t.Logf("読み戻し: %s", read)
	t.Logf("==: %v  Equal: %v", saved == read, saved.Equal(read))
}

// #endregion

// #region defer-close

// openPool は defer で閉じる。呼び出し元に戻った時点で閉じている。
func openPool(
	t *testing.T, base *pgxpool.Pool,
) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.NewWithConfig(t.Context(), base.Config())
	if err != nil {
		t.Fatalf("接続: %v", err)
	}
	defer pool.Close()
	return pool
}

func TestDeferClosesBeforeTheTest(t *testing.T) {
	base := dbtest.New(t, postgres.Migrations)
	pool := openPool(t, base)
	t.Logf("Ping: %v", pool.Ping(t.Context()))
}

// #endregion

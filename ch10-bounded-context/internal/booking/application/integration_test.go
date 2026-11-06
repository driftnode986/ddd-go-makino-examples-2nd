//go:build integration

package application_test

import (
	"errors"
	"sync"
	"testing"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/application/querytest"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestBookingQueries_Postgres(t *testing.T) {
	querytest.Run(t, func(t *testing.T) querytest.Fixture {
		pool := dbtest.New(t, postgres.Migrations)
		return querytest.Fixture{
			Repo:    postgres.NewBookingRepository(pool),
			Tx:      pgtx.NewTransactor(pool),
			Queries: postgres.NewBookingQueries(pool),
		}
	})
}

// outbox の失敗で Tx が戻り、予約も残らないことを実 DB で確かめる。
func TestPlaceBooking_RollsBackWhenOutboxFails(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	boom := errors.New("outbox")
	in := placeInput()

	failing := application.NewPlaceBooking(
		repo, postgres.NewBookingQueries(pool), failingOutbox{boom},
		tx, fixedClock{now}, lessons(t))
	_, err := failing.Execute(t.Context(), in)
	wantKind(t, err, application.KindInternal)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}

	member, err := ids.ParseID[ids.MemberMarker](in.MemberID)
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	queries := postgres.NewBookingQueries(pool)
	views, err := queries.ListByMember(t.Context(), member)
	if err != nil {
		t.Fatalf("ListByMember: %v", err)
	}
	if len(views) != 0 {
		t.Fatalf("予約が残った: %+v", views)
	}
}

// 同じ会員の重なる予約を同時に受けても、1件だけ成功する。
func TestPlaceBooking_ConcurrentOverlapOnPostgres(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	place := application.NewPlaceBooking(
		postgres.NewBookingRepository(pool),
		postgres.NewBookingQueries(pool), discardOutbox{},
		pgtx.NewTransactor(pool), fixedClock{now}, lessons(t))
	in := placeInput()

	errs := make([]error, 2)
	var done sync.WaitGroup
	start := make(chan struct{})
	for i := range errs {
		done.Go(func() {
			<-start
			_, errs[i] = place.Execute(t.Context(), in)
		})
	}
	close(start)
	done.Wait()

	ok, conflict := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, domain.ErrOverlap):
			wantKind(t, err, application.KindConflict)
			conflict++
		default:
			t.Fatalf("想定外のエラー: %v", err)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("成功 %d 件, 競合 %d 件", ok, conflict)
	}
}

// 順に来た重なる予約は、Add の前に読み取りと DetectOverlap で止まる。
func TestPlaceBooking_DetectsOverlapOnPostgres(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	repo := postgres.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	queries := postgres.NewBookingQueries(pool)
	first := application.NewPlaceBooking(
		repo, queries, discardOutbox{}, tx,
		fixedClock{now}, lessons(t))
	in := placeInput()
	if _, err := first.Execute(t.Context(), in); err != nil {
		t.Fatalf("1件目: %v", err)
	}
	// Add まで進めば boom になる
	boom := errors.New("add")
	second := application.NewPlaceBooking(
		failingRepository{repo, boom},
		queries, discardOutbox{}, tx,
		fixedClock{now}, lessons(t))
	in.LessonID = ids.NewID[ids.LessonMarker]().String()
	_, err := second.Execute(t.Context(), in)
	wantKind(t, err, application.KindConflict)
	if !errors.Is(err, domain.ErrOverlap) {
		t.Fatalf("err = %v", err)
	}
}

// 読み取りで見えなかった予約は、排他制約が ErrOverlap で止める。
func TestPlaceBooking_StaleReadIsStoppedByConstraint(t *testing.T) {
	pool := dbtest.New(t, postgres.Migrations)
	place := application.NewPlaceBooking(
		postgres.NewBookingRepository(pool),
		staleQueries{postgres.NewBookingQueries(pool)},
		discardOutbox{},
		pgtx.NewTransactor(pool),
		fixedClock{now}, lessons(t))
	in := placeInput()
	if _, err := place.Execute(t.Context(), in); err != nil {
		t.Fatalf("1件目: %v", err)
	}
	in.LessonID = ids.NewID[ids.LessonMarker]().String()
	_, err := place.Execute(t.Context(), in)
	wantKind(t, err, application.KindConflict)
	if !errors.Is(err, domain.ErrOverlap) {
		t.Fatalf("err = %v", err)
	}
	if _, ok := errors.AsType[*pgconn.PgError](err); !ok {
		t.Errorf("排他制約のエラーでない: %v", err)
	}
}

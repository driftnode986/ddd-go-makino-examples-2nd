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

	failing := application.NewPlaceBooking(repo, failingOutbox{boom},
		tx, fixedClock{now}, terms(t))
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
		discardOutbox{},
		pgtx.NewTransactor(pool),
		fixedClock{now}, terms(t))
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

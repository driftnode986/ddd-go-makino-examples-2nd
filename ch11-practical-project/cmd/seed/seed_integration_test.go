//go:build integration

package main

import (
	"testing"
	"time"

	bookingpg "example.com/trainingbooking/internal/booking/infrastructure/postgres"
	catalogpg "example.com/trainingbooking/internal/catalog/infrastructure/postgres"
	identitypg "example.com/trainingbooking/internal/identity/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
)

// 2回目の実行は既存の行を変えない。レッスンの日時も最初の実行で決まる。
func TestSeed_SecondRunKeepsExistingRows(t *testing.T) {
	pool := dbtest.New(t, bookingpg.Migrations,
		catalogpg.Migrations, outbox.Migrations,
		identitypg.Migrations)
	first := time.Date(2030, time.May, 1, 0, 0, 0, 0, time.UTC)
	if err := seed(t.Context(), pool, first); err != nil {
		t.Fatalf("1回目: %v", err)
	}
	later := first.AddDate(0, 0, 7)
	if err := seed(t.Context(), pool, later); err != nil {
		t.Fatalf("2回目: %v", err)
	}
	id, err := ids.ParseID[ids.LessonMarker](seedLessons[0].id)
	if err != nil {
		t.Fatal(err)
	}
	repo := catalogpg.NewLessonRepository(pool)
	l, err := repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	want, err := newLesson(seedLessons[0], first)
	if err != nil {
		t.Fatal(err)
	}
	if !l.State().Start.Equal(want.State().Start) {
		t.Errorf("start = %v, want %v",
			l.State().Start, want.State().Start)
	}
	if len(l.State().Options) != 2 {
		t.Errorf("options = %v", l.State().Options)
	}
	var members int
	err = pool.QueryRow(t.Context(),
		"SELECT count(*) FROM members").Scan(&members)
	if err != nil || members != len(seedMembers) {
		t.Errorf("members = %d, err = %v", members, err)
	}
}

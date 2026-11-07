// Command seed は動作確認用の会員とレッスンを保存する。
// SQL で直接入れず、本番のコードと同じドメインの検証を通す。
// 何度実行しても既存の行は変えない。
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	catalog "example.com/trainingbooking/internal/catalog/domain"
	catalogpg "example.com/trainingbooking/internal/catalog/infrastructure/postgres"
	identity "example.com/trainingbooking/internal/identity/domain"
	identitypg "example.com/trainingbooking/internal/identity/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultDatabaseURL = "postgres://dev:dev@localhost:5432/" +
	"trainingbooking?sslmode=disable"

func main() {
	ctx := context.Background()
	url := cmp.Or(os.Getenv("DATABASE_URL"), defaultDatabaseURL)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	err = seed(ctx, pool, time.Now())
	pool.Close()
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	fmt.Println("seed: done")
}

// seed は会員とレッスンを保存する。レッスンの日時は now の30日後の
// 日本時間で作るので、最初に実行した日で決まる。
func seed(
	ctx context.Context, pool *pgxpool.Pool, now time.Time,
) error {
	tx := pgtx.NewTransactor(pool)
	members := identitypg.NewMemberRepository(pool)
	for _, m := range seedMembers {
		member, err := newMember(m)
		if err != nil {
			return fmt.Errorf("member %s: %w", m.email, err)
		}
		err = tx.Do(ctx, func(ctx context.Context) error {
			return members.Add(ctx, member)
		})
		if err != nil && !errors.Is(err, identity.ErrDuplicate) {
			return fmt.Errorf("member %s: %w", m.email, err)
		}
	}
	lessons := catalogpg.NewLessonRepository(pool)
	for _, l := range seedLessons {
		lesson, err := newLesson(l, now)
		if err != nil {
			return fmt.Errorf("lesson %s: %w", l.title, err)
		}
		err = tx.Do(ctx, func(ctx context.Context) error {
			return lessons.Add(ctx, lesson)
		})
		if err != nil && !errors.Is(err, catalog.ErrDuplicate) {
			return fmt.Errorf("lesson %s: %w", l.title, err)
		}
	}
	return nil
}

type memberSeed struct{ id, email, name string }

var seedMembers = []memberSeed{
	{"0196f5a0-0000-7000-8000-000000000001",
		"hanako@example.com", "花子"},
	{"0196f5a0-0000-7000-8000-000000000002",
		"taro@example.com", "太郎"},
}

// #region new-member

// newMember は初期データの値を検証して Member にする。
func newMember(m memberSeed) (*identity.Member, error) {
	id, err := ids.ParseID[ids.MemberMarker](m.id)
	if err != nil {
		return nil, err
	}
	email, err := identity.NewEmail(m.email)
	if err != nil {
		return nil, err
	}
	return identity.NewMember(id, email, m.name)
}

// #endregion

type lessonSeed struct {
	id       string
	title    string
	hour     int
	feeYen   int64
	capacity int
	options  []catalog.LessonOption
}

var seedLessons = []lessonSeed{
	// 席に余裕があるレッスン
	{"0196f5a0-0000-7000-8001-000000000001", "朝のヨガ", 10, 3000, 10,
		[]catalog.LessonOption{
			{Name: "シューズ", UnitPriceYen: 300},
			{Name: "タオル", UnitPriceYen: 100},
		}},
	// 定員1のレッスン。満席の拒否を確かめる
	{"0196f5a0-0000-7000-8001-000000000002", "少人数ピラティス", 14,
		5000, 1, nil},
	// 朝のヨガと同じ時間帯のレッスン。重なる予約を確かめる
	{"0196f5a0-0000-7000-8001-000000000003", "朝の筋力トレーニング", 10,
		3000, 10, nil},
}

// #region new-lesson

// newLesson は初期データの値を検証して Lesson にする。
func newLesson(
	l lessonSeed, now time.Time,
) (*catalog.Lesson, error) {
	id, err := ids.ParseID[ids.LessonMarker](l.id)
	if err != nil {
		return nil, err
	}
	jst := time.FixedZone("JST", 9*60*60)
	y, m, d := now.In(jst).Date()
	return catalog.NewLesson(catalog.LessonSpec{
		ID:       id,
		Title:    l.title,
		Start:    time.Date(y, m, d+30, l.hour, 0, 0, 0, jst),
		Duration: time.Hour,
		FeeYen:   l.feeYen,
		Capacity: l.capacity,
		Options:  l.options,
	})
}

// #endregion

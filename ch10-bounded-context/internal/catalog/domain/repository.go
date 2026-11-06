package domain

import (
	"context"

	"example.com/trainingbooking/internal/ids"
)

// LessonRepository は Lesson 集約を保存し、読み出す。
// Add と Save は ctx の Tx に参加する。自分では Tx を開かない。
// ctx に Tx が無ければ実装はエラー（txn.ErrNoTransaction）を返す。
type LessonRepository interface {
	// Add は新しいレッスンを保存する。同じ ID があれば
	// ErrDuplicate を返す。
	Add(ctx context.Context, l *Lesson) error
	// Save は読み込んだときの版で判定して更新する。レッスンが
	// 無ければ ErrNotFound、版が違えば ErrVersionConflict を返す。
	// l の版は変えないので、保存後に使い続けるときは読み直す。
	Save(ctx context.Context, l *Lesson) error
	// FindByID は席とオプションを含めて読み出す。Tx なしでも読める。
	// レッスンが無ければ ErrNotFound を返す。
	FindByID(
		ctx context.Context, id ids.LessonID,
	) (*Lesson, error)
}

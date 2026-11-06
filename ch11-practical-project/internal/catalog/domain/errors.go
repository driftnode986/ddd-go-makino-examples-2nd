package domain

import "errors"

// リポジトリが返すエラーである。実装の違いをここで吸収する。
var (
	ErrNotFound        = errors.New("lesson not found")
	ErrDuplicate       = errors.New("lesson already exists")
	ErrVersionConflict = errors.New("lesson version conflict")
)

// 集約の操作が返すエラーである。
var (
	ErrInvalidLesson = errors.New("invalid lesson")
	ErrLessonFull    = errors.New("lesson is full")
)

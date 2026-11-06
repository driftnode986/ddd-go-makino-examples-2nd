// Package application は Booking のユースケースである。
package application

import (
	"errors"
	"fmt"

	"example.com/trainingbooking/internal/booking/domain"
)

// Kind はエラーの種類である。presentation が HTTP ステータスに変える。
type Kind string

// エラーの種類である。
const (
	KindInvalid  Kind = "invalid"
	KindNotFound Kind = "not_found"
	KindConflict Kind = "conflict"
	KindInternal Kind = "internal"
)

// ApplicationError はユースケースが返すエラーである。
// Message は外に出してよい文、Cause はログに記録する原因である。
type ApplicationError struct {
	Kind    Kind
	Message string
	Cause   error
}

// Error はログ用に原因を含めた文字列を返す。外には出さない。
func (e *ApplicationError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Cause)
}

// Unwrap は原因を返す。
func (e *ApplicationError) Unwrap() error { return e.Cause }

func invalid(message string, cause error) error {
	return &ApplicationError{
		Kind:    KindInvalid,
		Message: message,
		Cause:   cause,
	}
}

// messageInternal は原因を問わず外に出す文である。
const messageInternal = "内部エラーが発生しました"

// translate はドメインとリポジトリのエラーを変換する。
// 表に無いエラーは KindInternal とし、原因の文字列を外に出さない。
func translate(err error) error {
	for _, r := range translations {
		if errors.Is(err, r.target) {
			return &ApplicationError{
				Kind:    r.kind,
				Message: r.message,
				Cause:   err,
			}
		}
	}
	return &ApplicationError{
		Kind:    KindInternal,
		Message: messageInternal,
		Cause:   err,
	}
}

var translations = []struct {
	target  error
	kind    Kind
	message string
}{
	{domain.ErrNotFound, KindNotFound, "予約が見つかりません"},
	{domain.ErrDuplicate, KindConflict, "同じ予約が既にあります"},
	{domain.ErrOverlap, KindConflict, "同じ時間帯に別の予約があります"},
	{
		domain.ErrVersionConflict, KindConflict,
		"予約が他の処理で更新されました。読み直してください",
	},
	{
		domain.ErrInvalidTransition, KindConflict,
		"今の状態ではこの操作はできません",
	},
	{domain.ErrCancelTooLate, KindConflict, "キャンセルの期限を過ぎています"},
	{ErrUnknownOption, KindInvalid, "レッスンに無いオプションです"},
	{
		domain.ErrInvalidOption, KindInvalid,
		"オプションの指定が正しくありません",
	},
	{domain.ErrOptionLimit, KindInvalid, "オプションの数が上限を超えています"},
	{domain.ErrDuplicateOption, KindInvalid, "同じオプションが重複しています"},
	{
		domain.ErrChangeTooLate, KindInvalid,
		"オプションを追加できる期限を過ぎています",
	},
	{domain.ErrMoneyOverflow, KindInvalid, "金額が上限を超えています"},
}

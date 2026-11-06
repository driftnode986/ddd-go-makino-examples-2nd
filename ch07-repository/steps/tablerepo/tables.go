// Package tablerepo はテーブルごとに保存する形の失敗を見せる。
package tablerepo

import (
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// #region option-table

// OptionTable は booking_options テーブルと同じ単位の保存先である。
type OptionTable struct {
	rows map[ids.BookingID][]domain.OptionState
}

// Insert は明細を1行追加する。予約の本体を読まないので、
// 状態も件数も確かめられない。
func (o *OptionTable) Insert(
	id ids.BookingID, row domain.OptionState,
) {
	if o.rows == nil {
		o.rows = make(map[ids.BookingID][]domain.OptionState)
	}
	o.rows[id] = append(o.rows[id], row)
}

// #endregion

// Rows は予約の明細の行を返す。
func (o *OptionTable) Rows(
	id ids.BookingID,
) []domain.OptionState {
	return o.rows[id]
}

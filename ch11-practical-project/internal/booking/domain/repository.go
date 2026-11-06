package domain

import (
	"context"

	"example.com/trainingbooking/internal/ids"
)

// BookingRepository は Booking 集約を保存し、読み出す。
// Add と Save は ctx の Tx に参加する。自分では Tx を開かない。
// ctx に Tx が無ければ ErrNoTransaction を返す。
type BookingRepository interface {
	// Add は新しい予約を保存する。同じ ID があれば ErrDuplicate、
	// 同じ会員の時間帯が重なれば ErrOverlap を返す。
	Add(ctx context.Context, b *Booking) error
	// Save は読み込んだときの版で判定して更新する。予約が無ければ
	// ErrNotFound、版が違えば ErrVersionConflict を返す。
	// b の版は変えないので、保存後に使い続けるときは読み直す。
	Save(ctx context.Context, b *Booking) error
	// FindByID は明細を含めて読み出す。Tx なしでも読める。
	// 予約が無ければ ErrNotFound を返す。
	FindByID(
		ctx context.Context, id ids.BookingID,
	) (*Booking, error)
}

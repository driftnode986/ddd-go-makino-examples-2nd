//go:build integration

// Package rawrepo は排他制約とロックの動きを確かめるための保存処理を
// 持つ。PostgreSQL のエラーを変換せずにそのまま返す。
package rawrepo

import (
	"context"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres/sqlcgen"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

// Repo は Lock が false のとき会員のロックを取らない。
type Repo struct {
	Lock bool
}

// Add は ctx の Tx で親を保存する。エラーは変換しない。
func (r Repo) Add(ctx context.Context, b *domain.Booking) error {
	s := b.State()
	tx, _ := pgtx.From(ctx)
	q := sqlcgen.New(tx)
	if r.Lock {
		err := q.LockMember(ctx, s.MemberID.String())
		if err != nil {
			return err
		}
	}
	return q.InsertBooking(ctx, sqlcgen.InsertBookingParams{
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

// Save は ctx の Tx で親の状態を更新する。エラーは変換しない。
func (r Repo) Save(ctx context.Context, b *domain.Booking) error {
	s := b.State()
	tx, _ := pgtx.From(ctx)
	q := sqlcgen.New(tx)
	if r.Lock {
		err := q.LockMember(ctx, s.MemberID.String())
		if err != nil {
			return err
		}
	}
	_, err := q.UpdateBooking(ctx, sqlcgen.UpdateBookingParams{
		ID:          s.ID.String(),
		Version:     int64(s.Version),
		Status:      string(s.Status),
		DiscountYen: s.DiscountYen,
	})
	return err
}

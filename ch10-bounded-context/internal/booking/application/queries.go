package application

import (
	"context"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// #region booking-view

// BookingOptionView は一覧に載せる明細である。
type BookingOptionView struct {
	Name        string
	Quantity    int
	SubtotalYen int64
}

// BookingView は一覧のための読み取り専用の DTO である。
// Options は追加の順で、明細が無ければ空のスライスである。
type BookingView struct {
	ID       string
	LessonID string
	Start    time.Time
	End      time.Time
	Status   string
	TotalYen int64
	Options  []BookingOptionView
}

// #endregion

// #region booking-queries

// BookingQueries は予約の読み取りである。集約を組み立てない。
type BookingQueries interface {
	// ListByMember は会員の予約を開始の順に返す。
	// 予約が無ければ空のスライスを返す。
	ListByMember(
		ctx context.Context, member ids.MemberID,
	) ([]BookingView, error)
	// ActiveSlotsByMember は会員の pending と confirmed の予約の
	// 時間帯を開始の順に返す。無ければ空のスライスを返す。
	ActiveSlotsByMember(
		ctx context.Context, member ids.MemberID,
	) ([]domain.TimeSlot, error)
}

// #endregion

package application

import (
	"context"
	"time"

	"example.com/trainingbooking/internal/ids"
)

// BookingView は一覧のための読み取り専用の DTO である。
type BookingView struct {
	ID       string
	LessonID string
	Start    time.Time
	End      time.Time
	Status   string
	TotalYen int64
}

// BookingQueries は予約の読み取りである。集約を組み立てない。
type BookingQueries interface {
	// ListByMember は会員の予約を開始の順に返す。
	// 予約が無ければ空のスライスを返す。
	ListByMember(
		ctx context.Context, member ids.MemberID,
	) ([]BookingView, error)
}

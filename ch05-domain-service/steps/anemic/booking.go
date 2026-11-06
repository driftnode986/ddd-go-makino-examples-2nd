// Package anemic は、ルールをすべてサービスに置いた予約の例である。
package anemic

import (
	"errors"
	"time"
)

// ErrNotCancellable はキャンセルできないことを表す。
var ErrNotCancellable = errors.New("not cancellable")

// #region booking

// Booking はデータだけを持つ。ルールは1つも無い。
type Booking struct {
	Status string
	Start  time.Time
}

// CancelService は会員のキャンセルを扱う。
type CancelService struct{}

// Cancel は確定した予約を、開始の24時間前までキャンセルする。
func (CancelService) Cancel(b *Booking, now time.Time) error {
	late := now.After(b.Start.Add(-24 * time.Hour))
	if b.Status != "confirmed" || late {
		return ErrNotCancellable
	}
	b.Status = "cancelled"
	return nil
}

// #endregion

// #region cleanup

// CleanupService は別の担当者が後から足した夜間の処理である。
type CleanupService struct{}

// CancelStale は結果を待ち続けている予約をキャンセルにする。
func (CleanupService) CancelStale(b *Booking) {
	if b.Status == "pending" {
		b.Status = "cancelled"
	}
}

// #endregion

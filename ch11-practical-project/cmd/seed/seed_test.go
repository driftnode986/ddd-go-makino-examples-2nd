package main

import (
	"errors"
	"testing"
	"time"

	identity "example.com/trainingbooking/internal/identity/domain"
)

func TestNewMember_RejectsInvalidEmail(t *testing.T) {
	_, err := newMember(memberSeed{
		"0196f5a0-0000-7000-8000-000000000009", "hanako", "花子",
	})
	if !errors.Is(err, identity.ErrInvalidEmail) {
		t.Fatalf("err = %v", err)
	}
	t.Logf("初期データの hanako: %v", err)
}

func TestNewLesson_StartsThirtyDaysLaterInJST(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	// 日本時間では6月2日に入っている
	now := time.Date(2030, time.June, 1, 20, 0, 0, 0, time.UTC)
	l, err := newLesson(seedLessons[0], now)
	if err != nil {
		t.Fatalf("newLesson: %v", err)
	}
	want := time.Date(2030, time.July, 2, 10, 0, 0, 0, jst)
	if got := l.State().Start; !got.Equal(want) {
		t.Errorf("start = %v, want %v", got, want)
	}
}

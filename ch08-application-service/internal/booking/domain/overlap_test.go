package domain_test

import (
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
)

func TestDetectOverlap(t *testing.T) {
	existing := []domain.TimeSlot{
		slot(t, at(1, 10), time.Hour),
		slot(t, at(1, 14), 2*time.Hour),
	}
	// #region cases
	cases := []struct {
		name    string
		slot    domain.TimeSlot
		wantErr error
	}{
		{"同じ時間帯", slot(t, at(1, 10), time.Hour),
			domain.ErrOverlap},
		{"2件目と一部が重なる", slot(t, at(1, 15), 2*time.Hour),
			domain.ErrOverlap},
		{"既存の時間帯を含む", slot(t, at(1, 9), 3*time.Hour),
			domain.ErrOverlap},
		// 半開区間なので、接するだけなら重ならない
		{"直前に接する", slot(t, at(1, 9), time.Hour), nil},
		{"直後に接する", slot(t, at(1, 11), time.Hour), nil},
		{"別の日", slot(t, at(2, 10), time.Hour), nil},
		{"同じ瞬間を UTC で渡す", slot(t, at(1, 10).UTC(), time.Hour),
			domain.ErrOverlap},
	}
	// #endregion
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := domain.DetectOverlap(c.slot, existing)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestDetectOverlap_NoExistingSlots(t *testing.T) {
	s := slot(t, at(1, 10), time.Hour)
	if err := domain.DetectOverlap(s, nil); err != nil {
		t.Fatalf("err = %v", err)
	}
}

// ゼロ値の時間帯は何とも重ならないので、確かめずに通してしまう。
func TestDetectOverlap_RejectsZeroSlot(t *testing.T) {
	existing := []domain.TimeSlot{slot(t, at(1, 10), time.Hour)}
	err := domain.DetectOverlap(domain.TimeSlot{}, existing)
	if !errors.Is(err, domain.ErrInvalidTimeSlot) {
		t.Fatalf("err = %v", err)
	}
}

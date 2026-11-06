package separateids_test

import (
	"testing"

	"example.com/trainingbooking/steps/separateids"
)

// 別々に書いた型でも、中身の構造が同じなら型変換は通る。
func TestConversionIsNotBlocked(t *testing.T) {
	m := separateids.NewMemberID()
	l := separateids.LessonID(m)
	if l.String() != m.String() {
		t.Fatalf("got %s, want %s", l, m)
	}
}

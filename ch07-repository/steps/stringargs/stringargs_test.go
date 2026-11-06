package stringargs_test

import (
	"testing"
	"time"

	"example.com/trainingbooking/steps/stringargs"
)

// #region swapped
func TestSwappedArguments(t *testing.T) {
	// 名前とメールアドレスが逆
	m := stringargs.RegisterMember("taro@example.com", "山田太郎")
	t.Logf("name=%q email=%q", m.Name, m.Email)

	// 会員 ID とレッスン ID が逆、料金は負、開始は朝4時
	memberID, lessonID := "M-001", "L-100"
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 4, 0, 0, 0, jst)
	b := stringargs.PlaceBooking(lessonID, memberID, -3000, start)
	t.Logf("member=%s lesson=%s", b.MemberID, b.LessonID)
	t.Logf("price=%d start=%s", b.PriceYen, b.Start.Format("15:04"))
}

// #endregion

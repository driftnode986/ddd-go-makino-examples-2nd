package embedmember_test

import (
	"testing"

	"example.com/trainingbooking/steps/embedmember"
)

// #region lost-update

func TestMemberChangeIsLost(t *testing.T) {
	store := embedmember.NewStore(embedmember.Member{
		ID: "M-1", Email: "old@example.com",
	}, "B-1")

	// 確定の処理が予約を読む
	b := store.Load("B-1", "M-1")

	// その間に会員がメールアドレスを変更する
	m := store.Member("M-1")
	m.Email = "new@example.com"
	store.SaveMember(m)

	// 確定の処理が、読んだときの会員ごと保存する
	b.Status = "confirmed"
	store.Save(b)

	t.Logf("保存後のメール: %s", store.Member("M-1").Email)
}

// #endregion

// #region keep-update

func TestMemberChangeIsKept(t *testing.T) {
	store := embedmember.NewStore(embedmember.Member{
		ID: "M-1", Email: "old@example.com",
	}, "B-1")

	b := embedmember.BookingByID{
		ID: "B-1", MemberID: "M-1", Status: "pending",
	}

	m := store.Member("M-1")
	m.Email = "new@example.com"
	store.SaveMember(m)

	b.Status = "confirmed"
	store.SaveByID(b)

	t.Logf("保存後のメール: %s", store.Member("M-1").Email)
	t.Logf("予約の会員 ID: %s", store.FindByID("B-1").MemberID)
}

// #endregion

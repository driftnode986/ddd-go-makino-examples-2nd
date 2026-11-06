package ids_test

import (
	"errors"
	"reflect"
	"testing"

	"example.com/trainingbooking/internal/ids"
)

func TestNewID_IsNotZeroAndUnique(t *testing.T) {
	a := ids.NewID[ids.BookingMarker]()
	b := ids.NewID[ids.BookingMarker]()
	if a.IsZero() {
		t.Fatal("NewID はゼロでない ID を返す")
	}
	if a.Equal(b) {
		t.Fatalf("2回の採番が同じ値を返した: %s", a)
	}
}

func TestParseID_RoundTrip(t *testing.T) {
	want := ids.NewID[ids.MemberMarker]()
	got, err := ids.ParseID[ids.MemberMarker](want.String())
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestParseID_RejectsInvalid(t *testing.T) {
	inputs := []string{
		"",
		"not-a-uuid",
		// ゼロの ID を文字列から作らせない
		"00000000-0000-0000-0000-000000000000",
	}
	for _, in := range inputs {
		_, err := ids.ParseID[ids.LessonMarker](in)
		if !errors.Is(err, ids.ErrInvalidID) {
			t.Errorf("ParseID(%q) = %v", in, err)
		}
	}
}

func TestZeroValues_AreDetected(t *testing.T) {
	var id ids.BookingID
	if !id.IsZero() {
		t.Fatal("var で宣言した ID は IsZero が true")
	}
}

// 次のコードはコンパイルエラーになる。
//
//	var m ids.MemberID
//	var l ids.LessonID = m
func TestID_TypesDoNotMix(t *testing.T) {
	member := reflect.TypeFor[ids.MemberID]()
	lesson := reflect.TypeFor[ids.LessonID]()
	booking := reflect.TypeFor[ids.BookingID]()
	if member == lesson || lesson == booking ||
		member == booking {
		t.Fatal("3つの ID は別の型である")
	}
}

// #region conversion

// 同じ構造の型どうしは型変換が通る。止められるのは代入の取り違えで、
// 明示の変換は止められない。
func TestID_ConversionIsNotBlocked(t *testing.T) {
	member := ids.NewID[ids.MemberMarker]()
	booking := ids.BookingID(member)
	if booking.String() != member.String() {
		t.Fatalf("got %s, want %s", booking, member)
	}
}

// #endregion

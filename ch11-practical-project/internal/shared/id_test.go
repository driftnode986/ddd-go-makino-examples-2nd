package shared_test

import (
	"errors"
	"reflect"
	"testing"

	"example.com/trainingbooking/internal/shared"
)

func TestNewID_IsNotZeroAndUnique(t *testing.T) {
	a := shared.NewID[shared.BookingMarker]()
	b := shared.NewID[shared.BookingMarker]()
	if a.IsZero() {
		t.Fatal("NewID はゼロでない ID を返す")
	}
	if a.Equal(b) {
		t.Fatalf("2回の採番が同じ値を返した: %s", a)
	}
}

func TestParseID_RoundTrip(t *testing.T) {
	want := shared.NewID[shared.MemberMarker]()
	got, err := shared.ParseID[shared.MemberMarker](want.String())
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
		_, err := shared.ParseID[shared.LessonMarker](in)
		if !errors.Is(err, shared.ErrInvalidID) {
			t.Errorf("ParseID(%q) = %v", in, err)
		}
	}
}

func TestZeroValues_AreDetected(t *testing.T) {
	var id shared.BookingID
	if !id.IsZero() {
		t.Fatal("var で宣言した ID は IsZero が true")
	}
}

// 次のコードはコンパイルエラーになる。
//
//	var m shared.MemberID
//	var l shared.LessonID = m
func TestID_TypesDoNotMix(t *testing.T) {
	member := reflect.TypeFor[shared.MemberID]()
	lesson := reflect.TypeFor[shared.LessonID]()
	booking := reflect.TypeFor[shared.BookingID]()
	if member == lesson || lesson == booking ||
		member == booking {
		t.Fatal("3つの ID は別の型である")
	}
}

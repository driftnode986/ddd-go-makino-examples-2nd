package dto_test

import (
	"encoding/json"
	"testing"

	"example.com/trainingbooking/dto"
)

func encode(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(b)
}

// 非公開フィールドだけの struct を Encode すると {} になる。
func TestTicket_EncodesToEmptyObject(t *testing.T) {
	if got := encode(t, dto.NewTicket("A1")); got != "{}" {
		t.Fatalf("got %s", got)
	}
}

func TestToResponse_EncodesFields(t *testing.T) {
	got := encode(t, dto.ToResponse(dto.NewTicket("A1")))
	want := `{"code":"A1","used":false}`
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

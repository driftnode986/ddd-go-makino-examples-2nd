package contract_test

import (
	"encoding/json"
	"testing"
	"time"

	"example.com/trainingbooking/internal/catalog/contract"
	"github.com/google/go-cmp/cmp"
)

var occurredAt = time.Date(
	2030, time.May, 1, 0, 0, 0, 0, time.UTC)

// 統合イベントの JSON は outbox に残り、別のコンテキストが読む。
// 項目名を変えると保存済みのイベントが読めなくなるので、文字列で
// 固定する。
func TestEvent_JSONCompatibility(t *testing.T) {
	cases := []struct {
		name  string
		event any
		json  string
		read  func([]byte) (any, error)
	}{
		{
			name: contract.EventSeatReserved,
			event: contract.SeatReserved{
				LessonID:   "l1",
				BookingID:  "b1",
				OccurredAt: occurredAt,
			},
			json: `{"lesson_id":"l1","booking_id":"b1",` +
				`"occurred_at":"2030-05-01T00:00:00Z"}`,
			read: read[contract.SeatReserved],
		},
		{
			name: contract.EventSeatRejected,
			event: contract.SeatRejected{
				LessonID:   "l1",
				BookingID:  "b1",
				Reason:     contract.ReasonLessonFull,
				OccurredAt: occurredAt,
			},
			json: `{"lesson_id":"l1","booking_id":"b1",` +
				`"reason":"lesson_full",` +
				`"occurred_at":"2030-05-01T00:00:00Z"}`,
			read: read[contract.SeatRejected],
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := json.Marshal(c.event)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if diff := cmp.Diff(c.json, string(got)); diff != "" {
				t.Errorf("書く形 (-want +got):\n%s", diff)
			}
			newer := c.json[:len(c.json)-1] + `,"added_later":1}`
			for _, body := range []string{c.json, newer} {
				decoded, err := c.read([]byte(body))
				if err != nil {
					t.Fatalf("read: %v", err)
				}
				if diff := cmp.Diff(c.event, decoded); diff != "" {
					t.Errorf("読む形 (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func read[T any](body []byte) (any, error) {
	var v T
	err := json.Unmarshal(body, &v)
	return v, err
}

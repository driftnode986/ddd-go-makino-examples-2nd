// Package pitfalls_test は、第3章で失敗や落とし穴を動かすテストである。
package pitfalls_test

import (
	"encoding/json"
	"math"
	"net/mail"
	"testing"
	"time"

	booking "example.com/trainingbooking/internal/booking/domain"
	identity "example.com/trainingbooking/internal/identity/domain"
)

func TestParseAddressAcceptsDisplayName(t *testing.T) {
	// #region parse-address
	addr, _ := mail.ParseAddress("Taro <taro@example.com>")
	t.Logf("name=%q address=%q", addr.Name, addr.Address)
	// #endregion
}

func TestZeroEmailCompiles(t *testing.T) {
	// #region zero-email
	var e identity.Email
	t.Logf("String=%q IsZero=%v", e.String(), e.IsZero())
	// #endregion
	if !e.IsZero() {
		t.Fatal("var で宣言した Email は IsZero が true")
	}
}

func TestFloatPrice(t *testing.T) {
	// #region float
	price := 3000.0
	withTax := price * 1.1
	t.Logf("withTax=%v ceil=%v", withTax, math.Ceil(withTax))
	// #endregion
	if math.Ceil(withTax) != 3301 {
		t.Fatal("誤差で1円増えるはず")
	}
}

func TestIntegerPrice(t *testing.T) {
	// #region integer
	price := int64(3000)
	withTax := (price*110 + 99) / 100 // 切り上げ
	t.Logf("withTax=%d", withTax)
	// #endregion
	if withTax != 3300 {
		t.Fatalf("got %d", withTax)
	}
}

func TestInt64Overflow(t *testing.T) {
	// #region overflow
	total := int64(math.MaxInt64)
	total++
	t.Logf("total=%d", total)
	// #endregion
	if total >= 0 {
		t.Fatal("負の値になるはず")
	}
}

func TestOperatorOnTimeSlot(t *testing.T) {
	// #region time-equal
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	a, _ := booking.NewTimeSlot(start, time.Hour, jst)
	b, _ := booking.NewTimeSlot(start.UTC(), time.Hour, jst)
	t.Logf("a == b: %v", a == b)
	t.Logf("a.Equal(b): %v", a.Equal(b))
	// #endregion
	if a == b || !a.Equal(b) {
		t.Fatal("== は false、Equal は true のはず")
	}
}

func TestMarshalEmail(t *testing.T) {
	// #region json
	e, _ := identity.NewEmail("taro@example.com")
	b, _ := json.Marshal(e)
	t.Logf("%s", b)
	// #endregion
	if string(b) != "{}" {
		t.Fatalf("非公開フィールドは JSON に出ないはず: %s", b)
	}
}

// Start は渡された形のまま返すので、時を読むときは In を通す。
func TestStartKeepsTheGivenZone(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	s, err := booking.NewTimeSlot(start.UTC(), time.Hour, jst)
	if err != nil {
		t.Fatal(err)
	}
	if s.Start().Hour() != 1 || s.Start().In(jst).Hour() != 10 {
		t.Fatalf("Start = %v", s.Start())
	}
}

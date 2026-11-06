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

// #region parse-address
func TestParseAddressAcceptsDisplayName(t *testing.T) {
	addr, _ := mail.ParseAddress("Taro <taro@example.com>")
	t.Logf("name=%q address=%q", addr.Name, addr.Address)
}

// #endregion

// #region zero-email
func TestZeroEmailCompiles(t *testing.T) {
	var e identity.Email
	t.Logf("String=%q IsZero=%v", e.String(), e.IsZero())
	if !e.IsZero() {
		t.Fatal("var で宣言した Email は IsZero が true")
	}
}

// #endregion

// #region float
func TestFloatPrice(t *testing.T) {
	price := 3000.0
	withTax := price * 1.1
	t.Logf("withTax=%v ceil=%v", withTax, math.Ceil(withTax))
	if math.Ceil(withTax) != 3301 {
		t.Fatal("誤差で1円増えるはず")
	}
}

// #endregion

// #region overflow
func TestInt64Overflow(t *testing.T) {
	total := int64(math.MaxInt64)
	one := int64(1)
	total += one
	t.Logf("total=%d", total)
	if total >= 0 {
		t.Fatal("負の値になるはず")
	}
}

// #endregion

// #region time-equal
func TestOperatorOnTimeSlot(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	a, _ := booking.NewTimeSlot(start, time.Hour, jst)
	b, _ := booking.NewTimeSlot(start.UTC(), time.Hour, jst)
	t.Logf("a == b: %v", a == b)
	t.Logf("a.Equal(b): %v", a.Equal(b))
	if a == b || !a.Equal(b) {
		t.Fatal("== は false、Equal は true のはず")
	}
}

// #endregion

// #region json
func TestMarshalEmail(t *testing.T) {
	e, _ := identity.NewEmail("taro@example.com")
	b, _ := json.Marshal(e)
	t.Logf("%s", b)
	if string(b) != "{}" {
		t.Fatalf("非公開フィールドは JSON に出ないはず: %s", b)
	}
}

// #endregion

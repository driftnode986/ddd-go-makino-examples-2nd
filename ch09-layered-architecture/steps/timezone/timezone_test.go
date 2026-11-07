// Package timezone は同じ瞬間が JSON で別の文字列になることを見せる。
package timezone_test

import (
	"encoding/json"
	"testing"
	"time"
)

// #region encode

func TestSameInstantEncodesDifferently(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)

	inJST, err := json.Marshal(start)
	if err != nil {
		t.Fatal(err)
	}
	inUTC, err := json.Marshal(start.UTC())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("JST の値:         %s", inJST)
	t.Logf("UTC に変換した値: %s", inUTC)
	if string(inJST) == string(inUTC) {
		t.Fatal("同じ文字列になった")
	}
}

// #endregion

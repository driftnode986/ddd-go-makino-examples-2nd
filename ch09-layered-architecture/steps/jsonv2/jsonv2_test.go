// Package jsonv2 は encoding/json と encoding/json/v2 の既定の違いを見せる。
package jsonv2_test

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"testing"
)

type request struct {
	MemberID string `json:"member_id"`
}

// #region duplicate

func TestDuplicateName(t *testing.T) {
	body := []byte(`{"member_id":"a","member_id":"b"}`)

	var v1 request
	err := json.Unmarshal(body, &v1)
	t.Logf("encoding/json: %q, err = %v", v1.MemberID, err)
	if err != nil {
		t.Fatal(err)
	}

	var v2 request
	err = jsonv2.Unmarshal(body, &v2)
	t.Logf("encoding/json/v2: err = %v", err)
	if err == nil {
		t.Fatal("v2 が受け付けた")
	}
}

// #endregion

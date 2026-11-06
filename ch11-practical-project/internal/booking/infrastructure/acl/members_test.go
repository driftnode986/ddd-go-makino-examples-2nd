package acl_test

import (
	"context"
	"errors"
	"testing"

	"example.com/trainingbooking/internal/booking/infrastructure/acl"
	"example.com/trainingbooking/internal/ids"
)

type stubMemberReader struct {
	exists bool
	err    error
}

func (s stubMemberReader) MemberExists(
	context.Context, ids.MemberID,
) (bool, error) {
	return s.exists, s.err
}

func TestMemberDirectory_PassesTheAnswer(t *testing.T) {
	id := ids.NewID[ids.MemberMarker]()
	boom := errors.New("boom")
	cases := []struct {
		reader stubMemberReader
		want   bool
	}{
		{stubMemberReader{exists: true}, true},
		{stubMemberReader{}, false},
		{stubMemberReader{err: boom}, false},
	}
	for _, c := range cases {
		d := acl.NewMemberDirectory(c.reader)
		got, err := d.Exists(t.Context(), id)
		if got != c.want || !errors.Is(err, c.reader.err) {
			t.Errorf("%+v: %v, %v", c.reader, got, err)
		}
	}
}

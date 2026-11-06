package acl_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/infrastructure/acl"
	"example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/ids"
)

var jst = time.FixedZone("JST", 9*60*60)

type stubReader struct {
	lesson contract.Lesson
	err    error
}

func (s stubReader) FindLesson(
	context.Context, ids.LessonID,
) (contract.Lesson, error) {
	return s.lesson, s.err
}

func TestLessonCatalog_TranslatesContractLesson(t *testing.T) {
	start := time.Date(2030, time.June, 1, 1, 0, 0, 0, time.UTC)
	lesson := contract.Lesson{
		Start:    start,
		Duration: time.Hour,
		FeeYen:   3000,
		Options: []contract.LessonOption{
			{Name: "シューズ", UnitPriceYen: 300},
		},
	}
	catalog := acl.NewLessonCatalog(stubReader{lesson: lesson}, jst)
	got, err := catalog.Terms(
		t.Context(), ids.NewID[ids.LessonMarker]())
	if err != nil {
		t.Fatalf("Terms: %v", err)
	}
	if !got.Start.Equal(start) || got.Duration != time.Hour ||
		got.Fee.Amount() != 3000 || got.Location != jst {
		t.Errorf("got = %+v", got)
	}
	if len(got.Options) != 1 || got.Options[0].Name != "シューズ" ||
		got.Options[0].UnitPrice.Amount() != 300 {
		t.Errorf("Options = %+v", got.Options)
	}
}

func TestLessonCatalog_TranslatesErrors(t *testing.T) {
	missing := acl.NewLessonCatalog(
		stubReader{err: contract.ErrLessonNotFound}, jst)
	_, err := missing.Terms(
		t.Context(), ids.NewID[ids.LessonMarker]())
	if !errors.Is(err, application.ErrLessonNotFound) {
		t.Errorf("無いレッスン: %v", err)
	}

	boom := errors.New("boom")
	failing := acl.NewLessonCatalog(stubReader{err: boom}, jst)
	_, err = failing.Terms(
		t.Context(), ids.NewID[ids.LessonMarker]())
	if !errors.Is(err, boom) ||
		errors.Is(err, application.ErrLessonNotFound) {
		t.Errorf("その他の失敗: %v", err)
	}
}

func TestLessonCatalog_RejectsNegativePrices(t *testing.T) {
	cases := []contract.Lesson{
		{FeeYen: -1},
		{FeeYen: 1, Options: []contract.LessonOption{
			{Name: "x", UnitPriceYen: -1},
		}},
	}
	for _, lesson := range cases {
		c := acl.NewLessonCatalog(stubReader{lesson: lesson}, jst)
		_, err := c.Terms(t.Context(), ids.NewID[ids.LessonMarker]())
		if err == nil {
			t.Errorf("lesson = %+v", lesson)
		}
	}
}

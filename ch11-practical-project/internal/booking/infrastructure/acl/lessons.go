// Package acl は他のコンテキストの contract を Booking の言葉に
// 変える腐敗防止層である。
package acl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/ids"
)

// LessonCatalog は Catalog のレッスンを LessonTerms に変える。
type LessonCatalog struct {
	reader   contract.LessonReader
	location *time.Location
}

// NewLessonCatalog は LessonCatalog を返す。location は営業時間の
// 判定に使うタイムゾーンで、Booking の側の設定である。
func NewLessonCatalog(
	reader contract.LessonReader, location *time.Location,
) *LessonCatalog {
	return &LessonCatalog{reader: reader, location: location}
}

// #region terms

// Terms はレッスンの条件を返す。
func (c *LessonCatalog) Terms(
	ctx context.Context, id ids.LessonID,
) (application.LessonTerms, error) {
	lesson, err := c.reader.FindLesson(ctx, id)
	if errors.Is(err, contract.ErrLessonNotFound) {
		return application.LessonTerms{}, fmt.Errorf(
			"%w: %s", application.ErrLessonNotFound, id)
	}
	if err != nil {
		return application.LessonTerms{}, err
	}
	fee, err := domain.NewMoney(lesson.FeeYen)
	if err != nil {
		return application.LessonTerms{}, err
	}
	options, err := toOffered(lesson.Options)
	if err != nil {
		return application.LessonTerms{}, err
	}
	return application.LessonTerms{
		Start:    lesson.Start,
		Duration: lesson.Duration,
		Fee:      fee,
		Location: c.location,
		Options:  options,
	}, nil
}

// #endregion

func toOffered(
	in []contract.LessonOption,
) ([]application.OfferedOption, error) {
	options := make([]application.OfferedOption, len(in))
	for i, o := range in {
		price, err := domain.NewMoney(o.UnitPriceYen)
		if err != nil {
			return nil, err
		}
		options[i] = application.OfferedOption{
			Name:      o.Name,
			UnitPrice: price,
		}
	}
	return options, nil
}

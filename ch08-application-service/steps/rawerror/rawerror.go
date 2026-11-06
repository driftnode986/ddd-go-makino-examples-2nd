// Package rawerror はユースケースのエラーをそのまま HTTP にする例である。
package rawerror

import (
	"errors"
	"net/http"

	"example.com/trainingbooking/internal/booking/domain"
)

// #region respond

// Respond は domain のエラーを知っていて、残りは本文に文字列を出す。
func Respond(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "予約が見つかりません"
	case errors.Is(err, domain.ErrVersionConflict),
		errors.Is(err, domain.ErrOverlap):
		return http.StatusConflict, err.Error()
	}
	return http.StatusInternalServerError, err.Error()
}

// #endregion

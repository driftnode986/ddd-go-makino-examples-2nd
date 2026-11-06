package application

import (
	"context"
	"errors"

	"example.com/trainingbooking/internal/catalog/domain"
)

// maxAttempts は版の競合を読み直す回数の上限である。上限を超えた
// ら競合のまま返し、配信処理の再配信に任せる。
const maxAttempts = 5

// retryOnConflict は版の競合のとき attempt をやり直す。attempt は
// 毎回 Tx を開き直して読み直す。失敗した集約は持ち越さない。
func retryOnConflict(
	ctx context.Context, attempt func() error,
) error {
	var err error
	for range maxAttempts {
		err = attempt()
		if !errors.Is(err, domain.ErrVersionConflict) {
			return err
		}
		if ctx.Err() != nil {
			return err
		}
	}
	return err
}

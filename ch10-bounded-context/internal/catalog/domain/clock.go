package domain

import "time"

// Clock は現在時刻を返す。
type Clock interface {
	Now() time.Time
}

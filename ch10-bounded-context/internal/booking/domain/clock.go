package domain

import "time"

// Clock は現在時刻を提供する。ドメインは time.Now を呼ばない。
type Clock interface {
	Now() time.Time
}

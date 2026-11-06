// わざとコンパイルできないコード。同じ package 名を別名なしで2つ import する。
package main

// #region imports

import (
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/identity/domain"
)

// #endregion

func main() {
	_ = domain.ErrOverlap
}

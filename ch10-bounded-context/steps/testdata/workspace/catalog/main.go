// Command catalog は別の module の内部を import しようとする例である。
package main

import (
	"fmt"

	"example.com/workspace/booking/internal/domain"
)

func main() { fmt.Println(domain.Booking{}) }

// Command catalog は Booking の内部を import しようとする例である。
package main

// #region import
import (
	"fmt"

	"example.com/trainingbooking/steps/testdata/nested/booking/internal/domain"
)

// #endregion

func main() { fmt.Println(domain.Booking{}) }

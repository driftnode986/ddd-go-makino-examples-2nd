// わざとコンパイルできないコードである。
package main

// #region blank-field
type ID[T any] struct {
	_     [0]T
	value string
}

// #endregion

type (
	memberMarker  struct{}
	bookingMarker struct{}
)

func main() {
	member := ID[memberMarker]{value: "m"}
	_ = ID[bookingMarker](member)
}

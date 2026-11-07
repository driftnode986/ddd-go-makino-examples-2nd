// わざとコンパイルできないコード。Card を値のまま渡す。
package main

import "example.com/trainingbooking/receiver"

// #region call

type charger interface {
	Charge(p receiver.Points)
}

func topUp(c charger) { c.Charge(receiver.NewPoints(100)) }

func main() {
	topUp(receiver.Card{})
}

// #endregion

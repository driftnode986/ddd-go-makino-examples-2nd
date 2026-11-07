// わざとコンパイルできないコード。型引数のない Get を要求する。
package main

import "example.com/trainingbooking/generics"

// #region call

type intGetter interface {
	Get(key string) (int, error)
}

func main() {
	var _ intGetter = generics.NewCache()
}

// #endregion

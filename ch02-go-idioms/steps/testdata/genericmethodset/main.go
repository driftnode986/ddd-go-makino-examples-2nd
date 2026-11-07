// わざとコンパイルできないコード。型引数のない Find を要求する。
package main

import "example.com/trainingbooking/generics"

// #region call

type intFinder interface {
	Find(key string) (int, error)
}

func main() {
	var _ intFinder = generics.NewStore()
}

// #endregion

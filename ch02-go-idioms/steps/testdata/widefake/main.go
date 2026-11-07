// わざとコンパイルできないコード。Send だけの型を渡す。
package main

import (
	"errors"

	"example.com/trainingbooking/steps/wideinterface"
)

// #region call

type failing struct{ err error }

func (f failing) Send(string, string) error { return f.err }

func main() {
	wideinterface.NewReminder(failing{errors.New("boom")})
}

// #endregion

// Package zerovalue は、非公開フィールドで守れることと守れない
// ことを示す。ゼロ値は外から作れるので、受け取る側で止める。
package zerovalue

import "errors"

// ErrInvalid は不正な値を表す。
var ErrInvalid = errors.New("invalid value")

// #region public

// PublicPoints はフィールドが公開なので、負の値も書き込める。
type PublicPoints struct {
	Balance int
}

// #endregion

// #region nickname

// Nickname は非公開フィールドで値を包む。
type Nickname struct {
	value string
}

// NewNickname は空でない名前から Nickname を作る。
func NewNickname(s string) (Nickname, error) {
	if s == "" {
		return Nickname{}, ErrInvalid
	}
	return Nickname{value: s}, nil
}

// IsZero は NewNickname を通していないとき true を返す。
func (n Nickname) IsZero() bool { return n.value == "" }

// #endregion

// #region profile

// Profile は Nickname を受け取って保持する側である。
type Profile struct {
	nickname Nickname
}

// NewProfile はゼロ値の Nickname を拒否する。
func NewProfile(n Nickname) (*Profile, error) {
	if n.IsZero() {
		return nil, ErrInvalid
	}
	return &Profile{nickname: n}, nil
}

// #endregion

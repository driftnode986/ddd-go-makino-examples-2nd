// Package generics は Go 1.27 のジェネリックメソッドと、
// インターフェースで差し替えられないことを示す。
package generics

import (
	"encoding/json"
	"fmt"
)

// Store は型を問わず値を JSON で保持する。
type Store struct {
	items map[string][]byte
}

// NewStore は Store を作る。
func NewStore() *Store { return &Store{items: map[string][]byte{}} }

// Find は保持している値を T に読み替える。s の状態を読むので
// メソッドにする。型引数を持てるのは Go 1.27 からである。
func (s *Store) Find[T any](key string) (T, error) {
	var v T
	raw, ok := s.items[key]
	if !ok {
		return v, fmt.Errorf("no item: %s", key)
	}
	err := json.Unmarshal(raw, &v)
	return v, err
}

// Put は値を JSON にして保持する。
func (s *Store) Put(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	s.items[key] = raw
	return nil
}

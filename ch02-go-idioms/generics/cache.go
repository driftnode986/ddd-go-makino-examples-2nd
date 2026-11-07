// Package generics は Go 1.27 のジェネリックメソッドと、
// インターフェースで差し替えられないことを示す。
package generics

import (
	"encoding/json"
	"fmt"
)

// #region cache

// Cache は値を JSON で保持する。
type Cache struct {
	items map[string][]byte
}

// NewCache は Cache を作る。
func NewCache() *Cache { return &Cache{items: map[string][]byte{}} }

// Get は保持している値を T に読み替える。c の状態を読むので
// メソッドにする。型引数を持てるのは Go 1.27 からである。
func (c *Cache) Get[T any](key string) (T, error) {
	var v T
	raw, ok := c.items[key]
	if !ok {
		return v, fmt.Errorf("no item: %s", key)
	}
	err := json.Unmarshal(raw, &v)
	return v, err
}

// #endregion

// Put は値を JSON にして保持する。
func (c *Cache) Put(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	c.items[key] = raw
	return nil
}

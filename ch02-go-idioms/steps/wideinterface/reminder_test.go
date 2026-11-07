package wideinterface_test

import (
	"testing"

	"example.com/trainingbooking/steps/wideinterface"
)

// インターフェースを埋め込むとメソッドを書かずに渡せる。
type embedded struct {
	wideinterface.Notifier
}

func (embedded) Send(string, string) error { return nil }

// コンパイルは通るが、書かなかったメソッドを呼ぶと panic する。
func TestEmbedded_PanicsOnUnwrittenMethod(t *testing.T) {
	var _ = wideinterface.NewReminder(embedded{})
	defer func() {
		if recover() == nil {
			t.Fatal("panic しなかった")
		}
	}()
	embedded{}.Sent()
}

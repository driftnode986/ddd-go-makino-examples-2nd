package steps_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// testdata のコードがコンパイルできないことを確かめる。
// 本文の「コンパイルエラーになる」の裏づけである。
func TestCompileErrors(t *testing.T) {
	out, err := exec.Command("go", "build", "-o", os.DevNull,
		"./testdata/setstatus").CombinedOutput()
	if err == nil {
		t.Fatal("コンパイルが通った")
	}
	if !strings.Contains(string(out), "status") {
		t.Errorf("status の行が出力に無い:\n%s", out)
	}
}

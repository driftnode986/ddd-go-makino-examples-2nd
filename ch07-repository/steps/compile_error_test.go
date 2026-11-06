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
	cases := map[string][]string{
		"swapemail": {
			"as string value in argument to registerMember",
			"as domain.Email value in argument to registerMember",
		},
		"emailliteral": {
			"cannot convert",
			"value",
			"as domain.Email value in variable declaration",
		},
		"swapids": {
			"as ids.MemberID value in argument to placeBooking",
			"as ids.LessonID value in argument to placeBooking",
		},
		"blankfield": {"cannot convert member"},
	}
	for name, wants := range cases {
		t.Run(name, func(t *testing.T) {
			out, err := exec.Command("go", "build", "-o", os.DevNull,
				"./testdata/"+name).CombinedOutput()
			if err == nil {
				t.Fatal("コンパイルが通った")
			}
			for _, want := range wants {
				if !strings.Contains(string(out), want) {
					t.Errorf("%q が出力に無い:\n%s", want, out)
				}
			}
		})
	}
}

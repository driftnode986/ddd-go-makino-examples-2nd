#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# テストの結果は、t.Logf の中身（と指定があれば --- PASS の行）だけを残す。
# ビルドの結果は、見出しの行とディレクトリを除く。
# 紙面に入らない行は 69 桁で折り返す。
shape() { expand -t 4 | fold -s -w 69 | sed 's/ *$//'; }
logs() { # 名前 パターン パッケージ
	go test -count=1 -v -run "$2" "$3" 2>&1 | grep -E '_test\.go:[0-9]+:' | sed -E 's/^ +[a-z_]+_test\.go:[0-9]+: //' | shape >"out/$1.txt"
}
logs_pass() { # 名前 パターン パッケージ
	go test -count=1 -v -run "$2" "$3" 2>&1 | grep -E '_test\.go:[0-9]+:|^--- ' | sed -E 's/^ +[a-z_]+_test\.go:[0-9]+: //' | shape >"out/$1.txt"
}
build() { # 名前 ディレクトリ
	go build "./steps/testdata/$2" 2>&1 | grep -v '^#' | sed "s|^steps/testdata/$2/||" | shape >"out/$1.txt"
}
logs_pass swapped TestSwappedArguments ./steps/stringargs/
build swapemail swapemail
logs_pass stringemail TestConstructorIsBypassed ./steps/stringemail/
build emailliteral emailliteral
logs parse-address TestParseAddress ./steps/pitfalls/
logs zero-email TestZeroEmailCompiles ./steps/pitfalls/
logs float TestFloatPrice ./steps/pitfalls/
logs integer TestIntegerPrice ./steps/pitfalls/
logs overflow TestInt64Overflow ./steps/pitfalls/
logs zones TestSameInstantDifferentResult ./steps/naiveslot/
logs time-equal TestOperatorOnTimeSlot ./steps/pitfalls/
build swapids swapids
logs json TestMarshalEmail ./steps/pitfalls/

#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# テストの結果は、t.Logf の中身だけを残す。
# ビルドの結果は、見出しの行とディレクトリを除く。
# 紙面に入らない行は 69 桁で折り返す。
shape() { expand -t 4 | fold -s -w 69 | sed 's/ *$//'; }
logs() { # 名前 パターン パッケージ
	go test -count=1 -v -run "$2" "$3" 2>&1 | grep -E '_test\.go:[0-9]+:' | sed -E 's/^ +[a-z_]+_test\.go:[0-9]+: //' | shape >"out/$1.txt"
}
build() { # 名前 ディレクトリ
	go build "./steps/testdata/$2" 2>&1 | grep -v '^#' | sed "s|^steps/testdata/$2/||" | shape >"out/$1.txt"
}
build widefake widefake
go list -f '{{.ImportPath}}: {{.Imports}}' ./smallinterface/... | sed 's|^example.com/trainingbooking/smallinterface/||' | shape >out/imports.txt
logs public TestPublicField_AcceptsNegative ./zerovalue/
logs same-package TestSamePackage_CanWriteUnexported ./zerovalue/
logs zero TestZeroValue_BufferAndNickname ./zerovalue/
logs compare TestTake_CompareSentinel ./domainerror/
logs typed TestTake_TypedError ./domainerror/
build valuecard valuecard
logs bypass TestBypass ./steps/setterstock/service/
build genericmethodset genericmethodset
# インターフェースのメソッドに型引数を書くと構文エラーになる。
# gofmt が読めるように .go.txt で置き、ここで一時的に .go にして確かめる。
tmp=$(mktemp -d)
cp steps/testdata/genericinterface/main.go.txt "$tmp/main.go"
(cd "$tmp" && go build main.go 2>&1) | grep -v '^#' | grep 'type parameters' | sed -E 's|^[^ ]*main.go:[0-9]+:[0-9]+: ||' | shape >out/genericinterface.txt
rm -rf "$tmp"

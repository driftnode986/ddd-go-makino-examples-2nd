#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# 紙面に入らない行は 69 桁で折り返す。
run() { # 名前 コマンド...
	name=$1
	shift
	"$@" 2>&1 | sed -E 's/^ok[[:space:]]+/ok  /' | expand -t 4 | fold -s -w 69 | sed 's/ *$//' >"out/$name.txt"
}
run swapped go test -count=1 -v -run TestSwappedArguments ./steps/stringargs/
run swapemail go build ./steps/testdata/swapemail
run stringemail go test -count=1 -v -run TestConstructorIsBypassed ./steps/stringemail/
run emailliteral go build ./steps/testdata/emailliteral
run parse-address go test -count=1 -v -run TestParseAddress ./steps/pitfalls/
run normalizes-domain go test -count=1 -v -run TestEmail_NormalizesDomain ./internal/identity/domain/
run zero-email go test -count=1 -v -run TestZeroEmailCompiles ./steps/pitfalls/
run zero-stop go test -count=1 -v -run TestZeroEmailIsRejected ./steps/registermember/
run float go test -count=1 -v -run TestFloatPrice ./steps/pitfalls/
run overflow go test -count=1 -v -run TestInt64Overflow ./steps/pitfalls/
run zones go test -count=1 -v -run TestSameInstantDifferentResult ./steps/naiveslot/
run zones-fixed go test -count=1 -v -run TestTimeSlot_SameInstantAcrossZones ./internal/booking/domain/
run time-equal go test -count=1 -v -run TestOperatorOnTimeSlot ./steps/pitfalls/
run business-hours go test -count=1 -run TestTimeSlot_BusinessHours ./internal/booking/domain/
run swapids go build ./steps/testdata/swapids
run conversion go test -count=1 -v -run TestID_ConversionIsNotBlocked ./internal/ids/
run blankfield go build ./steps/testdata/blankfield
run json go test -count=1 -v -run TestMarshalEmail ./steps/pitfalls/
run all go test -count=1 ./...

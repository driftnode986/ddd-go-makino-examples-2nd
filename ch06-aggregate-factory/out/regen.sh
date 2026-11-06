#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# テストの結果は、t.Logf の中身だけを残す。
# 紙面に入らない行は 69 桁で折り返す。
shape() { expand -t 4 | fold -s -w 69 | sed "s/ *$//"; }
logs() { # 名前 パターン パッケージ
	go test -count=1 -v -run "$2" "$3" 2>&1 | grep -E "_test\.go:[0-9]+:" | sed -E "s/^ +[a-z_]+_test\.go:[0-9]+: //" | shape >"out/$1.txt"
}
logs negative-total TestTotalGoesNegative ./steps/separateoptions/
logs six-options TestLimitIsBypassed ./steps/separateoptions/
logs rewrite TestRewriteThroughGetter ./steps/leakyoptions/
logs shallow TestCloneIsShallow ./steps/leakyoptions/
logs apply-coupon TestApplyCoupon ./steps/pitfalls/
logs discount-order TestDiscountAmountIsFixed ./steps/pitfalls/
logs lost-update TestMemberChangeIsLost ./steps/embedmember/
logs keep-update TestMemberChangeIsKept ./steps/embedmember/
logs one-tx TestOneTxFailsWithSeatsLeft ./steps/twoaggregates/
logs separate-tx TestSeparateTxKeepsBothBookings ./steps/twoaggregates/
logs order TestOrderChangesResult ./steps/withoptions/
logs past-start TestNewBookingCannotRestorePast ./steps/pitfalls/
logs broken-row TestBrokenRowIsAccepted ./steps/rawrebuild/
logs broken-state TestReconstructRejectsBrokenRow ./steps/pitfalls/
logs bypass TestReconstructSkipsNewBookingChecks ./steps/pitfalls/
logs cmp-equal TestCmpUsesEqual ./steps/pitfalls/
logs cmp-panic TestCmpPanicsOnUnexported ./steps/pitfalls/
logs stale-copies TestStaleCopiesBothAddOption ./steps/pitfalls/
# go-cmp の差分は複数行で、版により空白の文字を変えるので普通の空白にそろえる。
go test -count=1 -v -run TestCmpShowsDifferenceInState ./steps/pitfalls/ 2>&1 |
	sed -n '/_test\.go:/,/^--- PASS/p' | grep -v '^--- PASS' |
	sed -E 's/^ +[a-z_]+_test\.go:[0-9]+: //; s/^ {8}//' |
	perl -CSD -pe 's/\x{00A0}/ /g' | sed '/^ *$/d' | expand -t 4 >out/cmp-state.txt
# 別のコンテキストの domain を import すると、golangci-lint が止める。
tmp=internal/booking/domain/zz_import.go
cat >"$tmp" <<'GO'
package domain

import _ "example.com/trainingbooking/internal/identity/domain"
GO
golangci-lint run ./internal/booking/domain/ 2>&1 | grep -E "\(depguard\)" | shape >out/depguard.txt
rm -f "$tmp"

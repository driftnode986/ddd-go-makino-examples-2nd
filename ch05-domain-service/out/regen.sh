#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# テストの結果は、t.Logf の中身だけを残す。
# 紙面に入らない行は 69 桁で折り返す。
shape() { expand -t 4 | fold -s -w 69 | sed "s/ *$//"; }
logs() { # 名前 パターン パッケージ
	go test -count=1 -v -run "$2" "$3" 2>&1 | grep -E "_test\.go:[0-9]+:" | sed -E "s/^ +[a-z_]+_test\.go:[0-9]+: //" | shape >"out/$1.txt"
}
logs same-slot TestSameMemberSameSlot ./steps/pitfalls/
logs zero-slot TestZeroSlot ./steps/pitfalls/
logs adjacent TestAdjacentSlots ./steps/closedslot/
logs check-then-save TestCheckThenSave ./steps/pitfalls/
logs float TestFloatDiscount ./steps/pitfalls/
logs integer TestIntegerDiscount ./steps/pitfalls/
logs differs TestRoundingDiffers ./steps/discountplace/
logs client-percent TestPercentFromClient ./steps/pitfalls/
logs shared TestSharedAcrossMembers ./steps/statefuldetector/
logs bypass TestRuleIsBypassed ./steps/anemic/
go list -f '{{join .Imports "\n"}}' ./internal/booking/domain >out/imports.txt

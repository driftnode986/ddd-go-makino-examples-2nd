#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# Docker が要る。testcontainers-go が起動したコンテナは、終わると止まる。
# テストの結果は、t.Logf の中身だけを残す。
# 紙面に入らない行は 69 桁で折り返す。
shape() { expand -t 4 | fold -s -w 69 | sed "s/ *$//"; }
logs() { # 名前 パターン パッケージ
	go test -count=1 -v -tags=integration -run "$2" "$3" 2>&1 | grep -E "_test\.go:[0-9]+:" | sed -E "s/^ +[a-z_]+_test\.go:[0-9]+: //" | shape >"out/$1.txt"
}
logs two-paths TestTwoPathsDiffer ./steps/twopaths/
logs separate-tx TestSeparateTxLeavesBookingWithoutEvent ./steps/pitfalls/
logs retry-same TestRetryWithSameBookingLosesEvent ./steps/pitfalls/
logs plain-log TestStandardHandlerIgnoresContext ./steps/pitfalls/
logs outer-read TestOuterContextReadsOutsideTheTx ./steps/pitfalls/
logs bound-read TestBoundStoresReadInsideTheTx ./steps/pitfalls/
logs reread TestConflictThenRereadJudgesAgain ./steps/pitfalls/
logs n-plus-one TestListByFindByIDRunsManyQueries ./steps/pitfalls/
logs cause-message TestCauseStaysOutOfMessage ./steps/pitfalls/
logs raw-error TestRawErrorsReachTheClient ./steps/rawerror/
# application が pgx を import すると、golangci-lint が止める。
tmp=internal/booking/application/zz_tx.go
cat >"$tmp" <<'GO'
package application

import _ "github.com/jackc/pgx/v5"
GO
golangci-lint run --allow-parallel-runners ./internal/booking/application/ 2>&1 | grep -E "\(depguard\)" | shape >out/depguard.txt
rm -f "$tmp"
# queries.sql のコメントは sqlc が生成コードに写すので、リージョンの印を入れない。
q=internal/booking/infrastructure/postgres/queries.sql
sql() { sed -n "/^-- name: $2 /,/;\$/p" "$q" >"out/$1.sql"; }
sql sql-active ListActiveSlotsByMember

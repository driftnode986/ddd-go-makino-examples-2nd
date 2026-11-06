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
logs sixth-row TestSixthRowIsSaved ./steps/tablerepo/
logs lost-update TestSecondSaveOverwritesFirst ./steps/noversion/
logs version-in-methods TestVersionInMethods ./steps/versionminusone/
logs outer-ctx TestOuterContextCommitsSeparately ./steps/autotx/
logs two-selects TestTwoSelectsMixVersions ./steps/twoselects/
logs unsaved TestUnsavedChangeIsVisible ./steps/sharedpointer/
logs raw-error TestUntranslatedErrorDiffersFromMemory ./steps/pitfalls/
logs no-btree-gist TestExclusionNeedsBtreeGist ./steps/pitfalls/
logs check-then-save TestCheckThenSaveLetsBothIn ./steps/pitfalls/
logs with-constraint TestConstraintStopsTheSecond ./steps/pitfalls/
TZ=UTC logs time-zone TestTimeZoneAfterRoundTrip ./steps/pitfalls/
logs lock-confirm TestConfirmAndOverlappingAdd ./steps/rawrepo/
logs lock-retry TestRetryWithoutLock ./steps/rawrepo/
logs nanosecond TestNanosecondIsTruncated ./steps/pitfalls/
logs lock-member TestLockIsPerMember ./steps/rawrepo/
# デッドロックのエラーは続きの行があるので、行の頭の字下げだけを落とす。
go test -count=1 -v -tags=integration -run TestDeadlockMessage ./steps/rawrepo/ 2>&1 |
	sed -n '/_test\.go:/,/^--- PASS/p' | grep -v '^--- PASS' |
	sed -E 's/^ +[a-z_]+_test\.go:[0-9]+: //; s/^ {8}//' | shape >out/deadlock.txt
# 2つの実装の契約テストの実行時間。末尾の時間は実行ごとに変わる。
go test -count=1 -tags=integration -run TestBookingRepository_Contract \
	./internal/booking/infrastructure/memory/ ./internal/booking/infrastructure/postgres/ 2>&1 |
	grep -E '^(ok|FAIL)' | sed -E 's#example.com/trainingbooking/internal/booking/infrastructure/##' | expand -t 4 | sed -E 's/ +/ /g' | shape >out/contract.txt
# queries.sql のコメントは sqlc が生成コードに写すので、リージョンの印を入れない。
# 本に載せる SQL はここで切り出す。
q=internal/booking/infrastructure/postgres/queries.sql
sql() { sed -n "/^-- name: $2 /,/;\$/p" "$q" >"out/$1.sql"; }
sql sql-update UpdateBooking
sql sql-find FindBookingWithOptions
sql sql-lock LockMember
# domain から生成コードを import すると、golangci-lint が止める。
tmp=internal/booking/domain/zz_import.go
cat >"$tmp" <<'GO'
package domain

import _ "example.com/trainingbooking/internal/booking/infrastructure/postgres/sqlcgen"
GO
golangci-lint run --allow-parallel-runners ./internal/booking/domain/ 2>&1 | grep -E "\(depguard\)" | shape >out/depguard.txt
rm -f "$tmp"
# コンテナの起動に失敗したログを本に載せない。
if grep -l 'コンテナの起動\|管理用の接続' out/*.txt; then
	echo "コンテナの起動に失敗したログがあります。実行し直します" >&2
	exit 1
fi

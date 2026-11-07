#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# Docker と jq が要る。別の章と重ならないように、ポートを変えて動かす。
# テストの結果は、t.Logf の中身だけを残す。
# 紙面に入らない行は 69 桁で折り返す。
set -u
shape() { expand -t 4 | fold -s -w 69 | sed "s/ *$//"; }
logs() { # 名前 パターン パッケージ
	result=$(go test -tags=integration -count=1 -v -timeout 300s -run "$2" "$3" 2>&1)
	# コンテナの起動の失敗などを、実行結果として残さない
	if ! echo "$result" | grep -q '^ok'; then
		echo "$result" >&2
		echo "out/$1.txt を作れませんでした" >&2
		exit 1
	fi
	echo "$result" | grep -E "_test\.go:[0-9]+:" |
		sed -E "s/^ +[a-z0-9_]+_test\.go:[0-9]+: //" | shape >"out/$1.txt"
}
# 本文の SQL を切り出す。sqlc がコメントを生成コードに写すので、
# queries.sql には印を入れない。
q=internal/platform/outbox/queries.sql
m=internal/platform/outbox/migrations/0003_create_outbox.sql
sed -n '/^-- name: InsertEvent /,/;$/p' "$q" | grep -v '^-- name' >out/sql-insert-event.sql
sed -n '/^-- name: ListDeliverable /,/;$/p' "$q" | grep -v '^-- name' >out/sql-deliverable.sql
sed -n '/^CREATE TABLE outbox /,/^);/p' "$m" >out/sql-outbox.sql
sed -n '/^CREATE TABLE processed_events /,/^);/p' "$m" >out/sql-processed.sql

logs seat-as-full TestConflictTreatedAsFull ./steps/seatconflict/
logs seat-retried TestConflictIsRetried ./steps/seatconflict/
logs dual-after TestPublishAfterCommit ./steps/dualwrite/
logs dual-before TestPublishBeforeCommit ./steps/dualwrite/
logs dual-outbox TestOutboxKeepsBoth ./steps/dualwrite/
logs split-rejected TestProcessThenMark_RejectionTwice ./steps/splitreceiver/
logs split-lost TestMarkThenProcess_EventIsLost ./steps/splitreceiver/
logs split-counted TestCountedSeats_ReservedTwice ./steps/splitreceiver/
logs split-same-tx TestSameTransaction ./steps/splitreceiver/
logs skip-locked TestSkipLockedOvertakes ./steps/orderbreak/
logs cancel-first TestCancelBeforePlaced ./steps/orderbreak/
logs batch TestFailuresFillTheBatch ./steps/orderbreak/
logs stall TestWarnThenError ./steps/stall/
logs deadlock-recovery TestRedeliveryRecoversFromDeadlock ./steps/confirmdeadlock/
logs deadlock-race TestConfirmWithoutLockRecovers ./steps/confirmdeadlock/

# Booking から Catalog の内部を import して、go build と depguard を見る。
f=internal/booking/application/zz_catalog.go
printf 'package application\n\nimport _ "example.com/trainingbooking/internal/catalog/domain"\n' >"$f"
go build ./internal/booking/application/ 2>&1
echo "go build の終了コード: $?" | shape >out/import-catalog.txt
golangci-lint run --allow-parallel-runners ./internal/booking/application/ 2>&1 |
	grep -E "\(depguard\)" | sed 's|^[^ ]*internal/booking/||; s|example.com/trainingbooking/||' | shape >>out/import-catalog.txt
rm -f "$f"

# 入れ子の internal と、複数 module。
go build ./steps/testdata/nested/catalog 2>&1 | grep 'use of internal' |
	sed -E 's|^[[:space:]]*steps/testdata/nested/||; s|example.com/trainingbooking/steps/testdata/nested/||' | shape >out/nested-internal.txt
(
	cd steps/testdata/workspace || exit 1
	go build ./booking/... ./catalog/... 2>&1 | grep 'use of internal' | sed -E 's|^[[:space:]]*||' | shape
	go test ./... 2>&1 | grep '^pattern' | shape
) >out/workspace.txt

# 起動して curl で動かす。ログは時刻を外して載せる。
export POSTGRES_PORT=55410
export DATABASE_URL="postgres://dev:dev@localhost:55410/trainingbooking?sslmode=disable"
export ADDR=localhost:18010
base="http://$ADDR"
log=$(mktemp)
bin=$(mktemp)
make -s down >/dev/null 2>&1
make -s up
make -s seed >/dev/null
go build -o "$bin" ./cmd/api
"$bin" >"$log" 2>&1 &
pid=$!
until curl -s -o /dev/null "$base/"; do sleep 0.2; done

book() { # 会員 レッスン オプションの JSON
	curl -s -X POST "$base/bookings" -H 'Content-Type: application/json' \
		-d "{\"member_id\":\"$1\",\"lesson_id\":\"$2\",\"options\":${3:-[]}}"
}
wait_status() { # 会員 予約
	for _ in $(seq 50); do
		state=$(curl -s "$base/members/$1/bookings" | jq -r --arg id "$2" '.bookings[] | select(.id == $id) | .status')
		[ "$state" != pending ] && break
		sleep 0.1
	done
	echo "$state"
}
m1=0196f5a0-0000-7000-8000-000000000001
m2=0196f5a0-0000-7000-8000-000000000002
yoga=0196f5a0-0000-7000-8001-000000000001
pilates=0196f5a0-0000-7000-8001-000000000002
missing=0196f5a0-0000-7000-8001-0000000000ff

body=$(book "$m1" "$yoga" '[{"name":"シューズ","quantity":2}]')
echo "$body" | jq -c '{status}' | shape >out/run-post.txt
id=$(echo "$body" | jq -r .id)
wait_status "$m1" "$id" >/dev/null
curl -s "$base/members/$m1/bookings" | jq -c '.bookings[] | {status,total_yen}' | shape >out/run-list.txt

# 料金は固定値でなくレッスンの値である。無いレッスンは 400 になる。
{
	first=$(book "$m1" "$pilates" | jq -r .id)
	echo "1人目: $(wait_status "$m1" "$first")"
	second=$(book "$m2" "$pilates" | jq -r .id)
	echo "2人目: $(wait_status "$m2" "$second")"
	curl -s -X POST "$base/bookings/$first/cancel" -o /dev/null
	for _ in $(seq 50); do
		[ "$(docker compose exec -T postgres psql -U dev -d trainingbooking -Atc "SELECT published_at IS NOT NULL FROM outbox WHERE event_type='booking.cancelled'")" = t ] && break
		sleep 0.1
	done
	echo "1人目をキャンセル"
	third=$(book "$m2" "$pilates" | jq -r .id)
	echo "2人目がもう一度: $(wait_status "$m2" "$third")"
} | shape >out/run-pilates.txt
curl -si -X POST "$base/bookings" -H 'Content-Type: application/json' \
	-d "{\"member_id\":\"$m2\",\"lesson_id\":\"$missing\"}" | tr -d '\r' | grep -E '^HTTP/|^\{' | shape >out/run-missing.txt

docker compose exec -T postgres psql -U dev -d trainingbooking -c \
	"SELECT event_type, aggregate_type AS aggregate, sequence AS seq, published_at IS NOT NULL AS published FROM outbox WHERE id > 2 ORDER BY id" | sed '$d' | sed '$d' | shape >out/run-outbox.txt

kill -TERM "$pid"
wait "$pid"
echo "終了コード: $?" >out/run-stop.txt
make -s down >/dev/null 2>&1
unset POSTGRES_PORT DATABASE_URL ADDR
rm -f "$log" "$bin"

#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# Docker が要る。5432 と 8080 が空いていること。
# テストの結果は、t.Logf の中身だけを残す。
# 紙面に入らない行は 69 桁で折り返す。
set -u
shape() { expand -t 4 | fold -s -w 69 | sed "s/ *$//"; }
logs() { # 名前 パターン パッケージ
	go test -count=1 -v -run "$2" "$3" 2>&1 | grep -E "_test\.go:[0-9]+:" | sed -E "s/^ +[a-z0-9_]+_test\.go:[0-9]+: //" | shape >"out/$1.txt"
}
logs encode TestEncodeBooking ./steps/encodeaggregate/
logs timezone TestSameInstantEncodesDifferently ./steps/timezone/
logs order-wrong TestRecoveryOutside ./steps/middlewareorder/
logs order-right TestRecoveryInside ./steps/middlewareorder/
logs shutdown 'TestCloseDropsRunningRequest|TestShutdownWaitsForRunningRequest' ./steps/shutdown/
logs server-timeouts 'TestReadHeaderTimeoutDoesNotLimitBody|TestReadTimeoutLimitsBody|TestWriteTimeoutDoesNotCancelHandler' ./steps/servertimeouts/

# 実際の import（テストを除く）。層と ids だけを出す。
list_imports() {
	for pkg in "$@"; do
		mine=$(go list -f '{{join .Imports "\n"}}' "./internal/booking/$pkg" | grep '^example.com/trainingbooking/' | grep -v sqlcgen | grep -v platform | sed 's|example.com/trainingbooking/internal/booking/||; s|example.com/trainingbooking/internal/||' | tr '\n' ' ')
		echo "$pkg -> $mine"
	done
}
list_imports domain application infrastructure/postgres presentation | sed 's/ *$//' | shape >out/imports.txt

# 依存の向きを破るファイルを足して、go build と golangci-lint の結果を見る。
violation() { # 名前 ファイル パッケージ名 import
	local f=$2
	printf 'package %s\n\nimport _ "%s"\n' "$3" "$4" >"$f"
	build=$(go build "./$(dirname "$f")/" 2>&1)
	echo "go build の終了コード: $?" | shape >"out/$1.txt"
	[ -n "$build" ] && echo "$build" | grep -v '^#' | sed 's|example.com/trainingbooking/internal/booking/||' | shape >>"out/$1.txt"
	golangci-lint run --allow-parallel-runners "./$(dirname "$f")/" 2>&1 | grep -E "\(depguard\)" | sed 's|^[^ ]*internal/booking/||' | shape >>"out/$1.txt"
	rm -f "$f"
}
violation import-repo internal/booking/presentation/zz_repo.go presentation example.com/trainingbooking/internal/booking/infrastructure/postgres
violation import-cycle internal/booking/domain/zz_cycle.go domain example.com/trainingbooking/internal/booking/application

# 起動して curl で動かす。go run はシグナルを子に渡さない場合があるので、
# ビルドしたものを動かす。ログは時刻と処理時間を外して載せる。
log=$(mktemp)
bin=$(mktemp)
make -s up
go build -o "$bin" ./cmd/api
TZ=UTC "$bin" >"$log" 2>&1 &
pid=$!
until curl -s -o /dev/null localhost:8080/; do sleep 0.2; done

member=0196f5a0-0000-7000-8000-000000000001
lesson=0196f5a0-0000-7000-8000-000000000002
body='{"member_id":"'$member'","lesson_id":"'$lesson'","options":[{"name":"シューズ","quantity":2}],"coupon_code":"WELCOME10"}'
# 状態行と X-Request-ID と本文だけを残す
post() { curl -si "$@" | tr -d '\r' | grep -E '^HTTP/|^X-Request-Id|^\{' | shape; }
post -X POST localhost:8080/bookings -H 'Content-Type: application/json' -d "$body" >out/run-post.txt
id=$(grep -o '"id":"[^"]*"' out/run-post.txt | cut -d'"' -f4)
post -X POST localhost:8080/bookings -H 'Content-Type: application/json' -d "$body" >out/run-dup.txt
curl -s "localhost:8080/members/$member/bookings" | jq . >out/run-list.txt
post -X POST "localhost:8080/bookings/$id/cancel" >out/run-cancel.txt
jq -c 'del(.time,.duration)' "$log" | grep -E '"event recorded"|"status":201' | shape >out/run-log.txt

docker compose stop -t 1 postgres >/dev/null 2>&1
# 1回目は切れた接続の後始末でエラーの文が変わるので、2回目を載せる
other=0196f5a0-0000-7000-8000-000000000003
down='{"member_id":"'$other'","lesson_id":"'$lesson'"}'
post -X POST localhost:8080/bookings -H 'Content-Type: application/json' -d "$down" >/dev/null
: >"$log.mark"; wc -l <"$log" >"$log.mark"
post -X POST localhost:8080/bookings -H 'Content-Type: application/json' -d "$down" >out/run-db-down.txt
tail -n "+$(($(cat "$log.mark") + 1))" "$log" | jq -c 'del(.time,.duration)' | grep '"request failed"' | shape >out/run-db-down-log.txt
rm -f "$log.mark"

kill -TERM "$pid"
wait "$pid"
echo "終了コード: $?" >out/run-stop.txt
jq -c 'del(.time,.duration)' "$log" | grep -E '"shutting down"' | shape >>out/run-stop.txt
make -s down >/dev/null 2>&1
rm -f "$log" "$bin"

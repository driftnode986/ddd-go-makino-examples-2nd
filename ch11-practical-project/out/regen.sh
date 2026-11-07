#!/bin/bash
# 本に載せる実行結果を作り直す。章のディレクトリで実行する。
#   bash out/regen.sh
# Docker と jq が要る。別の章と重ならないように、ポートを変えて動かす。
# 紙面に入らない行は 69 桁で折り返す。
set -u
shape() { expand -t 4 | fold -s -w 69 | sed "s/ *$//"; }
psql_run() { docker compose exec -T postgres psql -U dev -d trainingbooking "$@"; }

export POSTGRES_PORT=55411
export DATABASE_URL="postgres://dev:dev@localhost:55411/trainingbooking?sslmode=disable"
export ADDR=localhost:18011
export API=$ADDR
source scripts/try.sh
ghost=0196f5a0-0000-7000-8000-00000000ffff

start() { # ビルドしたサーバー
	# 別のサーバーが待ち受けていたら、その応答を結果にしない
	if curl -s -o /dev/null --max-time 1 "$API/"; then
		echo "$API は別のサーバーが使っています" >&2
		exit 1
	fi
	"$1" >"$log" 2>&1 &
	pid=$!
	until curl -s -o /dev/null "$API/"; do sleep 0.1; done
}
stop() {
	kill -TERM "$pid"
	wait "$pid"
}
log=$(mktemp)
work=$(mktemp -d)
make -s down >/dev/null 2>&1

# 第10章のサーバーと DB で、存在しない会員が席を取る。
(
	cd ../ch10-bounded-context || exit 1
	make -s down >/dev/null 2>&1
	make -s up
	make -s seed >/dev/null
	go build -o "$work/api10" ./cmd/api
)
start "$work/api10"
{
	post "$ghost" "$PILATES" | head -n 1 | jq -c '{status}'
	id=$(curl -s "$API/members/$ghost/bookings" | jq -r '.bookings[0].id')
	echo "状態: $(wait_status "$ghost" "$id")"
	seats=$(cd ../ch10-bounded-context && psql_run -Atc "SELECT count(*) FROM lesson_seats WHERE lesson_id = '$PILATES'")
	echo "ピラティスの席: $seats / 定員1"
} | shape >out/ghost.txt
stop
(cd ../ch10-bounded-context && make -s down >/dev/null 2>&1)
make -s up
make -s seed >/dev/null

# 第10章からの差分（テストと生成コードを除く）。
diff -rq ../ch10-bounded-context/internal internal | grep -v '_test.go' | grep -v 'sqlcgen' |
	sed -E 's|^Files \.\./ch10-bounded-context/internal/([^ ]+) and .*|変更: \1|; s|^Only in internal/?([^:]*): (.*)|追加: \1/\2|' |
	grep -v 'identity/infrastructure/postgres/migrations' | shape >out/diff-dirs.txt
diff ../ch10-bounded-context/cmd/api/app.go cmd/api/app.go | grep -E '^[<>]' |
	sed 's|example.com/trainingbooking/internal/|…/|' | shape >out/diff-app.txt

# SQL で検証を通らない会員を入れると、予約のときに初めて分かる。
go build -o "$work/api11" ./cmd/api
start "$work/api11"
{
	psql_run -c "INSERT INTO members (id, email, name) VALUES ('0196f5a0-0000-7000-8000-000000000009', 'hanako', '花子')" | sed '$d'
	post 0196f5a0-0000-7000-8000-000000000009 "$YOGA"
} | shape >out/bad-seed.txt
sleep 0.2
grep '"level":"ERROR"' "$log" | jq -r '"\(.level) \(.msg): \(.error)"' | shape >out/bad-seed-log.txt
stop
make -s reset >out/seed.txt 2>/dev/null
go test -count=1 -v -run TestNewMember_RejectsInvalidEmail ./cmd/seed/ 2>&1 |
	grep -E "_test\.go:[0-9]+:" | sed -E "s/^ +[a-z0-9_]+_test\.go:[0-9]+: //" | shape >out/seed-invalid.txt

# カバレッジつきでビルドして、手順どおりに動かす。
export GOCOVERDIR="$work/cov"
mkdir -p "$GOCOVERDIR"
go build -cover -coverpkg=./... -o "$work/seed" ./cmd/seed
"$work/seed" >/dev/null
go build -cover -coverpkg=./... -o "$work/api" ./cmd/api
start "$work/api"

# 予約して確定する
post "$HANAKO" "$YOGA" '"options":[{"name":"シューズ","quantity":2}]' |
	sed -E 's/"id":"[^"]*",?//' | shape >out/run-post.txt
sleep 1
curl -s "$API/members/$HANAKO/bookings" | jq -c '.bookings[] | {status,total_yen,options}' | shape >out/run-confirmed.txt

# クーポン
post "$TARO" "$YOGA" '"options":[{"name":"シューズ","quantity":1}],"coupon_code":"WELCOME10"' >/dev/null
sleep 1
curl -s "$API/members/$TARO/bookings" | jq -c '.bookings[] | {status,total_yen,options}' | shape >out/run-coupon.txt

# 満席とキャンセル
FIRST=$(book "$HANAKO" "$PILATES")
{
	echo "花子を待たずにキャンセル:"
	cancel "$FIRST"
	echo "花子: $(wait_status "$HANAKO" "$FIRST")"
	SECOND=$(book "$TARO" "$PILATES")
	echo "太郎: $(wait_status "$TARO" "$SECOND")"
	echo "花子をキャンセル:"
	cancel "$FIRST" | sed -E 's/"id":"[^"]*",?//'
	THIRD=$(book "$TARO" "$PILATES")
	echo "太郎がもう一度: $(wait_status "$TARO" "$THIRD")"
} | shape >out/run-pilates.txt
if ! grep -q '^409$' out/run-pilates.txt; then
	echo "待たずにキャンセルが通りました。もう一度実行してください" >&2
	exit 1
fi

# 同期で拒否するもの
{
	echo "花子が同じ時間帯を予約:"
	post "$HANAKO" "$STRENGTH"
	echo "存在しない会員:"
	post "$ghost" "$YOGA"
	echo "不明なクーポン:"
	post "$TARO" "$STRENGTH" '"coupon_code":"UNKNOWN"'
	echo "存在しない会員の一覧:"
	curl -s -w '%{http_code}\n' "$API/members/$ghost/bookings"
} | shape >out/run-rejected.txt

psql_run -c "SELECT event_type, sequence AS seq, attempts, published_at IS NOT NULL AS published FROM outbox ORDER BY id" |
	sed '$d' | sed '$d' | shape >out/run-outbox.txt

stop
echo "終了コード: $?" >out/run-stop.txt
unset GOCOVERDIR

go tool covdata textfmt -i="$work/cov" -o "$work/cov.txt"
go tool cover -func="$work/cov.txt" |
	sed -E 's|example.com/trainingbooking/internal/||; s|:[0-9]+:[[:space:]]+| |; s|[[:space:]]+| |g' >"$work/f.txt"
# 流れの部品。名前の重複は親のファイルで絞る。
pick() { grep -E "^$1 ($2) " "$work/f.txt" | awk '$3!="0.0%"{n=split($1,p,"/"); printf "%-20s %-20s %s\n", p[n], $2, $3}'; }
{
	pick identity/domain/email.go 'NewEmail'
	pick identity/domain/member.go 'NewMember|ReconstructMember'
	pick booking/domain/money.go 'NewMoney|Add|Multiply'
	pick booking/domain/time_slot.go 'NewTimeSlot|Overlaps'
	pick booking/domain/discount.go 'NewCoupon|CalculateDiscount'
	pick booking/domain/overlap.go 'DetectOverlap'
	pick booking/domain/booking_option.go 'NewBookingOption'
	pick booking/domain/booking.go 'NewBooking|AddOption|ApplyDiscount|Total|Confirm|Reject|Cancel|PullEvents'
	pick catalog/domain/lesson.go 'NewLesson|ReserveSeat|ReleaseSeat'
	pick booking/application/place_booking.go 'Execute'
	pick booking/application/cancel_booking.go 'Execute'
	pick platform/outbox/store.go 'Append'
	pick platform/outbox/relay.go 'RunOnce'
	pick platform/outbox/processed.go 'MarkProcessed'
} | shape >out/coverage-parts.txt
# domain で一度も通らなかったもの。
grep -E '^(booking|catalog|identity)/domain/.* 0\.0%$' "$work/f.txt" |
	awk '{n=split($1,p,"/"); printf "%-20s %s\n", p[n], $2}' | shape >out/coverage-zero.txt

# テストの層ごとの時間（手元の一例）。統合テストは Docker を使う。
go test -tags=integration -count=1 ./cmd/... ./internal/booking/... ./internal/catalog/domain/ 2>&1 |
	grep -E '^ok' | grep -E 'booking/domain|booking/application|booking/presentation|booking/infrastructure/(memory|postgres)|catalog/domain|cmd/api' |
	sed -E 's|example.com/trainingbooking/||; s|internal/||; s|^ok[[:space:]]+||' | awk '{printf "%-36s %s\n",$1,$2}' | shape >out/test-layers.txt
make -s down >/dev/null 2>&1
unset POSTGRES_PORT DATABASE_URL ADDR API
rm -rf "$work" "$log"

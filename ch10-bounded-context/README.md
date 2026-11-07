# 第10章 境界づけられたコンテキストとドメインイベント

Catalog を足し、イベントを outbox と配信処理で流します。予約が `confirmed` と `rejected` に進みます。

## できるようになったこと

- `Lesson` 集約が定員を守ります。残席1を2つの処理が読んでも、1件だけが確保できます
- `BookingPlaced` を予約と同じ Tx で outbox に書き、配信処理が Catalog に渡します
- 再配信されても、処理済みの記録と集約の保存を同じ Tx で行うので、席は1回だけ動きます
- `confirmed` をキャンセルすると席が戻ります
- コンテキスト間の import を `contract` だけに絞り、`golangci-lint` の depguard で止めます
- 時間帯・料金・オプションの単価を、第9章の固定値でなく Catalog のレッスンから取ります

会員のコンテキストはまだ無いので、`member_id` は形式が正しければ何でも受け付けます。

## 第9章からの変更

- 追加: `internal/catalog/`、`internal/platform/outbox/`、`internal/booking/contract/`、`internal/booking/infrastructure/acl/lessons.go`、`internal/booking/application/seat_result_handler.go`、`internal/booking/infrastructure/postgres/outbox.go`、`cmd/api/app.go`、`scripts/seed.sql`
- 変更: `place_booking.go` と `lessons.go`（`LessonTerms` を固定値でなく `LessonCatalog` から取る）、`main.go`（配信処理の起動と停止）
- 削除: `LoggingOutbox`

## 本文の失敗を動かす

`steps/` に本文で動かした失敗があります。実行結果は `out/` にあり、`bash out/regen.sh` で作り直せます（Docker と jq が要ります。数分かかります）。

- `steps/seatconflict`: 版の競合を満席と取り違える / 読み直す
- `steps/dualwrite`: 保存とイベントが片方だけ成功する / outbox
- `steps/splitreceiver`: 受け手の処理を Tx で分けると再配信で結果が変わる / 同じ Tx
- `steps/orderbreak`: キャンセルの追い越し、失敗で埋まる枠、`SKIP LOCKED`
- `steps/confirmdeadlock`: 確定の 40P01 が再配信で回復する（約1分）
- `steps/stall`: 配信の失敗が `WARN` から `ERROR` に変わる

```bash
go test -v -tags=integration ./steps/splitreceiver/
```

## 動かし方

```bash
cd ch10-bounded-context
go test ./...
go test -tags=integration ./...
make up      # PostgreSQL を起動し、初回はスキーマを流す
make seed    # レッスンの初期データを入れる
make run     # API と配信処理を1つのプロセスで起動する
```

PostgreSQL は `127.0.0.1:5432` で待ち受けます。別の章で起動したままのときは、その章のディレクトリで `make down` を実行してから起動します。ポートを変えるときは環境変数 `POSTGRES_PORT` を指定します。接続先は環境変数 `DATABASE_URL`、待ち受けるアドレスは `ADDR` で変えられます。

初期データのレッスンの日時は、初めて `make seed` を実行した日の30日後です。もう一度流しても日時は変わりません。日が過ぎて予約できなくなったら `make reset` で作り直します（予約も消えます）。

- `0196f5a0-0000-7000-8001-000000000001` 朝のヨガ。定員10、10時から。オプションはシューズ（300円）とタオル（100円）
- `0196f5a0-0000-7000-8001-000000000002` 少人数ピラティス。定員1、14時から
- `0196f5a0-0000-7000-8001-000000000003` 朝の筋力トレーニング。定員10、10時から

別のターミナルから予約します。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000001",
    "lesson_id": "0196f5a0-0000-7000-8001-000000000001",
    "options": [{"name": "シューズ", "quantity": 2}]
  }'
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000001/bookings
```

クーポンのコードを付けると、合計から割り引きます。使えるコードは `WELCOME10`（10%）です。不明なコードは `400` を返します。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000003",
    "lesson_id": "0196f5a0-0000-7000-8001-000000000001",
    "coupon_code": "WELCOME10"
  }'
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000003/bookings
```

少し待って一覧を読むと `"status":"confirmed"`・`"total_yen":2700` になっています。

オプションは予約のときに付けます。後から足したり外したりする API はありません。レッスンに無い名前、0以下の数量、同じ名前の重複、金額のあふれは `400` を返します。

### 満席とキャンセル

定員1のレッスンで、満席の拒否とキャンセルで席が戻るところを確かめます。手順には `jq` を使います。まず、会員とレッスンの ID と、次の2つの関数を同じターミナルで定義します。

```bash
# 予約して id を表示する（$1=会員 $2=レッスン）
book() {
  curl -s -X POST localhost:8080/bookings \
    -H 'Content-Type: application/json' \
    -d "{\"member_id\":\"$1\",\"lesson_id\":\"$2\"}" | jq -r .id
}
# pending でなくなるまで待ち、状態を表示する（$1=会員 $2=予約）
wait_status() {
  for _ in $(seq 50); do
    state=$(curl -s "localhost:8080/members/$1/bookings" |
      jq -r --arg id "$2" \
        '.bookings[] | select(.id == $id) | .status')
    [ "$state" != pending ] && break
    sleep 0.2
  done
  echo "$state"
}

MEMBER1=0196f5a0-0000-7000-8000-000000000001
MEMBER2=0196f5a0-0000-7000-8000-000000000002
PILATES=0196f5a0-0000-7000-8001-000000000002
```

1人目が定員1のレッスンを予約します。キャンセルできるのは `confirmed` だけなので（`pending` のキャンセルは `409`）、状態を確かめてから次へ進みます。

```bash
FIRST=$(book "$MEMBER1" "$PILATES")
wait_status "$MEMBER1" "$FIRST"
```

`confirmed` と表示されます。同じレッスンを2人目が予約すると、`201` が返った後で `rejected` になります。

```bash
SECOND=$(book "$MEMBER2" "$PILATES")
wait_status "$MEMBER2" "$SECOND"
```

`rejected` と表示されます。1人目の予約をキャンセルすると席が戻り、2人目がもう一度予約すると `confirmed` になります。

```bash
curl -s -X POST "localhost:8080/bookings/$FIRST/cancel"
echo
THIRD=$(book "$MEMBER2" "$PILATES")
wait_status "$MEMBER2" "$THIRD"
```

キャンセルの応答は `"status":"cancelled"` で、最後に `confirmed` と表示されます。

### 止める


止めるときは `make run` のターミナルで Ctrl+C を押し、`make down` を実行します。

## 期待する出力

予約は `201` と `"status":"pending"` を返します。席の確保は配信処理が行うので、少し待って一覧を読むと次のようになります。`id` と日時は実行ごとに変わります。

```json
{"bookings":[{"id":"…","lesson_id":"0196f5a0-0000-7000-8001-000000000001","start":"…","end":"…","status":"confirmed","total_yen":3600,"options":[{"name":"シューズ","quantity":2,"subtotal_yen":600}]}]}
```

`go test -tags=integration ./...` は次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。 `steps/` のテストも `ok` で出ます（下の一覧には載せていません）。

```text
ok  	example.com/trainingbooking/cmd/api
ok  	example.com/trainingbooking/internal/booking/application
ok  	example.com/trainingbooking/internal/booking/contract
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/booking/infrastructure/acl
ok  	example.com/trainingbooking/internal/booking/infrastructure/fixed
ok  	example.com/trainingbooking/internal/booking/infrastructure/memory
ok  	example.com/trainingbooking/internal/booking/infrastructure/postgres
ok  	example.com/trainingbooking/internal/booking/presentation
ok  	example.com/trainingbooking/internal/catalog/application
ok  	example.com/trainingbooking/internal/catalog/contract
ok  	example.com/trainingbooking/internal/catalog/domain
ok  	example.com/trainingbooking/internal/catalog/infrastructure/memory
ok  	example.com/trainingbooking/internal/catalog/infrastructure/postgres
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
ok  	example.com/trainingbooking/internal/platform/logging
ok  	example.com/trainingbooking/internal/platform/outbox
ok  	example.com/trainingbooking/internal/platform/txn/memtx
```

`pending` が続いたときの扱いは、第11章の README の「配信が遅れたとき」と同じです。第11章で Identity を足すと、存在しない会員の予約を拒否できるようになります。

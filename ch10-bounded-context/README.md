# 第10章 境界づけられたコンテキストとドメインイベント

Catalog を足し、イベントを outbox と配信処理で流します。予約が `confirmed` と `rejected` に進みます。

## できるようになったこと

- `Lesson` 集約が定員を守ります。残席1を2つの処理が読んでも、1件だけが確保できます
- `BookingPlaced` を予約と同じ Tx で outbox に書き、配信処理が Catalog に渡します
- 再配信されても、処理済みの記録と集約の保存を同じ Tx で行うので、席は1回だけ動きます
- `confirmed` をキャンセルすると席が戻ります
- コンテキスト間の import を `contract` だけに絞り、`golangci-lint` の depguard で止めます
- 時間帯と料金をレッスンから取ります。オプションも指定でき、一覧の `options` に明細が出ます

会員のコンテキストはまだ無いので、`member_id` は形式が正しければ何でも受け付けます。

## 第9章からの変更

- 追加: `internal/catalog/`、`internal/platform/outbox/`、`internal/booking/contract/`、`internal/booking/infrastructure/acl/lessons.go`、`internal/booking/application/` の `lessons.go`・`seat_result_handler.go`、`internal/booking/infrastructure/postgres/outbox.go`、`cmd/api/app.go`、`scripts/seed.sql`
- 変更: `place_booking.go` と HTTP の DTO（時間帯と料金を受け取らず、オプションを受け取る）、`main.go`（配信処理の起動と停止）
- 削除: `LoggingOutbox`

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

定員1のレッスンは、別の `member_id` で2回予約すると2件目が `rejected` になります。`POST /bookings/{id}/cancel` で1件目をキャンセルすると席が戻り、次の予約は `confirmed` になります。

止めるときは `make run` のターミナルで Ctrl+C を押し、`make down` を実行します。

## 期待する出力

予約は `201` と `"status":"pending"` を返します。席の確保は配信処理が行うので、少し待って一覧を読むと次のようになります。`id` と日時は実行ごとに変わります。

```json
{"bookings":[{"id":"…","lesson_id":"0196f5a0-0000-7000-8001-000000000001","start":"…","end":"…","status":"confirmed","total_yen":3600,"options":[{"name":"シューズ","quantity":2,"subtotal_yen":600}]}]}
```

`go test -tags=integration ./...` は次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

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

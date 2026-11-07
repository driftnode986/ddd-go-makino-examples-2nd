# 第9章 レイヤー化アーキテクチャとパッケージ分割

HTTP の層と `main` を足して、初めて `go run` で動かします。

## できるようになったこと

- `POST /bookings`・`POST /bookings/{id}/cancel`・`GET /members/{id}/bookings` を `net/http` の `ServeMux` で受けられます
- `POST /bookings` は任意の `options`（名前と数量）と `coupon_code` を受け取ります。時間帯・料金・単価・割引率は受け取りません
- `ApplicationError` の `Kind` を HTTP ステータスに変換します。500 の本文に内部の文字列を出しません
- `Recovery`・`RequestID`・`Logging` のミドルウェアで、すべてのリクエストをログに記録します
- SIGTERM で処理中のリクエストを待って止まります
- 一覧の時刻は `TZ` によらず JST で返します。1リクエストの処理には5秒の期限があります

イベントは `LoggingOutbox` がログに書くだけで、配信はしません。そのため予約は `pending` のままで、確定とキャンセルはまだ成功しません。

## 第8章からの変更

- 追加: `internal/booking/presentation/`、`cmd/api/`、`internal/booking/infrastructure/postgres/outbox_log.go`
- 変更: `Makefile` に `run` を追加
- `steps/` と `out/` はこの章の失敗の例と実行結果です。`bash out/regen.sh` で作り直せます（Docker が要ります。5432 と 18089 を使います）

## 動かし方

```bash
cd ch09-layered-architecture
go test ./...
go test -tags=integration ./...
make up
go run ./cmd/api
```

PostgreSQL は `127.0.0.1:5432` で待ち受けます。別の章で起動したままのときは、その章のディレクトリで `make down` を実行してから起動します。ポートを変えるときは環境変数 `POSTGRES_PORT` を指定します。接続先は環境変数 `DATABASE_URL`、待ち受けるアドレスは `ADDR` で変えられます。

この章のレッスンの条件は `cmd/api` の固定値で、1通りです。時間帯は起動した日の30日後の10時から1時間、料金は3000円、オプションはシューズ（300円）とタオル（100円）です。`lesson_id` は形式が正しければ何でも受け付けます。

別のターミナルから予約します。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000001",
    "lesson_id": "0196f5a0-0000-7000-8000-000000000002",
    "options": [{"name": "シューズ", "quantity": 2}]
  }'
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000001/bookings
```

クーポンのコードを付けると、合計から割り引きます。使えるコードは `WELCOME10`（10%）です。時間帯は1通りなので、別の会員で予約します。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000003",
    "lesson_id": "0196f5a0-0000-7000-8000-000000000002",
    "coupon_code": "WELCOME10"
  }'
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000003/bookings
```

止めるときは `go run` のターミナルで Ctrl+C を押し、`make down` を実行します。

## 期待する出力

予約は `201` で次の本文を返します。`id` は実行ごとに変わります。

```json
{"id":"01a1b10d-f4db-713f-ba48-e8fff14ad498","status":"pending"}
```

同じ内容でもう一度予約すると、`409` と次の本文を返します。

```json
{"kind":"conflict","message":"同じ時間帯に別の予約があります"}
```

一覧には次の予約が出ます。`id` と日時は実行ごとに変わります。

```json
{"bookings":[{"id":"…","lesson_id":"0196f5a0-0000-7000-8000-000000000002","start":"…","end":"…","status":"pending","total_yen":3600,"options":[{"name":"シューズ","quantity":2,"subtotal_yen":600}]}]}
```

クーポンつきの予約の合計は `"total_yen":2700` で、`options` は `[]` です。

`coupon_code` を `UNKNOWN` のような不明なコードにすると、`400` と次の本文を返し、予約は保存されません。

```json
{"kind":"invalid","message":"クーポンのコードが正しくありません"}
```

レッスンに無いオプション名も `400` で、本文の `message` は「レッスンに無いオプションです」です。0以下の数量、同じ名前の重複、金額のあふれも `400` です。

`go test -tags=integration ./...` は次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/cmd/api
ok  	example.com/trainingbooking/internal/booking/application
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/booking/infrastructure/fixed
ok  	example.com/trainingbooking/internal/booking/infrastructure/memory
ok  	example.com/trainingbooking/internal/booking/infrastructure/postgres
ok  	example.com/trainingbooking/internal/booking/presentation
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
ok  	example.com/trainingbooking/internal/platform/logging
ok  	example.com/trainingbooking/internal/platform/txn/memtx
```

定員を見る処理はまだありません。第10章で Catalog を足し、`LessonTerms` の取得元を固定値から Catalog に入れ替えます。

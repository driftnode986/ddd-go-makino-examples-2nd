# 第9章 レイヤー化アーキテクチャとパッケージ分割

HTTP の層と `main` を足して、初めて `go run` で動かします。

## できるようになったこと

- `POST /bookings`・`POST /bookings/{id}/cancel`・`GET /members/{id}/bookings` を `net/http` の `ServeMux` で受けられます
- `POST /bookings` は任意の `coupon_code` を受け取ります。割引率は受け取りません
- `ApplicationError` の `Kind` を HTTP ステータスに変換します。500 の本文に内部の文字列を出しません
- `Recovery`・`RequestID`・`Logging` のミドルウェアで、すべてのリクエストをログに記録します
- SIGTERM で処理中のリクエストを待って止まります

イベントは `LoggingOutbox` がログに書くだけで、配信はしません。そのため予約は `pending` のままで、確定とキャンセルはまだ成功しません。

## 第8章からの変更

- 追加: `internal/booking/presentation/`、`cmd/api/`、`internal/booking/infrastructure/postgres/outbox_log.go`
- 変更: `Makefile` に `run` を追加

## 動かし方

```bash
cd ch09-layered-architecture
go test ./...
go test -tags=integration ./...
make up
go run ./cmd/api
```

PostgreSQL は `127.0.0.1:5432` で待ち受けます。別の章で起動したままのときは、その章のディレクトリで `make down` を実行してから起動します。ポートを変えるときは環境変数 `POSTGRES_PORT` を指定します。接続先は環境変数 `DATABASE_URL`、待ち受けるアドレスは `ADDR` で変えられます。

別のターミナルから予約します。`start` は未来の日時にします。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000001",
    "lesson_id": "0196f5a0-0000-7000-8000-000000000002",
    "start": "2030-06-01T10:00:00+09:00",
    "duration_minutes": 60
  }'
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000001/bookings
```

クーポンのコードを付けると、合計から割り引きます。使えるコードは `WELCOME10`（10%）です。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000001",
    "lesson_id": "0196f5a0-0000-7000-8000-000000000002",
    "start": "2030-06-01T14:00:00+09:00",
    "duration_minutes": 60,
    "coupon_code": "WELCOME10"
  }'
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

一覧には `"status":"pending"`・`"total_yen":3000` の予約が出ます。クーポンつきの予約の合計は `"total_yen":2700` です。

`coupon_code` を `UNKNOWN` のような不明なコードにすると、`400` と次の本文を返し、予約は保存されません。

```json
{"kind":"invalid","message":"クーポンのコードが正しくありません"}
```

`go test -tags=integration ./...` は次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/cmd/api
ok  	example.com/trainingbooking/internal/booking/application
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/booking/infrastructure/memory
ok  	example.com/trainingbooking/internal/booking/infrastructure/postgres
ok  	example.com/trainingbooking/internal/booking/presentation
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
ok  	example.com/trainingbooking/internal/platform/logging
ok  	example.com/trainingbooking/internal/platform/txn/memtx
```

定員を見る処理はまだありません。第10章で Catalog を足します。

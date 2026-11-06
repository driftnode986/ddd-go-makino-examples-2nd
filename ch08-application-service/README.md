# 第8章 アプリケーションサービスとユースケース

リポジトリとドメインを呼ぶ手順を、ユースケースにまとめます。

## できるようになったこと

- `PlaceBooking`・`CancelBooking` が入力を値に変え、1つの Tx で保存します。`application` は `pgx` を import しません
- 集約が記録したイベントを、同じ Tx で `OutboxWriter` に書きます。書けなければ予約も残りません
- エラーを `ApplicationError` の `Kind` に変換し、外に出す文と原因を分けます
- `BookingQueries` で集約を組み立てずに一覧を読めます
- `request_id` を context から読む `slog` のハンドラで、全ログに付けられます

この章の予約は、時間帯を入力で受け、料金を `LessonTerms` で固定します。レッスンから取る形は第10章で作ります。

## 第7章からの変更

- 追加: `internal/booking/application/`、`internal/booking/domain/events.go`、`internal/platform/logging/`、`internal/booking/infrastructure/memory/` の `outbox.go`・`queries.go`、`internal/booking/infrastructure/postgres/queries.go`
- 変更: `booking.go` がイベントを記録します。`queries.sql` と生成コードに一覧を追加しました

## 動かし方

```bash
cd ch08-application-service
go test ./...
go test -tags=integration ./...
```

ロールバックのテストだけを見るときは次を実行します。

```bash
go test -v -tags=integration -run TestPlaceBooking_RollsBackWhenOutboxFails ./internal/booking/application/
```

## 期待する出力

`go test -tags=integration ./...` は次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/internal/booking/application
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/booking/infrastructure/memory
ok  	example.com/trainingbooking/internal/booking/infrastructure/postgres
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
ok  	example.com/trainingbooking/internal/platform/logging
ok  	example.com/trainingbooking/internal/platform/txn/memtx
```

ユースケースはまだ外から呼べません。第9章で HTTP につなぎます。

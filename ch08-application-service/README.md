# 第8章 アプリケーションサービスとユースケース

リポジトリとドメインを呼ぶ手順を、ユースケースにまとめます。

## できるようになったこと

- `PlaceBooking`・`CancelBooking` が入力を値に変え、1つの Tx で保存します。`application` は `pgx` を import しません
- 集約が記録したイベントを、同じ Tx で `OutboxWriter` に書きます。書けなければ予約も残りません
- エラーを `ApplicationError` の `Kind` に変換し、外に出す文と原因を分けます
- `PlaceBooking` が第5章のドメインサービスを呼びます。同じ会員の有効な予約の時間帯を読んで `DetectOverlap` で確かめ、`CouponCode` があれば `Coupons` から引いた `Coupon` で `CalculateDiscount` を計算して `ApplyDiscount` します。割引率は入力で受け取りません。不明なコードは `KindInvalid` です
- `PlaceBooking` がオプションを受け付けます。入力は名前と数量だけで、単価は `LessonTerms` の値を使います。レッスンに無い名前は `ErrUnknownOption` で、数量の誤り、同じ名前の重複、数の上限、期限切れ、金額のあふれと同じく `KindInvalid` です
- 読み取りの後に同時に来た予約は、第7章の排他制約が `ErrOverlap` で止めます
- `BookingQueries` で集約を組み立てずに、明細を含む一覧と有効な予約の時間帯を読めます
- `request_id` を context から読む `slog` のハンドラで、全ログに付けられます

この章の予約は、時間帯・料金・オプションの単価を `LessonTerms` から取ります。取得元を Catalog に入れ替えるのは第10章です。

## 第7章からの変更

- 追加: `internal/booking/application/`、`internal/booking/infrastructure/fixed/`（クーポンの固定の表）、`internal/booking/domain/events.go`、`internal/platform/logging/`、`internal/booking/infrastructure/memory/` の `outbox.go`・`queries.go`、`internal/booking/infrastructure/postgres/queries.go`
- 変更: `booking.go` がイベントを記録します。`booking_state.go` に `ReconstructTimeSlot` を追加しました。`queries.sql` と生成コードに一覧と有効な予約の時間帯を追加しました

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
ok  	example.com/trainingbooking/internal/booking/infrastructure/fixed
ok  	example.com/trainingbooking/internal/booking/infrastructure/memory
ok  	example.com/trainingbooking/internal/booking/infrastructure/postgres
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
ok  	example.com/trainingbooking/internal/platform/logging
ok  	example.com/trainingbooking/internal/platform/txn/memtx
```

ユースケースはまだ外から呼べません。第9章で HTTP につなぎます。

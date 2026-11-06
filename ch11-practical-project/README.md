# 完成形（第11章）

3つのコンテキストを1つの module に置く予約システムです。Booking が予約、Catalog がレッスンと定員、Identity が会員を受け持ちます。

## 第10章からの変更

Identity を足しました。`Member` 集約と `contract`、Booking 側の腐敗防止層（`internal/booking/infrastructure/acl/members.go`）で、存在しない会員の予約を `400` で拒否します。初期データに会員が増えました。

## 確かめ方

```bash
go test ./...
go test -tags=integration ./...
golangci-lint run ./...
```

`-tags=integration` のテストは Docker で PostgreSQL を起動します。

## 動かし方

```bash
make up      # PostgreSQL を起動し、初回はスキーマを流す
make seed    # 会員とレッスンの初期データを入れる
make run     # API と配信処理を1つのプロセスで起動する
```

接続先は `postgres://dev:dev@localhost:5432/trainingbooking` です。別の章で PostgreSQL を起動したままのときは、その章のディレクトリで `make down` を実行してから起動します。ポートを変えるときは環境変数 `POSTGRES_PORT` と `DATABASE_URL` を指定します。待ち受けるアドレスは `ADDR` で変えられます。

初期データのレッスンの日時は、初めて `make seed` を実行した日の30日後です。もう一度流しても日時は変わりません。日が過ぎて予約できなくなったら `make reset` で作り直します（予約も消えます）。

- `0196f5a0-0000-7000-8001-000000000001` 朝のヨガ。定員10、10時から。オプションはシューズ（300円）とタオル（100円）
- `0196f5a0-0000-7000-8001-000000000002` 少人数ピラティス。定員1、14時から
- `0196f5a0-0000-7000-8001-000000000003` 朝の筋力トレーニング。定員10、10時から

会員は `0196f5a0-0000-7000-8000-000000000001`（花子）と `0196f5a0-0000-7000-8000-000000000002`（太郎）です。

## 予約して確定する

別のターミナルから予約します。時間帯と料金はレッスンから取るので、指定するのは会員とレッスンとオプションです。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000001",
    "lesson_id": "0196f5a0-0000-7000-8001-000000000001",
    "options": [{"name": "シューズ", "quantity": 2}]
  }'
```

`201` と `"status":"pending"` が返ります。席の確保は配信処理が行うので、少し待って一覧を読むと `confirmed` になっています。合計は3600円です。

```bash
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000001/bookings
```

## 満席・キャンセル・入力の誤り

定員1のレッスンを花子が予約します。`id` を控えておきます。

```bash
curl -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000001",
    "lesson_id": "0196f5a0-0000-7000-8001-000000000002"
  }'
```

同じレッスンを太郎が予約すると、`201` が返った後で `rejected` になります。

```bash
curl -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000002",
    "lesson_id": "0196f5a0-0000-7000-8001-000000000002"
  }'
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000002/bookings
```

花子の `confirmed` の予約をキャンセルすると席が戻ります。太郎がもう一度予約すると、今度は `confirmed` になります。

```bash
curl -X POST localhost:8080/bookings/控えた予約のID/cancel
```

次の2つは同期で拒否します。

- 花子が同じ時間帯の朝の筋力トレーニング（`...8001-000000000003`）を予約すると `409`
- 存在しない会員（例: `0196f5a0-0000-7000-8000-00000000ffff`）で予約すると `400`

## 配信が遅れたとき

`pending` が続く時間の許容は5分です。配信が失敗している間も止めずに再配信を続け、`GET /members/{id}/bookings` では `pending` のまま見えます。失敗は `WARN` でログに残し、5分を超えたら `ERROR` に上げます。原因は `outbox` テーブルの `attempts` と `last_error` で確かめられます。止まるのは同じ集約のイベントだけです。配信処理は失敗の少ないイベントから配るので、失敗が続く集約が1回に配る件数の上限（100件）を埋めても、ほかの集約の配信は続きます。

## やり直す

```bash
make reset   # データを消して、起動と初期データの投入をやり直す
make down    # 停止してデータも消す
```

## 生成

```bash
make sqlc    # queries.sql からコードを生成する
```

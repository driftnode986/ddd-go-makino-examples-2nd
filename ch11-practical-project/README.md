# 完成形（第11章）

3つのコンテキストを1つの module に置く予約システムです。

## 確かめ方

```bash
go test ./...
go test -tags=integration ./...
golangci-lint run ./...
```

`-tags=integration` のテストは Docker で PostgreSQL を起動します。

## データベース

```bash
make up      # PostgreSQL を起動し、初回はスキーマを流す
make down    # 停止してデータも消す
make sqlc    # queries.sql からコードを生成する
```

接続先は `postgres://dev:dev@localhost:5432/trainingbooking` です。ポートを変えるときは環境変数 `POSTGRES_PORT` を指定します。

## 動かし方

```bash
make up
go run ./cmd/api
```

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
```

`201` と `"status":"pending"` が返ります。同じ時間帯にもう一度予約すると `409` が返ります。一覧は `GET /members/{id}/bookings` で確かめます。接続先は環境変数 `DATABASE_URL`、待ち受けるアドレスは `ADDR` で変えられます。

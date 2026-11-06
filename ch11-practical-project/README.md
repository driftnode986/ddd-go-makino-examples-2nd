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

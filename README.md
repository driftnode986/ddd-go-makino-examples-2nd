# Go 言語ドメイン駆動設計 サンプルコード

トレーニング予約システムを題材に、値オブジェクトから複数のコンテキストの統合までを Go で実装するサンプルコードです。

## 必要なもの

- Go 1.27 以降
- Docker（PostgreSQL を使うテストと起動に使います）
- golangci-lint v2

## 構成

- `ch11-practical-project/` 完成形です

章ごとのディレクトリは順次追加します。各ディレクトリが独立した Go module です。module 名はすべて `example.com/trainingbooking` です。

## 確かめ方

各ディレクトリで次を実行します。

```bash
go test ./...
go test -tags=integration ./...
golangci-lint run ./...
```

## ライセンス

MIT License です。`LICENSE` を参照してください。

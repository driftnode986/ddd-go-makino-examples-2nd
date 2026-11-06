# Go 言語ドメイン駆動設計 サンプルコード

トレーニング予約システムを題材に、値オブジェクトから複数のコンテキストの統合までを Go で実装するサンプルコードです。

第3章から第11章までは同じコードを育てます。各章のディレクトリは、直前の章を複製してその章の主題の差分だけを足したものです。章をまたいで比べるときは `diff -r ch03-value-object ch04-entity` のように実行します。

## 必要なもの

- Go 1.27 以降
- Docker（第7章以降の統合テストと起動に使います）
- golangci-lint v2

## 章の一覧

各ディレクトリが独立した Go module です。module 名はすべて `example.com/trainingbooking` で、パスも章をまたいで同じです。詳しい動かし方は各ディレクトリの README にあります。

- `ch02-go-idioms/` 第2章。Go の慣習を確かめる短い例
- `ch03-value-object/` 第3章。`Email`・`Money`・`TimeSlot`・`ID[T]`
- `ch04-entity/` 第4章。`Booking` の状態遷移と24時間ルール、`Clock`
- `ch05-domain-service/` 第5章。割引の計算と `Coupon`、時間帯の重なりの `DetectOverlap`
- `ch06-aggregate-factory/` 第6章。`BookingOption` と `Total`、再構築
- `ch07-repository/` 第7章。PostgreSQL と in-memory のリポジトリ、契約テスト
- `ch08-application-service/` 第8章。ユースケース、Query、Tx、エラーの変換。重なりの確認とクーポンの割引を予約につなぎます
- `ch09-layered-architecture/` 第9章。HTTP と `main`。`go run` で予約が `pending` で返ります
- `ch10-bounded-context/` 第10章。Catalog とイベントの配信。予約が `confirmed` まで進みます
- `ch11-practical-project/` 第11章。Identity を足した完成形

## 確かめ方

各ディレクトリで次を実行します。

```bash
go test ./...
golangci-lint run ./...
```

第7章以降は、Docker を動かして統合テストも実行します。

```bash
go test -tags=integration ./...
```

第9章以降の起動の手順は各章の README にあります。PostgreSQL のポートは章をまたいで同じ `5432` なので、別の章に移るときは前の章のディレクトリで `make down` を実行します。

## ライセンス

MIT License です。`LICENSE` を参照してください。

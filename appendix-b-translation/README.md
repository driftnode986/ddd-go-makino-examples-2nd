# 付録B Java / Kotlin DDD から Go DDD への対訳

Kotlin と Spring の書き方が Go で何に当たるかを、最小の予約の例で確かめます。第3章から育てるコードとは別に、対訳に要る分だけを1つのパッケージに置きました。

## 対応

- `data class` と `init` → `booking/money.go` の struct、コンストラクタ関数、`Equal`
- `@Entity`・`@Id` → `booking/booking.go` の struct。アノテーションと永続化の情報を持たない
- `RuntimeException` → `booking/booking.go` の sentinel と `error` の戻り値
- `@DomainEvents` → `booking/booking.go` の `PullEvents`
- `JpaRepository`・`@Version` → `booking/repository.go` のインターフェースと版の判定
- `@Service`・`@Transactional` → `booking/cancel_booking.go` の struct、コンストラクタ、`Transactor`
- `@MockBean` → `booking/booking_test.go` の小さい型と in-memory のリポジトリ

本番の形（Tx の必須化、PostgreSQL の実装、outbox）は第7〜10章のディレクトリにあります。

## 動かし方

```bash
cd appendix-b-translation
go test ./...
```

## 期待する出力

次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/booking
```

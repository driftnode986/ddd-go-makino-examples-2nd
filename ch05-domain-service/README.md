# 第5章 ドメインサービス

`Coupon` と `Money` のどちらのものでもない割引の計算を、状態を持たない関数にします。

## できるようになったこと

- `NewCoupon` で1〜100パーセントの割引券を作れます
- `CalculateDiscount` で割引額を計算できます。端数は切り捨てます
- ゼロ値の `Coupon` と、計算のあふれをエラーにできます

## 第4章からの変更

- 追加: `internal/booking/domain/discount.go`・`discount_test.go`
- 変更: `internal/booking/domain/errors.go` に `ErrInvalidCoupon` を追加

## 動かし方

```bash
cd ch05-domain-service
go test ./...
```

## 期待する出力

次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
```

端数のテストだけを見るときは次を実行します。

```bash
go test -v -run TestCalculateDiscount_TruncatesFraction ./internal/booking/domain/
```

計算した割引額を `Booking` に適用する操作は、第6章で足します。

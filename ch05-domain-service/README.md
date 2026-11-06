# 第5章 ドメインサービス

`Coupon` と `Money` のどちらのものでもない割引の計算と、1つの `Booking` では判定できない時間帯の重なりを、状態を持たない関数にします。

## できるようになったこと

- `NewCoupon` で1〜100パーセントの割引券を作れます。`IsZero` と `Equal` を持ちます
- `CalculateDiscount` で割引額を計算できます。端数は切り捨てます
- ゼロ値の `Coupon` と、計算のあふれをエラーにできます
- `DetectOverlap` で時間帯が既存の時間帯と重なるかを判定できます。重なれば `ErrOverlap` を返します

## 第4章からの変更

- 追加: `internal/booking/domain/` の `discount.go`・`discount_test.go`・`overlap.go`・`overlap_test.go`
- 変更: `internal/booking/domain/errors.go` に `ErrInvalidCoupon` と `ErrOverlap` を追加
- `steps/` 本文で失敗を動かすためのコード（この章だけにあります）
- `out/` 本に載せた実行結果。`bash out/regen.sh` で作り直せます

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

計算した割引額を `Booking` に適用する操作は、第6章で足します。どちらのドメインサービスも、第8章の `PlaceBooking` から呼びます。

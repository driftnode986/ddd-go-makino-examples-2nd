# 第6章 集約とファクトリ

`BookingOption` を足して、`Booking` を集約にします。

## できるようになったこと

- シューズのレンタルなどの明細を、`AddOption`・`RemoveOption` だけで変更できます。上限は5件です
- `Options` はコピーを返すので、受け取った側が書き換えても集約は変わりません
- `Total` で料金と明細の合計から割引を引いた額を読めます。`ApplyDiscount` で第5章の割引額を適用できます
- `State` と `ReconstructBooking` で集約の中身を取り出し、組み立て直せます。過去の開始は許し、未知の状態や負の金額は拒否します

## 第5章からの変更

- 追加: `internal/booking/domain/` の `booking_option.go`・`booking_state.go`・`reconstruct_test.go`
- 変更: `booking.go`（明細・合計・割引）、`booking_test.go`、`errors.go`
- 依存: テストで `github.com/google/go-cmp` を使います

## 動かし方

```bash
cd ch06-aggregate-factory
go test ./...
```

## 期待する出力

次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
```

再構築のテストだけを見るときは次を実行します。

```bash
go test -v -run TestReconstruct ./internal/booking/domain/
```

集約はまだプロセスを終えると消えます。第7章で保存します。

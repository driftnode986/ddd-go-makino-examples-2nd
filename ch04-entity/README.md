# 第4章 エンティティと不変条件保護

第3章の値を使って、状態を持つ `Booking` を作ります。

## できるようになったこと

- `Booking` を `NewBooking` で `pending` として作り、ゼロ値の ID や過去の開始を拒否できます
- `Confirm`・`Reject`・`Cancel` だけで状態を変えられます。遷移できない状態からはエラーを返します
- キャンセルは `confirmed` のときだけ、開始の24時間前まで受け付けます
- `Clock` を受け取るので、期限のテストが実行する時刻に左右されません

## 第3章からの変更

- 追加: `internal/booking/domain/` の `booking.go`・`booking_test.go`・`clock.go`・`errors.go`
- `steps/` 本文で失敗を動かすためのコード（この章だけにあります）。`steps/testdata/` はわざとコンパイルできないコードで、`go build ./...` の対象になりません
- `out/` 本に載せた実行結果。`bash out/regen.sh` で作り直せます

## 動かし方

```bash
cd ch04-entity
go test ./...
```

## 期待する出力

次の行が `ok` で出ます。最後に実行時間が付きます。

```text
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
```

`steps/` の下のパッケージも同じように `ok` で出ます。

4つの状態と3つの操作の組み合わせは、次のテストで確かめます。

```bash
go test -v -run TestBooking_Transitions ./internal/booking/domain/
```

`--- PASS: TestBooking_Transitions` と、12個のサブテストの `PASS` が出ます。

料金は `NewBooking` で受け取って保持しますが、読み出す手段はまだありません。合計は第6章で足します。

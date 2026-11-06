# 第3章 値オブジェクト

文字列や整数をそのまま渡す代わりに、業務の値を型にします。このディレクトリから第11章まで同じコードを育てます。

## できるようになったこと

- `Email`・`Money`・`TimeSlot` で値を受け取り、不正な値をコンストラクタで拒否できます
- `ID[T]` で会員・レッスン・予約の ID を別の型にして、取り違えをコンパイルエラーにできます
- `var e Email` で作ったゼロ値を `IsZero` で見分けられます
- 同じ瞬間の時刻なら、どのタイムゾーンで渡しても営業時間の判定と `Equal` の結果が同じになります

## 主なファイル

- `internal/ids/id.go` `ID[T]` と3つの ID
- `internal/identity/domain/email.go` `Email`
- `internal/booking/domain/money.go` `Money`
- `internal/booking/domain/time_slot.go` `TimeSlot`

- `steps/` 本文で失敗を動かすためのコード（この章だけにあります）。`steps/testdata/` はわざとコンパイルできないコードで、`go build ./...` の対象になりません
- `out/` 本に載せた実行結果。`bash out/regen.sh` で作り直せます

## 動かし方

```bash
cd ch03-value-object
go test ./...
```

## 期待する出力

次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
```

`steps/` の下のパッケージも同じように `ok` で出ます。

タイムゾーンのテストだけを見るときは次を実行します。

```bash
go test -v -run TestTimeSlot_SameInstantAcrossZones ./internal/booking/domain/
```

`--- PASS: TestTimeSlot_SameInstantAcrossZones` と出ます。

この章の時点では、状態を持つものがまだありません。次の第4章で予約を足します。

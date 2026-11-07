# 第2章 Go の慣習と DDD の原則を擦り合わせる

判断の根拠になる Go の動きを、短い例で確かめます。第3章から育てる予約システムとは別のコードです。

## 例の一覧

- `smallinterface/` 小さいインターフェースはインターフェースの大きさの話で、構造体のメソッド数の話ではないこと。インターフェースを使う側に置き、依存をコンストラクタで受け取る形
- `zerovalue/` 公開フィールドと非公開フィールドの違い。非公開でもゼロ値は作れるので、受け取る側で止めること
- `domainerror/` sentinel と型つきエラー。`errors.Is` と `errors.AsType` での判定
- `receiver/` 値レシーバとポインタレシーバ。getter の名前
- `generics/` Go 1.27 のジェネリックメソッドと、インターフェースで差し替えられないこと
- `steps/` 失敗を見せるコード。`steps/testdata` はわざとコンパイルできないコードです
- `out/` 本に載せた実行結果。`bash out/regen.sh` で作り直せます

## 動かし方

```bash
cd ch02-go-idioms
go test ./...
```

## 期待する出力

次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/domainerror
ok  	example.com/trainingbooking/generics
ok  	example.com/trainingbooking/receiver
ok  	example.com/trainingbooking/smallinterface/reminder
ok  	example.com/trainingbooking/steps/setterstock/service
ok  	example.com/trainingbooking/zerovalue
```

テストの名前が確かめる内容を表します。一覧は `go test -v ./...` で見られます。

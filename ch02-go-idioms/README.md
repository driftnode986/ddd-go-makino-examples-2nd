# 第2章 Go の慣習と DDD の原則を擦り合わせる

判断の根拠になる Go の動きを、短い例で確かめます。第3章から育てる予約システムとは別のコードです。

## 例の一覧

- `smallinterface/` 小さい interface は interface の大きさの話で、struct のメソッド数の話ではないこと。依存をコンストラクタで受け取る形
- `zerovalue/` 公開フィールドと非公開フィールドの違い。非公開でもゼロ値は作れるので、受け取る側で止めること
- `domainerror/` sentinel と型つきエラー。`errors.Is` と `errors.AsType` での判定
- `receiver/` 値レシーバとポインタレシーバ。値で渡すとコピーにメソッドが効くこと。getter の名前
- `dto/` 非公開フィールドだけの struct を Encode すると `{}` になること。DTO を分ける形
- `generics/` Go 1.27 のジェネリックメソッドと、interface の実装に数えられないこと

## 動かし方

```bash
cd ch02-go-idioms
go test ./...
```

## 期待する出力

次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/domainerror
ok  	example.com/trainingbooking/dto
ok  	example.com/trainingbooking/generics
ok  	example.com/trainingbooking/receiver
ok  	example.com/trainingbooking/smallinterface
ok  	example.com/trainingbooking/zerovalue
```

テストの名前が確かめる内容を表します。一覧は `go test -v ./...` で見られます。

# 第7章 リポジトリと永続化

`Booking` 集約を PostgreSQL と in-memory に保存します。1本の契約テストを両方に流します。

## できるようになったこと

- `Add`・`Save`・`FindByID` で集約単位に保存し、読み出せます。親と明細は1つの SQL で読みます
- 読み込んだときの版で更新を判定し、2者の更新の競合を `ErrVersionConflict` で返します
- 同じ会員の重なる予約を PostgreSQL の排他制約で止め、`ErrOverlap` で返します
- `Add` と `Save` は ctx の Tx に参加します。Tx が無ければ `ErrNoTransaction` を返します

## 第6章からの変更

- 追加: `internal/booking/domain/repository.go`、`internal/booking/domain/repositorytest/`（契約テスト）、`internal/booking/infrastructure/`、`internal/platform/txn/`、`internal/platform/dbtest/`、`compose.yaml`、`Makefile`、`sqlc.yaml`
- 変更: `booking.go` と `booking_state.go` に版を追加、`errors.go` にリポジトリのエラーを追加
- 本文で失敗を動かすコードは `steps/` に、実行結果は `out/` にあります。`bash out/regen.sh` で作り直せます（Docker が要ります）

## 動かし方

統合テストは Docker で PostgreSQL を起動します。Docker を動かしてから実行します。

```bash
cd ch07-repository
go test ./...
go test -tags=integration ./...
```

開発用の PostgreSQL を起動してテーブルを確かめるときは次を実行します。

```bash
make up
docker compose exec postgres psql -U dev -d trainingbooking -c '\dt'
make down
```

PostgreSQL は `127.0.0.1:5432` で待ち受けます。別の章で起動したままのときは、その章のディレクトリで `make down` を実行してから起動します。ポートを変えるときは環境変数 `POSTGRES_PORT` を指定します。

`queries.sql` を変えたら `make sqlc` でコードを生成します。

## 期待する出力

`go test -tags=integration ./...` は次の行が `ok` で出ます。最後に実行時間が付きます。テストの無いパッケージは `[no test files]` と出ます。

```text
ok  	example.com/trainingbooking/internal/booking/domain
ok  	example.com/trainingbooking/internal/booking/infrastructure/memory
ok  	example.com/trainingbooking/internal/booking/infrastructure/postgres
ok  	example.com/trainingbooking/internal/identity/domain
ok  	example.com/trainingbooking/internal/ids
ok  	example.com/trainingbooking/internal/platform/txn/memtx
```

`\dt` には `bookings` と `booking_options` の2つのテーブルが出ます。

Tx を誰が始めるのかはまだ決めていません。第8章でユースケースが受け持ちます。

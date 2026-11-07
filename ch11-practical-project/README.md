# 完成形（第11章）

3つのコンテキストを1つの module に置く予約システムです。Booking が予約、Catalog がレッスンと定員、Identity が会員を受け持ちます。

## 第10章からの変更

Identity を足しました。`Member` 集約と `contract`、Booking 側の腐敗防止層（`internal/booking/infrastructure/acl/members.go`）で、存在しない会員の予約を `400` で拒否します。初期データは SQL をやめ、`cmd/seed` が `NewMember` と `NewLesson` の検証を通して入れます。

## 確かめ方

```bash
go test ./...
go test -tags=integration ./...
golangci-lint run ./...
```

`-tags=integration` のテストは Docker で PostgreSQL を起動します。

## 動かし方

Docker と `jq` が要ります。

```bash
make up      # PostgreSQL を起動し、初回はスキーマを流す
make seed    # 会員とレッスンの初期データを入れる
make run     # API と配信処理を1つのプロセスで起動する
```

止めるときは `make run` のターミナルで Ctrl+C を押します。処理中のリクエストを待ち、配信処理を止めてから終わります。止まったことはログの `shutting down` とプロセスの終了で確かめます（手元では `go run` の終了コードが 1 になり、`make` がエラーを表示しました）。ビルドしたものに SIGTERM を送るときの動きは第9章と第10章を参照してください（`go run` は SIGTERM をサーバーに渡しません）。

接続先は `postgres://dev:dev@localhost:5432/trainingbooking` です。別の章で PostgreSQL を起動したままのときは、その章のディレクトリで `make down` を実行してから起動します。ポートを変えるときは環境変数 `POSTGRES_PORT` と `DATABASE_URL` を指定します。待ち受けるアドレスの既定は `localhost:8080` です。別のプロセスが使っているときは `ADDR=localhost:18080 make run` のように変え、手順を実行するターミナルでも `export API=localhost:18080` を `scripts/try.sh` の前に実行します。先に `lsof -i :8080` で空いていることを確かめます。使われていると、`curl` が別のサーバーに届きます。

初期データのレッスンの日時は、初めて `make seed` を実行した日の30日後です。もう一度実行しても日時は変わりません。日が過ぎて予約できなくなったら `make reset` で作り直します（予約も消えます）。

- `0196f5a0-0000-7000-8001-000000000001` 朝のヨガ。定員10、10時から。オプションはシューズ（300円）とタオル（100円）
- `0196f5a0-0000-7000-8001-000000000002` 少人数ピラティス。定員1、14時から
- `0196f5a0-0000-7000-8001-000000000003` 朝の筋力トレーニング。定員10、10時から

会員は `0196f5a0-0000-7000-8000-000000000001`（花子）と `0196f5a0-0000-7000-8000-000000000002`（太郎）です。

## 端から端まで動かす

別のターミナルで、変数と関数を読み込みます。手順は上から順に実行します。やり直すときは `make reset` で初期データに戻し、最初の花子の予約から進めます（同じ予約を送ると、時間帯の重なりで `409` になります）。

席の確保と確定は配信処理が後から行います。予約の応答は `pending` なので、次へ進む前に `wait_status` で状態が変わるのを待ちます。

```bash
source scripts/try.sh
```

### 予約して確定する

花子が朝のヨガをシューズ2足つきで予約します。時間帯と料金はレッスンから取るので、指定するのは会員とレッスンとオプションです。

```bash
post "$HANAKO" "$YOGA" '"options":[{"name":"シューズ","quantity":2}]'
```

本文の `"status":"pending"` と `201` が表示されます。少し待って一覧を読むと `confirmed` になっています。合計は3600円で、シューズの明細が `options` に出ます。オプションは予約のときに付けます。後から足したり外したりする API はありません。オプションが無い予約の `options` は `[]` です。

```bash
sleep 1
curl -s "$API/members/$HANAKO/bookings" | jq -c '.bookings[]'
```

### クーポンで割り引く

太郎が同じレッスンをクーポンつきで予約します。使えるコードは `WELCOME10`（10%）で、割引率は受け取りません。料金とオプションの合計3300円の10%を引いて、合計は2970円です。`options` の `subtotal_yen` は割引前の300円です。

```bash
post "$TARO" "$YOGA" '"options":[{"name":"シューズ","quantity":1}],
  "coupon_code":"WELCOME10"'
sleep 1
curl -s "$API/members/$TARO/bookings" | jq -c '.bookings[]'
```

### 満席とキャンセル

定員1のレッスンで確かめます。どの速さで打っても同じ結果になるように、状態が変わるのを `wait_status` で待ってから次へ進みます。

```bash
FIRST=$(book "$HANAKO" "$PILATES")
wait_status "$HANAKO" "$FIRST"
SECOND=$(book "$TARO" "$PILATES")
wait_status "$TARO" "$SECOND"
cancel "$SECOND"
cancel "$FIRST"
THIRD=$(book "$TARO" "$PILATES")
wait_status "$TARO" "$THIRD"
```

花子は `confirmed`、太郎は `201` が返った後で `rejected` です。`rejected` の予約のキャンセルは `409` です。花子の予約をキャンセルすると席が戻り、太郎がもう一度予約すると `confirmed` になります。

`pending` の予約のキャンセルも `409` です（`TestCancel_PendingIsConflict`）。手順に入れていないのは、配信が短い時間で終わるので、手で打つと確定が先に済んでキャンセルが通るからです。

### 同期で拒否するもの

```bash
# 花子は朝のヨガと同じ時間帯を予約できない → 409
post "$HANAKO" "$STRENGTH"
# 存在しない会員 → 400
post 0196f5a0-0000-7000-8000-00000000ffff "$YOGA"
# 不明なクーポンのコード → 400
post "$TARO" "$STRENGTH" '"coupon_code":"UNKNOWN"'
```

レッスンに無いオプション、0以下の数量、同じ名前の重複、金額のあふれも `400` です。

## 配信が遅れたとき

`pending` が続く時間の許容は5分です。配信が失敗している間も止めずに再配信を続け、`GET /members/{id}/bookings` では `pending` のまま見えます。失敗は `WARN` でログに残し、5分を超えたら `ERROR` に上げます。原因は `outbox` テーブルの `attempts` と `last_error` で確かめられます。止まるのは同じ集約のイベントだけです。配信処理は失敗の少ないイベントから配るので、失敗が続く集約が1回に配る件数の上限（100件）を埋めても、ほかの集約の配信は続きます。

## やり直す

```bash
make reset   # データを消して、起動と初期データの投入をやり直す
make down    # 停止してデータも消す
```

## 生成

```bash
make sqlc    # queries.sql からコードを生成する
```

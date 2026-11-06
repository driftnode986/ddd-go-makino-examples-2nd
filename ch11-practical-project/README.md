# 完成形（第11章）

3つのコンテキストを1つの module に置く予約システムです。Booking が予約、Catalog がレッスンと定員、Identity が会員を受け持ちます。

## 第10章からの変更

Identity を足しました。`Member` 集約と `contract`、Booking 側の腐敗防止層（`internal/booking/infrastructure/acl/members.go`）で、存在しない会員の予約を `400` で拒否します。初期データに会員が増えました。

## 確かめ方

```bash
go test ./...
go test -tags=integration ./...
golangci-lint run ./...
```

`-tags=integration` のテストは Docker で PostgreSQL を起動します。

## 動かし方

```bash
make up      # PostgreSQL を起動し、初回はスキーマを流す
make seed    # 会員とレッスンの初期データを入れる
make run     # API と配信処理を1つのプロセスで起動する
```

接続先は `postgres://dev:dev@localhost:5432/trainingbooking` です。別の章で PostgreSQL を起動したままのときは、その章のディレクトリで `make down` を実行してから起動します。ポートを変えるときは環境変数 `POSTGRES_PORT` と `DATABASE_URL` を指定します。待ち受けるアドレスは `ADDR` で変えられます。

初期データのレッスンの日時は、初めて `make seed` を実行した日の30日後です。もう一度流しても日時は変わりません。日が過ぎて予約できなくなったら `make reset` で作り直します（予約も消えます）。

- `0196f5a0-0000-7000-8001-000000000001` 朝のヨガ。定員10、10時から。オプションはシューズ（300円）とタオル（100円）
- `0196f5a0-0000-7000-8001-000000000002` 少人数ピラティス。定員1、14時から
- `0196f5a0-0000-7000-8001-000000000003` 朝の筋力トレーニング。定員10、10時から

会員は `0196f5a0-0000-7000-8000-000000000001`（花子）と `0196f5a0-0000-7000-8000-000000000002`（太郎）です。

## 予約して確定する

別のターミナルから予約します。時間帯と料金はレッスンから取るので、指定するのは会員とレッスンとオプションです。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000001",
    "lesson_id": "0196f5a0-0000-7000-8001-000000000001",
    "options": [{"name": "シューズ", "quantity": 2}]
  }'
```

`201` と `"status":"pending"` が返ります。席の確保は配信処理が行うので、少し待って一覧を読むと `confirmed` になっています。合計は3600円で、シューズの明細が `options` に出ます。`id` と日時は実行ごとに変わります。

```bash
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000001/bookings
```

```json
{"bookings":[{"id":"…","lesson_id":"0196f5a0-0000-7000-8001-000000000001","start":"…","end":"…","status":"confirmed","total_yen":3600,"options":[{"name":"シューズ","quantity":2,"subtotal_yen":600}]}]}
```

オプションは予約のときに付けます。後から足したり外したりする API はありません。オプションが無い予約の `options` は `[]` です。

## クーポンで割り引く

太郎が朝のヨガをクーポンつきで予約します。使えるコードは `WELCOME10`（10%）で、割引率は受け取りません。

```bash
curl -i -X POST localhost:8080/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "member_id": "0196f5a0-0000-7000-8000-000000000002",
    "lesson_id": "0196f5a0-0000-7000-8001-000000000001",
    "options": [{"name": "シューズ", "quantity": 1}],
    "coupon_code": "WELCOME10"
  }'
curl localhost:8080/members/0196f5a0-0000-7000-8000-000000000002/bookings
```

少し待って一覧を読むと `confirmed` になっています。料金とオプションの合計3300円の10%を引いて、合計は2970円です。`options` の `subtotal_yen` は割引前の300円です。

## 満席・キャンセル・入力の誤り

定員1のレッスンで、満席の拒否とキャンセルで席が戻るところを確かめます。手順には `jq` を使います。まず、会員とレッスンの ID と、次の2つの関数を同じターミナルで定義します。

```bash
# 予約して id を表示する（$1=会員 $2=レッスン）
book() {
  curl -s -X POST localhost:8080/bookings \
    -H 'Content-Type: application/json' \
    -d "{\"member_id\":\"$1\",\"lesson_id\":\"$2\"}" | jq -r .id
}
# pending でなくなるまで待ち、状態を表示する（$1=会員 $2=予約）
wait_status() {
  for _ in $(seq 50); do
    state=$(curl -s "localhost:8080/members/$1/bookings" |
      jq -r --arg id "$2" \
        '.bookings[] | select(.id == $id) | .status')
    [ "$state" != pending ] && break
    sleep 0.2
  done
  echo "$state"
}

HANAKO=0196f5a0-0000-7000-8000-000000000001
TARO=0196f5a0-0000-7000-8000-000000000002
PILATES=0196f5a0-0000-7000-8001-000000000002
```

花子が定員1のレッスンを予約します。キャンセルできるのは `confirmed` だけなので（`pending` のキャンセルは `409`）、状態を確かめてから次へ進みます。

```bash
FIRST=$(book "$HANAKO" "$PILATES")
wait_status "$HANAKO" "$FIRST"
```

`confirmed` と表示されます。同じレッスンを太郎が予約すると、`201` が返った後で `rejected` になります。

```bash
SECOND=$(book "$TARO" "$PILATES")
wait_status "$TARO" "$SECOND"
```

`rejected` と表示されます。花子の予約をキャンセルすると席が戻り、太郎がもう一度予約すると `confirmed` になります。

```bash
curl -s -X POST "localhost:8080/bookings/$FIRST/cancel"
echo
THIRD=$(book "$TARO" "$PILATES")
wait_status "$TARO" "$THIRD"
```

キャンセルの応答は `"status":"cancelled"` で、最後に `confirmed` と表示されます。

次のものは同期で拒否します。

- 花子が同じ時間帯の朝の筋力トレーニング（`...8001-000000000003`）を予約すると `409`
- 存在しない会員（例: `0196f5a0-0000-7000-8000-00000000ffff`）で予約すると `400`
- 不明なクーポンのコード（例: `"coupon_code": "UNKNOWN"`）で予約すると `400`
- レッスンに無いオプション、0以下の数量、同じ名前の重複、金額のあふれは `400`

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

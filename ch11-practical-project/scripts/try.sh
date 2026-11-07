# 手順で使う変数と関数である。source scripts/try.sh で読み込む。
HANAKO=0196f5a0-0000-7000-8000-000000000001
TARO=0196f5a0-0000-7000-8000-000000000002
YOGA=0196f5a0-0000-7000-8001-000000000001
PILATES=0196f5a0-0000-7000-8001-000000000002
STRENGTH=0196f5a0-0000-7000-8001-000000000003
API=${API:-localhost:8080}

# 予約を送り、応答の本文とステータスを表示する。
# $1=会員 $2=レッスン $3=足す項目（任意）
post() {
  curl -s -w '%{http_code}\n' -X POST "$API/bookings" \
    -H 'Content-Type: application/json' \
    -d "{\"member_id\":\"$1\",\"lesson_id\":\"$2\"${3:+,$3}}"
}

# 予約を送り、受け付けた予約の id だけを表示する。
book() {
  post "$@" | head -n 1 | jq -r .id
}

# #region wait-status
# pending でなくなるまで最大10秒待ち、状態を表示する。
# $1=会員 $2=予約
wait_status() {
  for i in $(seq 50); do
    state=$(curl -s "$API/members/$1/bookings" |
      jq -r --arg id "$2" \
        '.bookings[] | select(.id == $id) | .status')
    [ "$state" != pending ] && break
    sleep 0.2
  done
  echo "$state"
}
# #endregion

# キャンセルを送り、応答の本文とステータスを表示する。$1=予約
cancel() {
  curl -s -w '%{http_code}\n' -X POST "$API/bookings/$1/cancel"
}

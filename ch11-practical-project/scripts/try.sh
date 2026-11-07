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
# 受け付けられなければ、応答を標準エラーに出して失敗する。
book() {
  body=$(post "$@" | head -n 1)
  echo "$body" | jq -er '.id // empty' || {
    echo "予約できません: $body" >&2
    return 1
  }
}

# #region wait-status
# pending でなくなるまで 0.2 秒おきに50回確かめ、状態を表示する。
# 取得の失敗、予約が無いとき、pending のままのときは失敗を返す。
# $1=会員 $2=予約
wait_status() {
  for i in $(seq 50); do
    body=$(curl -fsS --max-time 2 "$API/members/$1/bookings") ||
      return 1
    state=$(echo "$body" | jq -er --arg id "$2" \
      '.bookings[] | select(.id == $id) | .status') || {
      echo "予約がありません: $2" >&2
      return 1
    }
    if [ "$state" != pending ]; then
      echo "$state"
      return 0
    fi
    sleep 0.2
  done
  echo "pending のままです: $2" >&2
  return 1
}
# #endregion

# キャンセルを送り、応答の本文とステータスを表示する。$1=予約
cancel() {
  curl -s -w '%{http_code}\n' -X POST "$API/bookings/$1/cancel"
}

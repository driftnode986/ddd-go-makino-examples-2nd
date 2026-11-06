-- name: InsertBooking :exec
INSERT INTO bookings (
    id, member_id, lesson_id, starts_at, ends_at,
    fee_yen, discount_yen, status, version
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: InsertBookingOption :exec
INSERT INTO booking_options (
    booking_id, position, name, unit_price_yen, quantity
) VALUES ($1, $2, $3, $4, $5);

-- name: UpdateBooking :execrows
UPDATE bookings
SET status = $3, discount_yen = $4, version = version + 1
WHERE id = $1 AND version = $2;

-- name: DeleteBookingOptions :exec
DELETE FROM booking_options WHERE booking_id = $1;

-- name: BookingExists :one
SELECT EXISTS (SELECT 1 FROM bookings WHERE id = $1);

-- 親と明細を1つの SQL で読み、同じ時点の組み合わせを得る。
-- name: FindBookingWithOptions :many
SELECT
    b.id, b.member_id, b.lesson_id, b.starts_at, b.ends_at,
    b.fee_yen, b.discount_yen, b.status, b.version,
    o.name AS option_name,
    o.unit_price_yen AS option_unit_price_yen,
    o.quantity AS option_quantity
FROM bookings b
LEFT JOIN booking_options o ON o.booking_id = b.id
WHERE b.id = $1
ORDER BY o.position;

-- 同じ会員の書き込みを Tx の終わりまで順番に並べる。
-- name: LockMember :exec
SELECT pg_advisory_xact_lock(hashtextextended(@member_id::text, 0));

-- 一覧は集約を組み立てず、合計と小計を SQL で求める。
-- 親と明細を1つの SQL で読むので、予約の行は明細の数だけ並ぶ。
-- name: ListBookingsByMember :many
SELECT
    b.id, b.lesson_id, b.starts_at, b.ends_at, b.status,
    (b.fee_yen - b.discount_yen
        + COALESCE(SUM(o.unit_price_yen * o.quantity)
            OVER (PARTITION BY b.id), 0))::bigint
        AS total_yen,
    o.name AS option_name,
    o.quantity AS option_quantity,
    COALESCE(o.unit_price_yen * o.quantity, 0)::bigint
        AS option_subtotal_yen
FROM bookings b
LEFT JOIN booking_options o ON o.booking_id = b.id
WHERE b.member_id = $1
ORDER BY b.starts_at, b.id, o.position;

-- 排他制約と同じ条件で、時間帯を占める予約だけを読む。
-- name: ListActiveSlotsByMember :many
SELECT starts_at, ends_at
FROM bookings
WHERE member_id = $1 AND status IN ('pending', 'confirmed')
ORDER BY starts_at;

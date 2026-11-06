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

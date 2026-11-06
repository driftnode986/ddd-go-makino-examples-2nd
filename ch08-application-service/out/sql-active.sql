-- name: ListActiveSlotsByMember :many
SELECT starts_at, ends_at
FROM bookings
WHERE member_id = $1 AND status IN ('pending', 'confirmed')
ORDER BY starts_at;

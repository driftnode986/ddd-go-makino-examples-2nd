-- name: UpdateBooking :execrows
UPDATE bookings
SET status = $3, discount_yen = $4, version = version + 1
WHERE id = $1 AND version = $2;

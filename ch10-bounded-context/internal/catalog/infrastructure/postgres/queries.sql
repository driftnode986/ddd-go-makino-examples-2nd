-- name: InsertLesson :exec
INSERT INTO lessons (
    id, title, starts_at, ends_at, fee_yen, capacity, version
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: InsertLessonOption :exec
INSERT INTO lesson_options (
    lesson_id, position, name, unit_price_yen
) VALUES ($1, $2, $3, $4);

-- name: InsertLessonSeat :exec
INSERT INTO lesson_seats (lesson_id, position, booking_id)
VALUES ($1, $2, $3);

-- name: BumpLessonVersion :execrows
UPDATE lessons SET version = version + 1
WHERE id = $1 AND version = $2;

-- name: DeleteLessonSeats :exec
DELETE FROM lesson_seats WHERE lesson_id = $1;

-- name: LessonExists :one
SELECT EXISTS (SELECT 1 FROM lessons WHERE id = $1);

-- 親と明細を1つの SQL で読み、同じ時点の組み合わせを得る。
-- name: FindLesson :one
SELECT
    l.id, l.title, l.starts_at, l.ends_at,
    l.fee_yen, l.capacity, l.version,
    ARRAY(
        SELECT o.name FROM lesson_options o
        WHERE o.lesson_id = l.id ORDER BY o.position
    )::text[] AS option_names,
    ARRAY(
        SELECT o.unit_price_yen FROM lesson_options o
        WHERE o.lesson_id = l.id ORDER BY o.position
    )::bigint[] AS option_prices,
    ARRAY(
        SELECT s.booking_id::text FROM lesson_seats s
        WHERE s.lesson_id = l.id ORDER BY s.position
    )::text[] AS seats
FROM lessons l
WHERE l.id = $1;

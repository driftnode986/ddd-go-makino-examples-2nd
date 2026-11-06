-- 同じ集約の書き込みを Tx の終わりまで順番に並べる。連番の順と
-- コミットの順がそろうので、先の連番が後から見える事態が起きない。
-- name: LockAggregate :exec
SELECT pg_advisory_xact_lock(hashtextextended(@aggregate_key::text, 1));

-- name: InsertEvent :exec
INSERT INTO outbox (
    event_id, aggregate_type, aggregate_id, sequence,
    event_type, payload, occurred_at
) VALUES (
    @event_id, @aggregate_type, @aggregate_id,
    (SELECT COALESCE(MAX(o.sequence), 0) + 1 FROM outbox o
     WHERE o.aggregate_type = @aggregate_type
       AND o.aggregate_id = @aggregate_id),
    @event_type, @payload, @occurred_at
);

-- 集約ごとに未配信の先頭だけを返す。先行が残るあいだ、後のものは
-- 候補に出ない。
-- name: ListDeliverable :many
SELECT h.* FROM (
    SELECT DISTINCT ON (aggregate_type, aggregate_id)
        id, event_id, aggregate_type, aggregate_id, sequence,
        event_type, payload, occurred_at, created_at, attempts
    FROM outbox
    WHERE published_at IS NULL
    ORDER BY aggregate_type, aggregate_id, sequence
) h
ORDER BY h.id
LIMIT $1;

-- name: MarkPublished :exec
UPDATE outbox SET published_at = now()
WHERE id = $1 AND published_at IS NULL;

-- name: RecordFailure :exec
UPDATE outbox SET attempts = attempts + 1, last_error = @last_error::text
WHERE id = @id;

-- name: MarkProcessed :execrows
INSERT INTO processed_events (subscriber, event_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

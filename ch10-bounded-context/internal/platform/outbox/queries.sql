-- 同じ集約の書き込みを Tx の終わりまで順番に並べる。連番は MAX + 1 で
-- 決めるので、ロックが無いと2つの Tx が同じ連番を計算し、後の Tx が
-- 一意制約で失敗する。
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
-- 候補に出ない。失敗の少ない順に返し、失敗した集約を次の周回で
-- 後ろへ回す。失敗し続ける集約が件数の上限を埋めないためである。
-- name: ListDeliverable :many
SELECT h.* FROM (
    SELECT DISTINCT ON (aggregate_type, aggregate_id)
        id, event_id, aggregate_type, aggregate_id, sequence,
        event_type, payload, occurred_at, created_at, attempts
    FROM outbox
    WHERE published_at IS NULL
    ORDER BY aggregate_type, aggregate_id, sequence
) h
ORDER BY h.attempts, h.id
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

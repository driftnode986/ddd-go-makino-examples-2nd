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

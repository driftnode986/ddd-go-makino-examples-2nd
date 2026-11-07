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

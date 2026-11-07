INSERT INTO processed_events (subscriber, event_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

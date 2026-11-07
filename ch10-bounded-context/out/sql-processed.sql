CREATE TABLE processed_events (
    subscriber   text NOT NULL,
    event_id     uuid NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (subscriber, event_id)
);

-- 集約と同じ Tx で書くイベントである。集約ごとに連番を持つ。
CREATE TABLE outbox (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id       uuid NOT NULL UNIQUE,
    aggregate_type text NOT NULL,
    aggregate_id   uuid NOT NULL,
    sequence       bigint NOT NULL CHECK (sequence > 0),
    event_type     text NOT NULL,
    payload        jsonb NOT NULL,
    occurred_at    timestamptz NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    published_at   timestamptz,
    attempts       integer NOT NULL DEFAULT 0,
    last_error     text,
    UNIQUE (aggregate_type, aggregate_id, sequence)
);

CREATE INDEX outbox_undelivered
    ON outbox (aggregate_type, aggregate_id, sequence)
    WHERE published_at IS NULL;

-- 受け手が処理したイベントである。集約の保存と同じ Tx で書く。
CREATE TABLE processed_events (
    subscriber   text NOT NULL,
    event_id     uuid NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (subscriber, event_id)
);

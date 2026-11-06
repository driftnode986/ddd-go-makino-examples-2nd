-- Catalog のテーブルである。予約との外部キーは張らない。
-- コンテキストをまたぐ整合はイベントで取る。
CREATE TABLE lessons (
    id        uuid PRIMARY KEY,
    title     text NOT NULL CHECK (title <> ''),
    starts_at timestamptz NOT NULL,
    ends_at   timestamptz NOT NULL,
    fee_yen   bigint NOT NULL CHECK (fee_yen > 0),
    capacity  integer NOT NULL CHECK (capacity > 0),
    version   bigint NOT NULL CHECK (version >= 0),
    CHECK (starts_at < ends_at)
);

CREATE TABLE lesson_options (
    lesson_id      uuid NOT NULL REFERENCES lessons (id),
    position       integer NOT NULL,
    name           text NOT NULL,
    unit_price_yen bigint NOT NULL CHECK (unit_price_yen >= 0),
    PRIMARY KEY (lesson_id, position),
    UNIQUE (lesson_id, name)
);

CREATE TABLE lesson_seats (
    lesson_id  uuid NOT NULL REFERENCES lessons (id),
    position   integer NOT NULL,
    booking_id uuid NOT NULL,
    PRIMARY KEY (lesson_id, position),
    UNIQUE (lesson_id, booking_id)
);

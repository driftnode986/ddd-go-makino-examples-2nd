-- 会員と時間帯の排他制約で uuid を gist に載せるために使う。
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE bookings (
    id           uuid PRIMARY KEY,
    member_id    uuid NOT NULL,
    lesson_id    uuid NOT NULL,
    starts_at    timestamptz NOT NULL,
    ends_at      timestamptz NOT NULL,
    fee_yen      bigint NOT NULL CHECK (fee_yen > 0),
    discount_yen bigint NOT NULL CHECK (discount_yen >= 0),
    status       text NOT NULL CHECK (status IN
        ('pending', 'confirmed', 'rejected', 'cancelled')),
    version      bigint NOT NULL CHECK (version >= 0),
    CHECK (starts_at < ends_at),
    -- 同じ会員の有効な予約は時間帯を重ねられない。
    -- tstzrange の既定は半開区間なので、接するだけなら通る。
    CONSTRAINT bookings_no_overlap EXCLUDE USING gist (
        member_id WITH =,
        tstzrange(starts_at, ends_at) WITH &&
    ) WHERE (status IN ('pending', 'confirmed'))
);

CREATE TABLE booking_options (
    booking_id     uuid NOT NULL REFERENCES bookings (id),
    position       integer NOT NULL,
    name           text NOT NULL,
    unit_price_yen bigint NOT NULL CHECK (unit_price_yen >= 0),
    quantity       bigint NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (booking_id, position),
    UNIQUE (booking_id, name)
);

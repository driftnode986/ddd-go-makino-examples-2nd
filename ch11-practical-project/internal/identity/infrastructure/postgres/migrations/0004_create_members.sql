-- Identity のテーブルである。予約との外部キーは張らない。
CREATE TABLE members (
    id    uuid PRIMARY KEY,
    email text NOT NULL UNIQUE,
    name  text NOT NULL CHECK (name <> '')
);

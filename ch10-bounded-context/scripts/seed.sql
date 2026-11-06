-- 動作確認用の初期データである。何度流しても既存の行は変えない。
-- レッスンの日時は、初めて流した日の30日後の日本時間で作る。

WITH day AS (
    SELECT date_trunc('day', now() AT TIME ZONE 'Asia/Tokyo')
        + interval '30 days' AS d
)
INSERT INTO lessons (
    id, title, starts_at, ends_at, fee_yen, capacity, version
)
SELECT l.id::uuid, l.title,
    (d + l.starts) AT TIME ZONE 'Asia/Tokyo',
    (d + l.starts + interval '1 hour') AT TIME ZONE 'Asia/Tokyo',
    l.fee_yen, l.capacity, 0
FROM day, (VALUES
    -- 席に余裕があるレッスン
    ('0196f5a0-0000-7000-8001-000000000001', '朝のヨガ',
     interval '10 hours', 3000::bigint, 10),
    -- 定員1のレッスン。満席の拒否を確かめる
    ('0196f5a0-0000-7000-8001-000000000002', '少人数ピラティス',
     interval '14 hours', 5000::bigint, 1),
    -- 朝のヨガと同じ時間帯のレッスン。重なる予約を確かめる
    ('0196f5a0-0000-7000-8001-000000000003', '朝の筋力トレーニング',
     interval '10 hours', 3000::bigint, 10)
) AS l (id, title, starts, fee_yen, capacity)
ON CONFLICT DO NOTHING;

INSERT INTO lesson_options (
    lesson_id, position, name, unit_price_yen
) VALUES
    ('0196f5a0-0000-7000-8001-000000000001', 0, 'シューズ', 300),
    ('0196f5a0-0000-7000-8001-000000000001', 1, 'タオル', 100)
ON CONFLICT DO NOTHING;

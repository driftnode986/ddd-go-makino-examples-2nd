-- name: LockMember :exec
SELECT pg_advisory_xact_lock(hashtextextended(@member_id::text, 0));

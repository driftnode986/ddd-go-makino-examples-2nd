-- name: InsertMember :exec
INSERT INTO members (id, email, name) VALUES ($1, $2, $3);

-- name: FindMember :one
SELECT id, email, name FROM members WHERE id = $1;

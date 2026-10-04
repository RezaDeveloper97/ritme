-- name: RecentReadAudit :one
-- The newest `read` row of this actor on this section of the owner's data through this link since `since`: a
-- companion read within the coalescing window is not written again (a reload must not flood the owner's trail;
-- B-N4-08b, CMP-L1).
SELECT id FROM `companion_audit_logs`
WHERE owner_id = sqlc.arg(owner_id) AND actor_id = sqlc.arg(actor_id) AND companion_id = sqlc.arg(companion_id)
  AND section = sqlc.arg(section) AND action = 'read' AND created_at >= sqlc.arg(since)
ORDER BY id DESC
LIMIT 1;

-- name: ListOwnerAuditPage :many
-- One page of the owner's audit trail, newest first: rows older than before_id (0 = from the newest), optionally
-- of one action (empty = every action).
SELECT * FROM `companion_audit_logs`
WHERE owner_id = sqlc.arg(owner_id)
  AND (sqlc.arg(before_id) = 0 OR id < sqlc.arg(before_id))
  AND (sqlc.arg(action) = '' OR action = sqlc.arg(action))
ORDER BY id DESC
LIMIT ?;

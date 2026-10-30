-- Audit hash chain (migration 00105, T-M10-25; SPEC v6 §5.8; written by T-M10-33). admin_audit_logs is append-only
-- (BEFORE UPDATE / DELETE triggers), so row_hash is computed before the insert. The writer locks the head's one row,
-- never the log's tail, so inserts take no gap locks on the log.

-- name: LockAdminAuditChainHead :one
SELECT * FROM `admin_audit_chain_head` WHERE `id` = 1 FOR UPDATE;

-- name: MoveAdminAuditChainHead :exec
UPDATE `admin_audit_chain_head` SET `last_id` = sqlc.arg(last_id), `last_hash` = sqlc.arg(last_hash) WHERE `id` = 1;

-- name: SetAdminAuditChainGenesis :exec
-- The one-off genesis job: the hash of every row written before the chain started (ids up to genesis_last_id).
UPDATE `admin_audit_chain_head`
SET `genesis_last_id` = sqlc.arg(genesis_last_id), `genesis_hash` = sqlc.arg(genesis_hash),
    `last_id` = sqlc.arg(genesis_last_id), `last_hash` = sqlc.arg(genesis_hash)
WHERE `id` = 1;

-- name: InsertChainedAdminAuditLog :execlastid
INSERT INTO `admin_audit_logs` (admin_id, action, target_type, target_id, ip, detail, created_at, prev_hash, row_hash)
VALUES (sqlc.arg(admin_id), sqlc.arg(action), sqlc.arg(target_type), sqlc.arg(target_id), sqlc.arg(ip),
        sqlc.arg(detail), sqlc.arg(created_at), sqlc.narg(prev_hash), sqlc.narg(row_hash));

-- name: ListAdminAuditLogsForChain :many
-- Walk the log by id from after_id (verification and genesis), oldest first.
SELECT id, admin_id, action, target_type, target_id, ip, detail, created_at, prev_hash, row_hash
FROM `admin_audit_logs`
WHERE id > sqlc.arg(after_id)
ORDER BY id
LIMIT ?;

-- name: GetAdminAuditChainHead :one
-- Verification reads the head without locking it (inside a read-only REPEATABLE READ snapshot).
SELECT * FROM `admin_audit_chain_head` WHERE `id` = 1;

-- name: LastAdminAuditLogBefore :one
-- The anchor of a partial verification (?from=): the newest row before from.
SELECT id, row_hash FROM `admin_audit_logs` WHERE id < sqlc.arg(before_id) ORDER BY id DESC LIMIT 1;

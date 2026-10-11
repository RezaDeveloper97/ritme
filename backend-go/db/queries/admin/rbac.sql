-- Admin roles & permissions, DB audit log, temporary health-data access (bloom B-N9-02, migration 00060).

-- ---------------------------------------------------------------------------
-- Roles

-- name: ListAdminRoles :many
SELECT * FROM `admin_roles` ORDER BY role;

-- name: GetAdminRolePermissions :one
SELECT permissions FROM `admin_roles` WHERE role = ? LIMIT 1;

-- name: UpsertAdminRole :exec
INSERT INTO `admin_roles` (role, permissions, updated_by, created_at, updated_at)
VALUES (sqlc.arg(role), sqlc.arg(permissions), sqlc.arg(updated_by), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE permissions = VALUES(permissions), updated_by = VALUES(updated_by), updated_at = VALUES(updated_at);

-- name: DeleteAdminRole :exec
DELETE FROM `admin_roles` WHERE role = ?;

-- name: CountAdminsByRole :many
SELECT role, COUNT(*) AS admins FROM `admins` GROUP BY role;

-- ---------------------------------------------------------------------------
-- Audit log

-- name: InsertAdminAuditLog :exec
INSERT INTO `admin_audit_logs` (admin_id, action, target_type, target_id, ip, detail, created_at)
VALUES (sqlc.arg(admin_id), sqlc.arg(action), sqlc.arg(target_type), sqlc.arg(target_id), sqlc.arg(ip),
        sqlc.arg(detail), sqlc.arg(created_at));

-- name: ListAdminAuditLogs :many
-- Filters are optional: 0 / '' / NULL = no filter. action matches a prefix ("article." → every article action).
SELECT l.id, l.admin_id, l.action, l.target_type, l.target_id, l.ip, l.detail, l.created_at,
       a.name AS admin_name, a.role AS admin_role
FROM `admin_audit_logs` l
LEFT JOIN `admins` a ON a.id = l.admin_id
WHERE (sqlc.arg(admin_id) = 0 OR l.admin_id = sqlc.arg(admin_id))
  AND (sqlc.arg(action) = '' OR l.action LIKE CONCAT(sqlc.arg(action), '%'))
  AND (sqlc.arg(target_type) = '' OR l.target_type = sqlc.arg(target_type))
  AND (sqlc.arg(target_id) = 0 OR l.target_id = sqlc.arg(target_id))
  AND (sqlc.narg(from_at) IS NULL OR l.created_at >= sqlc.narg(from_at))
  AND (sqlc.narg(to_at) IS NULL OR l.created_at < sqlc.narg(to_at))
ORDER BY l.id DESC
LIMIT ? OFFSET ?;

-- name: CountAdminAuditLogs :one
SELECT COUNT(*)
FROM `admin_audit_logs` l
WHERE (sqlc.arg(admin_id) = 0 OR l.admin_id = sqlc.arg(admin_id))
  AND (sqlc.arg(action) = '' OR l.action LIKE CONCAT(sqlc.arg(action), '%'))
  AND (sqlc.arg(target_type) = '' OR l.target_type = sqlc.arg(target_type))
  AND (sqlc.arg(target_id) = 0 OR l.target_id = sqlc.arg(target_id))
  AND (sqlc.narg(from_at) IS NULL OR l.created_at >= sqlc.narg(from_at))
  AND (sqlc.narg(to_at) IS NULL OR l.created_at < sqlc.narg(to_at));

-- ---------------------------------------------------------------------------
-- Temporary health-data access

-- name: CreateHealthAccessRequest :execresult
INSERT INTO `admin_health_access_requests` (admin_id, user_id, reason, ticket, status, created_at, updated_at)
VALUES (sqlc.arg(admin_id), sqlc.arg(user_id), sqlc.arg(reason), sqlc.arg(ticket), 'pending', sqlc.arg(now), sqlc.arg(now));

-- name: GetHealthAccessRequest :one
SELECT r.*, a.name AS admin_name, d.name AS decided_by_name
FROM `admin_health_access_requests` r
LEFT JOIN `admins` a ON a.id = r.admin_id
LEFT JOIN `admins` d ON d.id = r.decided_by
WHERE r.id = ? LIMIT 1;

-- name: ListHealthAccessRequests :many
-- admin_id = 0 → every admin's requests (super); an empty status → every status.
SELECT r.*, a.name AS admin_name, d.name AS decided_by_name
FROM `admin_health_access_requests` r
LEFT JOIN `admins` a ON a.id = r.admin_id
LEFT JOIN `admins` d ON d.id = r.decided_by
WHERE (sqlc.arg(admin_id) = 0 OR r.admin_id = sqlc.arg(admin_id))
  AND (sqlc.arg(status) = '' OR r.status = sqlc.arg(status))
ORDER BY r.id DESC
LIMIT ? OFFSET ?;

-- name: CountHealthAccessRequests :one
SELECT COUNT(*) FROM `admin_health_access_requests` r
WHERE (sqlc.arg(admin_id) = 0 OR r.admin_id = sqlc.arg(admin_id))
  AND (sqlc.arg(status) = '' OR r.status = sqlc.arg(status));

-- name: DecideHealthAccessRequest :execrows
-- Only a pending request changes (approve / reject); a concurrent second decision affects 0 rows.
UPDATE `admin_health_access_requests`
SET status = sqlc.arg(status), decided_by = sqlc.arg(decided_by), decided_at = sqlc.arg(now),
    expires_at = sqlc.narg(expires_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: RevokeHealthAccessRequest :execrows
UPDATE `admin_health_access_requests`
SET status = 'revoked', updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status IN ('pending', 'approved');

-- name: ActiveHealthAccess :one
-- The admin's live grant for this user (approved and not expired), newest first.
SELECT id, expires_at FROM `admin_health_access_requests`
WHERE admin_id = sqlc.arg(admin_id) AND user_id = sqlc.arg(user_id) AND status = 'approved'
  AND expires_at > sqlc.arg(now)
ORDER BY expires_at DESC
LIMIT 1;

-- name: CountAdminChangesBy :one
-- B-N9-02b approver independence: admin.create / admin.update rows written by actor_id about target_id since since_at
-- (creation, password, role or any other account change of the would-be approver by the requester).
SELECT COUNT(*) FROM `admin_audit_logs`
WHERE admin_id = sqlc.arg(actor_id) AND target_type = 'admin' AND target_id = sqlc.arg(target_id)
  AND action IN ('admin.create', 'admin.update') AND created_at >= sqlc.arg(since_at);

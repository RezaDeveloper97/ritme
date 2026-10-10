-- B-N7-08 admin console: department editor (super only). Reads and writes never touch the code-held emergency floor.

-- name: ListAdminDepartments :many
SELECT * FROM `assistant_departments` ORDER BY sort_order, id;

-- name: GetAdminDepartment :one
SELECT * FROM `assistant_departments` WHERE id = ?;

-- name: CountDepartmentCode :one
SELECT COUNT(*) FROM `assistant_departments` WHERE code = ?;

-- name: InsertDepartment :execlastid
INSERT INTO `assistant_departments`
  (code, icon, sort_order, is_active, name, tagline, intro, does_not_do, quick_replies, system_prompt, guardrails,
   needs_review, created_at, updated_at)
VALUES (sqlc.arg(code), sqlc.narg(icon), sqlc.arg(sort_order), 0, sqlc.arg(name), sqlc.narg(tagline), sqlc.narg(intro),
        sqlc.narg(does_not_do), sqlc.narg(quick_replies), sqlc.arg(system_prompt), sqlc.narg(guardrails), 1,
        sqlc.arg(now), sqlc.arg(now));

-- name: UpdateDepartment :exec
UPDATE `assistant_departments`
SET icon = sqlc.narg(icon), sort_order = sqlc.arg(sort_order), is_active = sqlc.arg(is_active), name = sqlc.arg(name),
    tagline = sqlc.narg(tagline), intro = sqlc.narg(intro), does_not_do = sqlc.narg(does_not_do),
    quick_replies = sqlc.narg(quick_replies), system_prompt = sqlc.arg(system_prompt), guardrails = sqlc.narg(guardrails),
    needs_review = sqlc.arg(needs_review), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ApproveDepartment :execrows
UPDATE `assistant_departments` SET needs_review = 0, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

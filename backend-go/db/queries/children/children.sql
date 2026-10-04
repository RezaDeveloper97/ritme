-- Children (bloom B-N5-02, goose 00031). Health data: every child read is scoped by the owner (owner_id) or by an
-- active spouse family (families.spouse_user_id + companions.status = 'active'); child sub-rows are only read after
-- the child itself passed that check (internal/children.Service.Access).

-- name: CreateChild :execlastid
INSERT INTO `children` (owner_id, name, birth_date, sex, birth_weight_kg, birth_length_cm, birth_head_cm, delivery_type, created_at, updated_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(name), sqlc.arg(birth_date), sqlc.narg(sex), sqlc.narg(birth_weight_kg), sqlc.narg(birth_length_cm),
        sqlc.narg(birth_head_cm), sqlc.narg(delivery_type), sqlc.arg(now), sqlc.arg(now));

-- name: GetChild :one
SELECT * FROM `children` WHERE id = ? LIMIT 1;

-- name: UpdateChild :exec
UPDATE `children`
SET name = sqlc.arg(name), birth_date = sqlc.arg(birth_date), sex = sqlc.narg(sex), birth_weight_kg = sqlc.narg(birth_weight_kg),
    birth_length_cm = sqlc.narg(birth_length_cm), birth_head_cm = sqlc.narg(birth_head_cm), delivery_type = sqlc.narg(delivery_type),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id);

-- name: DeleteChild :execrows
DELETE FROM `children` WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id);

-- name: ListOwnedChildren :many
-- Youngest first (the artboard lists the baby before the older child).
SELECT * FROM `children` WHERE owner_id = ? ORDER BY birth_date DESC, id;

-- name: CountOwnedChildren :one
SELECT COUNT(*) FROM `children` WHERE owner_id = ?;

-- name: ListOwnedChildIDs :many
SELECT id FROM `children` WHERE owner_id = ?;

-- name: ListSharedChildren :many
-- Children the owner shared with this spouse account through an active spouse link's family (B-N4-01).
SELECT c.id, c.owner_id, c.name, c.birth_date, c.sex, c.birth_weight_kg, c.birth_length_cm, c.birth_head_cm, c.delivery_type,
       c.created_at, c.updated_at, f.companion_id
FROM `children` c
JOIN `family_children` fc ON fc.child_id = c.id
JOIN `families` f ON f.id = fc.family_id AND f.owner_id = c.owner_id
JOIN `companions` cm ON cm.id = f.companion_id
WHERE f.spouse_user_id = sqlc.arg(user_id) AND cm.companion_user_id = sqlc.arg(user_id) AND cm.status = 'active'
ORDER BY c.birth_date DESC, c.id;

-- name: GetSharedLink :one
-- The active spouse link through which user may see child (no row = not shared with them).
SELECT f.companion_id
FROM `family_children` fc
JOIN `families` f ON f.id = fc.family_id
JOIN `companions` cm ON cm.id = f.companion_id
JOIN `children` c ON c.id = fc.child_id AND c.owner_id = f.owner_id
WHERE fc.child_id = sqlc.arg(child_id) AND f.spouse_user_id = sqlc.arg(user_id) AND cm.companion_user_id = sqlc.arg(user_id)
  AND cm.status = 'active'
LIMIT 1;

-- name: InsertChildAudit :exec
-- The spouse's read of a shared child in the owner's companion audit trail (section children; never a payload).
INSERT INTO `companion_audit_logs` (owner_id, actor_id, companion_id, section, action, created_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(actor_id), sqlc.arg(companion_id), 'children', 'read', sqlc.arg(now));

-- name: GetUserName :one
SELECT name FROM `users` WHERE id = ? LIMIT 1;

-- name: ListMeasurements :many
SELECT * FROM `child_measurements` WHERE child_id = ? ORDER BY measured_on DESC, id DESC;

-- name: GetMeasurement :one
SELECT * FROM `child_measurements` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id) LIMIT 1;

-- name: GetMeasurementOn :one
SELECT * FROM `child_measurements` WHERE child_id = sqlc.arg(child_id) AND measured_on = sqlc.arg(measured_on) LIMIT 1;

-- name: UpsertMeasurement :exec
-- One row per day: a second save of the same day merges (values sent replace, values not sent stay).
INSERT INTO `child_measurements` (child_id, measured_on, weight_kg, length_cm, head_cm, created_at, updated_at)
VALUES (sqlc.arg(child_id), sqlc.arg(measured_on), sqlc.narg(weight_kg), sqlc.narg(length_cm), sqlc.narg(head_cm), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  weight_kg = COALESCE(VALUES(weight_kg), weight_kg),
  length_cm = COALESCE(VALUES(length_cm), length_cm),
  head_cm = COALESCE(VALUES(head_cm), head_cm),
  updated_at = VALUES(updated_at);

-- name: UpdateMeasurement :exec
UPDATE `child_measurements`
SET measured_on = sqlc.arg(measured_on), weight_kg = sqlc.narg(weight_kg), length_cm = sqlc.narg(length_cm), head_cm = sqlc.narg(head_cm),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id);

-- name: DeleteMeasurement :execrows
DELETE FROM `child_measurements` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id);

-- name: ListDoses :many
SELECT * FROM `child_vaccine_doses` WHERE child_id = ? ORDER BY given_on, id;

-- name: UpsertDose :exec
INSERT INTO `child_vaccine_doses` (child_id, dose_code, given_on, note, created_at, updated_at)
VALUES (sqlc.arg(child_id), sqlc.arg(dose_code), sqlc.arg(given_on), sqlc.narg(note), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE given_on = VALUES(given_on), note = VALUES(note), updated_at = VALUES(updated_at);

-- name: DeleteDose :execrows
DELETE FROM `child_vaccine_doses` WHERE child_id = sqlc.arg(child_id) AND dose_code = sqlc.arg(dose_code);

-- name: ListMilestoneChecks :many
SELECT * FROM `child_milestone_checks` WHERE child_id = ? ORDER BY checked_on, id;

-- name: UpsertMilestoneCheck :exec
INSERT INTO `child_milestone_checks` (child_id, milestone_code, checked_on, created_at, updated_at)
VALUES (sqlc.arg(child_id), sqlc.arg(milestone_code), sqlc.arg(checked_on), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE checked_on = VALUES(checked_on), updated_at = VALUES(updated_at);

-- name: DeleteMilestoneCheck :execrows
DELETE FROM `child_milestone_checks` WHERE child_id = sqlc.arg(child_id) AND milestone_code = sqlc.arg(milestone_code);

-- name: ListLearnArticles :many
-- Published articles a child_learn tip links by slug.
SELECT id, slug, read_time_minutes, image_url, image_path
FROM `articles`
WHERE is_published = 1 AND slug IN (sqlc.slice(slugs));

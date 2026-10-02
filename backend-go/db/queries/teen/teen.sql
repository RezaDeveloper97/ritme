-- Teen mode (CB-TEEN-01, goose 00029): onboarding answers, the school-kit checklist and the parent note. Minors' health
-- data: every statement is scoped by user_id in the statement itself. The life-stage mode is bloom's
-- user_life_profiles.life_mode; what a parent may see goes through internal/companion (type parent, teen sections).

-- name: GetTeenProfile :one
SELECT * FROM `teen_profiles` WHERE user_id = ? LIMIT 1;

-- name: UpsertTeenProfile :exec
-- Onboarding answers (Teen_Onb); the parent note is kept.
INSERT INTO `teen_profiles` (user_id, age_band, menarche, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(age_band), sqlc.arg(menarche), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  age_band = VALUES(age_band),
  menarche = VALUES(menarche),
  updated_at = VALUES(updated_at);

-- name: SetTeenParentNote :execrows
UPDATE `teen_profiles` SET parent_note = sqlc.narg(parent_note), updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id);

-- name: ListTeenKitChecks :many
SELECT item_code FROM `teen_kit_checks` WHERE user_id = ? ORDER BY id;

-- name: CheckTeenKitItem :exec
INSERT IGNORE INTO `teen_kit_checks` (user_id, item_code, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(item_code), sqlc.arg(now), sqlc.arg(now));

-- name: UncheckTeenKitItem :exec
DELETE FROM `teen_kit_checks` WHERE user_id = sqlc.arg(user_id) AND item_code = sqlc.arg(item_code);

-- name: GetTeenLifeMode :one
-- The stored life-stage mode (bloom B-N2-01; no row / NULL = not teen).
SELECT life_mode FROM `user_life_profiles` WHERE user_id = ? LIMIT 1;

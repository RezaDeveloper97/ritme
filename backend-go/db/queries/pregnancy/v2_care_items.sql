-- Pregnancy v2 care plan (table pregnancy_care_items, T-M7-01): admin-defined visits / tests / scans /
-- vaccines with a pregnancy-week window. Visits themselves are M3 appointments (meta.care_item_key).

-- name: ListActiveCareItems :many
SELECT * FROM `pregnancy_care_items` WHERE is_active = 1 ORDER BY sort_order, id;

-- name: ListCareItems :many
-- Admin list: active and inactive.
SELECT * FROM `pregnancy_care_items` ORDER BY sort_order, id;

-- name: GetCareItem :one
SELECT * FROM `pregnancy_care_items` WHERE id = ? LIMIT 1;

-- name: GetCareItemByKey :one
SELECT * FROM `pregnancy_care_items` WHERE `key` = ? LIMIT 1;

-- name: CreateCareItem :execlastid
INSERT INTO `pregnancy_care_items` (
  `key`, title, prep, kind, week_from, week_to, remind_before, sort_order, is_active, created_at, updated_at
) VALUES (
  sqlc.arg(item_key), sqlc.arg(title), sqlc.arg(prep), sqlc.arg(kind), sqlc.arg(week_from), sqlc.arg(week_to),
  sqlc.arg(remind_before), sqlc.arg(sort_order), sqlc.arg(is_active), sqlc.arg(now), sqlc.arg(now)
);

-- name: UpdateCareItem :execrows
-- `key` is immutable (appointments reference it).
UPDATE `pregnancy_care_items` SET
  title = sqlc.arg(title), prep = sqlc.arg(prep), kind = sqlc.arg(kind), week_from = sqlc.arg(week_from),
  week_to = sqlc.arg(week_to), remind_before = sqlc.arg(remind_before), sort_order = sqlc.arg(sort_order),
  is_active = sqlc.arg(is_active), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetCareItemSortOrder :execrows
UPDATE `pregnancy_care_items` SET sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: SetCareItemActive :execrows
UPDATE `pregnancy_care_items` SET is_active = sqlc.arg(is_active), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteCareItem :execrows
DELETE FROM `pregnancy_care_items` WHERE id = ?;

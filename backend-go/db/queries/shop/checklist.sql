-- Layette checklist template + the user's own checklist state, and the period-supply reminder opt-in (CB-SHOP-01).
-- The reminder's timing comes from cycle predictions through a Go interface; no query here joins a health table.

-- name: ListActiveLayetteItems :many
SELECT * FROM `shop_layette_items` WHERE is_active = 1 ORDER BY group_key, sort_order, id;

-- name: ListUserChecklistItems :many
SELECT * FROM `shop_user_checklist_items` WHERE user_id = ? ORDER BY group_key, sort_order, id;

-- name: SetUserChecklistState :exec
INSERT INTO `shop_user_checklist_items` (`user_id`, `layette_item_id`, `group_key`, `state`, `quantity`, `created_at`,
  `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE `state` = VALUES(`state`), `quantity` = VALUES(`quantity`), `updated_at` = VALUES(`updated_at`);

-- name: GetSupplyReminder :one
SELECT * FROM `shop_supply_reminders` WHERE user_id = ? LIMIT 1;

-- name: UpsertSupplyReminder :exec
INSERT INTO `shop_supply_reminders` (`user_id`, `is_enabled`, `lead_days`, `product_id`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE `is_enabled` = VALUES(`is_enabled`), `lead_days` = VALUES(`lead_days`),
  `product_id` = VALUES(`product_id`), `updated_at` = VALUES(`updated_at`);

-- ---------------------------------------------------------------- CB-SHOP-04: checklist API + supply reminder

-- name: ListLayetteTemplate :many
-- The active template with the slug of its «see» category (an inactive category → no link).
SELECT l.id, l.group_key, l.title, l.note, l.suggested_qty, l.is_essential, l.is_first_days, l.sort_order,
  c.slug AS category_slug
FROM `shop_layette_items` l
LEFT JOIN `shop_categories` c ON c.id = l.category_id AND c.is_active = 1
WHERE l.is_active = 1
ORDER BY l.group_key, l.sort_order, l.id;

-- name: GetActiveLayetteItem :one
SELECT * FROM `shop_layette_items` WHERE id = ? AND is_active = 1 LIMIT 1;

-- name: CountCustomChecklistItems :one
SELECT COUNT(*) FROM `shop_user_checklist_items` WHERE user_id = ? AND layette_item_id IS NULL;

-- name: GetCustomChecklistItem :one
SELECT * FROM `shop_user_checklist_items` WHERE id = ? AND user_id = ? AND layette_item_id IS NULL LIMIT 1;

-- name: InsertCustomChecklistItem :execlastid
INSERT INTO `shop_user_checklist_items` (`user_id`, `layette_item_id`, `custom_title`, `group_key`, `state`, `quantity`,
  `sort_order`, `created_at`, `updated_at`)
VALUES (sqlc.arg(user_id), NULL, sqlc.arg(custom_title), sqlc.arg(group_key), sqlc.arg(state), sqlc.narg(quantity), 0,
  sqlc.arg(now), sqlc.arg(now));

-- name: UpdateCustomChecklistItem :execrows
UPDATE `shop_user_checklist_items`
SET `custom_title` = sqlc.arg(custom_title), `group_key` = sqlc.arg(group_key), `state` = sqlc.arg(state),
  `quantity` = sqlc.narg(quantity), `updated_at` = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND layette_item_id IS NULL;

-- name: DeleteCustomChecklistItem :execrows
DELETE FROM `shop_user_checklist_items` WHERE id = ? AND user_id = ? AND layette_item_id IS NULL;

-- name: CountActiveProduct :one
SELECT COUNT(*) FROM `shop_products` WHERE id = ? AND is_active = 1;

-- name: FindSupplyCareReminder :one
-- The care reminder the supply opt-in created (meta.source = shop_supply). It lives in `reminders` like any other
-- custom reminder, so GET /reminders lists it and the user can switch it off or delete it there.
SELECT * FROM `reminders`
WHERE user_id = ? AND `type` = 'custom' AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source')) = 'shop_supply'
ORDER BY id LIMIT 1;

-- name: InsertSupplyCareReminder :execlastid
INSERT INTO `reminders` (user_id, `type`, title, subtitle, notes, scheduled_at, recurrence, recurrence_time, starts_on,
  ends_on, is_active, meta, created_at, updated_at)
VALUES (sqlc.arg(user_id), 'custom', sqlc.arg(title), NULL, NULL, sqlc.arg(scheduled_at), 'none', NULL, NULL, NULL, 1,
  sqlc.arg(meta), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateSupplyCareReminder :exec
-- Moves the reminder to the current prediction; the user's own switch (is_active) stays.
UPDATE `reminders` SET title = sqlc.arg(title), scheduled_at = sqlc.arg(scheduled_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteSupplyCareReminders :exec
DELETE FROM `reminders`
WHERE user_id = ? AND `type` = 'custom' AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source')) = 'shop_supply';

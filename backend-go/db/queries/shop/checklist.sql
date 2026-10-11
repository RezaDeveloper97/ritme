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

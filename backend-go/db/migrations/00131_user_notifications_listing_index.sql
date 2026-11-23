-- 00131_user_notifications_listing_index.sql — T-M12-32 (P-07): the inbox list (GET /notifications,
-- home.ListUserNotifications: WHERE user_id = ? ORDER BY created_at DESC, id LIMIT ? OFFSET ?) is read through a
-- (user_id, created_at DESC, id) key instead of a filesort over every row of the user (Web Push grows the table).
-- created_at is DESC in the key so it matches the existing mixed order exactly (newest first, ties by id ascending):
-- the response order is unchanged. `user_notifications_user_id_read_at_index` stays — the unread count and
-- mark-all-read use it.
--
-- Laravel twin: 2026_11_18_000131_add_user_notifications_listing_index.php (schema only).

-- +goose Up
CREATE INDEX IF NOT EXISTS `user_notifications_user_id_created_at_id_index`
  ON `user_notifications` (`user_id`, `created_at` DESC, `id`);

-- +goose Down
DROP INDEX IF EXISTS `user_notifications_user_id_created_at_id_index` ON `user_notifications`;

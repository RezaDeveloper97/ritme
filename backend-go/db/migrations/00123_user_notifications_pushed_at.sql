-- 00123_user_notifications_pushed_at.sql — Web Push for the in-app inbox: `pushed_at` is when the push sender
-- (internal/care Pusher.TickNotices) handled a user_notifications row — pushed to the user's browsers, or
-- deliberately left silent (category off, already read, no subscription, too old). NULL = not handled yet; the
-- sender's working set is read through the (pushed_at, id) key. Rows that exist now are marked handled, so nothing
-- written before this migration is ever announced.
--
-- Laravel twin: 2026_11_11_000123_add_pushed_at_to_user_notifications.php (schema only).

-- +goose Up
ALTER TABLE `user_notifications`
  ADD COLUMN IF NOT EXISTS `pushed_at` datetime DEFAULT NULL,
  ADD KEY IF NOT EXISTS `user_notifications_pushed_at_id_index` (`pushed_at`,`id`);

UPDATE `user_notifications` SET `pushed_at` = COALESCE(`created_at`, NOW()) WHERE `pushed_at` IS NULL;

-- +goose Down
ALTER TABLE `user_notifications` DROP INDEX IF EXISTS `user_notifications_pushed_at_id_index`;
ALTER TABLE `user_notifications` DROP COLUMN IF EXISTS `pushed_at`;

-- 00120_drop_pregnancy_week_bookmark.sql — the week-page bookmark is removed from the product (no list ever read
-- it), so pregnancy_week_user_state keeps only the done task keys.
-- Laravel twin: 2026_11_10_000120_drop_pregnancy_week_bookmark.php.

-- +goose Up
ALTER TABLE `pregnancy_week_user_state` DROP COLUMN IF EXISTS `bookmarked`;

-- +goose Down
ALTER TABLE `pregnancy_week_user_state` ADD COLUMN IF NOT EXISTS `bookmarked` tinyint(1) NOT NULL DEFAULT 0 AFTER `week`;

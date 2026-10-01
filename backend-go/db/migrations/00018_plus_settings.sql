-- 00018_plus_settings.sql — admin-editable Ritme Plus settings (B-N2-06, bloom/N2).
--
--   plus_settings   key/value rows the admin «اشتراک‌ها و پرداخت» module edits (B-N2-09). One row per key; unknown
--                   or unreadable values fall back to the code default (internal/plus/settings.go), so a missing row
--                   never breaks checkout.
--
-- Keys:
--   trial_offer_percent   0–100: the discount every plan gets while the user's 7-day trial runs (the trial banner's
--                         «۵۰٪ تخفیف», applied server-side at checkout). 0 turns the offer off. Seeded 50.
--
-- 00016 is intentionally unused, 00017 is reserved by another work stream.
-- Twin of backend/database/migrations/2026_10_01_000018_create_plus_settings_table.php (schema + the seed row), so
-- `make schema-diff` stays green. IF NOT EXISTS / INSERT IGNORE: a database whose Laravel half already ran, or an
-- admin edit, is never overwritten.

-- +goose Up
CREATE TABLE IF NOT EXISTS `plus_settings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(64) NOT NULL,
  `value` varchar(255) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_settings_key_unique` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `plus_settings` (`key`, `value`, `created_at`, `updated_at`) VALUES
  ('trial_offer_percent', '50', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DROP TABLE IF EXISTS `plus_settings`;

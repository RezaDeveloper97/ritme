-- 00065_analytics_events.sql — first-party product analytics events (bloom B-N9-08, D-89).
--
-- analytics_events   one row per accepted event: the allow-listed `name` (app_open, onboarding_done, log_saved,
--                    plus_paywall_view, plus_purchase, …), the internal `user_id` (never exported, never hashed into a
--                    third party; rows go with the account) and `props` — a tiny JSON object of allow-listed, enumerated,
--                    non-health keys (e.g. {"surface":"home"}). NO health payload is ever stored: the ingest endpoint
--                    rejects every key that is not in the per-event allow-list. `occurred_on` is the Tehran calendar
--                    day (the DAU / cohort key). Rows are only written for users who granted the `anonymous_stats`
--                    consent. Read only as aggregates (internal/metrics, k >= 10 suppression).
--
-- Twin of backend/database/migrations/2026_10_11_000065_create_analytics_events_table.php. IF NOT EXISTS so a database
-- whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `analytics_events` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `name` varchar(48) NOT NULL,
  `props` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`props`)),
  `occurred_on` date NOT NULL,
  `occurred_at` datetime NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `analytics_events_occurred_on_user_id_index` (`occurred_on`,`user_id`),
  KEY `analytics_events_name_occurred_on_index` (`name`,`occurred_on`),
  KEY `analytics_events_user_id_name_occurred_on_index` (`user_id`,`name`,`occurred_on`),
  CONSTRAINT `analytics_events_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `analytics_events`;

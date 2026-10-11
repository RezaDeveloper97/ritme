-- 00064_metric_snapshots.sql — hourly/nightly aggregate snapshots for the admin overview dashboard (bloom B-N9-03, D-85).
--
-- metric_snapshots   one row per Tehran calendar day (UNIQUE `day`), rewritten by the aggregation job through the day.
--                    `payload` is the JSON metrics document (aggregates only — counts, rates, series; never a user id
--                    or per-user health value; groups under k = 10 are suppressed before storing).
--
-- Twin of backend/database/migrations/2026_10_11_000064_create_metric_snapshots_table.php. IF NOT EXISTS so a database
-- whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `metric_snapshots` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `day` date NOT NULL,
  `payload` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`payload`)),
  `computed_at` datetime NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `metric_snapshots_day_unique` (`day`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `metric_snapshots`;

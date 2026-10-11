-- 00076_okr.sql — OKR module bound to live metrics (bloom B-N9-09, D-95).
--
-- okr_objectives    an objective: title, owner (free text, a person or team), period label + the dates it covers.
-- okr_key_results   a key result with baseline and target. `metric_key` set = bound to a live aggregate metric
--                   (metrics snapshot / KPI, resolved at read time); NULL = manual, progress from `manual_value`.
--                   Progress and status are never stored: they are computed from the live value on every read.
-- Aggregates only: no user data is stored or resolved here.
--
-- Twin of backend/database/migrations/2026_10_11_000076_create_okr_tables.php. IF NOT EXISTS so a database whose
-- Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `okr_objectives` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(255) NOT NULL,
  `owner` varchar(120) DEFAULT NULL,
  `period` varchar(40) NOT NULL,
  `starts_on` date NOT NULL,
  `ends_on` date NOT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `okr_objectives_starts_on_ends_on_index` (`starts_on`,`ends_on`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `okr_key_results` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `objective_id` bigint(20) unsigned NOT NULL,
  `title` varchar(255) NOT NULL,
  `metric_key` varchar(40) DEFAULT NULL,
  `unit` varchar(16) NOT NULL DEFAULT 'count',
  `baseline` decimal(16,2) NOT NULL DEFAULT 0.00,
  `target` decimal(16,2) NOT NULL,
  `manual_value` decimal(16,2) DEFAULT NULL,
  `owner` varchar(120) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `okr_key_results_objective_id_sort_order_index` (`objective_id`,`sort_order`),
  CONSTRAINT `okr_key_results_objective_id_foreign` FOREIGN KEY (`objective_id`) REFERENCES `okr_objectives` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `okr_key_results`;
DROP TABLE IF EXISTS `okr_objectives`;

-- 00109_engine_snapshots.sql — the engine's output over time, for the SPEC v6 §5.6 KPIs and §5.2 user monitoring
-- (T-M9-39, D-137; docs/engine-v6/DECISIONS.md A-05, A-06; gap 06 §4.1, §4.2). Go-only analytics tables: written by
-- internal/cycle/snapshots, read by the KPI and admin jobs (T-M10-30, T-M10-32); never read by the engine.
--
-- engine_daily_snapshots   one row per active user (A-06) and user-local day, written by the nightly
--                          `engine:snapshot_nightly` job (idempotent upsert on the PK). Codes only — the spec spellings
--                          of main_phase, subphase, confidence, data_quality and mode, the warning bit set
--                          (engine.WarningSet), days_late and the engine_rev the state was built from — never a
--                          predicted date (§2.40). KEY (day) for the per-day distributions. Kept 400 days
--                          (`engine:snapshot_purge`).
-- engine_prediction_log    per real period start: `first` = the prediction of the next start emitted for that cycle
--                          (resolved on the start day), `last` = the prediction valid the day before the next real
--                          start; actual_start / error_days (actual − predicted, signed) are filled when that next
--                          start is written. UNIQUE (user_id, cycle_start, kind). Kept forever (A-05; the MAE KPI uses
--                          `first` with basis_cycles ≥ 2, T-M10-30).
--
-- Both cascade with the user (account deletion).
--
-- Twin of backend/database/migrations/2026_10_27_000109_create_engine_snapshot_tables.php (Laravel owns the prod
-- schema until T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built
-- tables; IF [NOT] EXISTS so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `engine_daily_snapshots` (
  `user_id` bigint(20) unsigned NOT NULL,
  `day` date NOT NULL,
  `main_phase` varchar(24) NOT NULL,
  `subphase` varchar(32) NOT NULL,
  `confidence` varchar(8) NOT NULL,
  `data_quality` varchar(16) NOT NULL,
  `warnings_mask` int(10) unsigned NOT NULL DEFAULT 0,
  `days_late` smallint(6) DEFAULT NULL,
  `mode` varchar(16) NOT NULL,
  `engine_rev` bigint(20) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`user_id`,`day`),
  KEY `engine_daily_snapshots_day_index` (`day`),
  CONSTRAINT `engine_daily_snapshots_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `engine_prediction_log` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `cycle_start` date NOT NULL,
  `kind` enum('first','last') NOT NULL,
  `predicted_next` date NOT NULL,
  `basis_cycles` tinyint(3) unsigned NOT NULL,
  `ovulation_source` varchar(16) NOT NULL,
  `confidence` varchar(8) NOT NULL,
  `actual_start` date DEFAULT NULL,
  `error_days` smallint(6) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `engine_prediction_log_user_id_cycle_start_kind_unique` (`user_id`,`cycle_start`,`kind`),
  CONSTRAINT `engine_prediction_log_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `engine_prediction_log`;
DROP TABLE IF EXISTS `engine_daily_snapshots`;

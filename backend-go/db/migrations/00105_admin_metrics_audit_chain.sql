-- 00105_admin_metrics_audit_chain.sql — storage for metric alerts, user activity, the admin user directory, saved
-- segments, reference codes, the audit hash chain and persisted safety runs (T-M10-25; SPEC v6 §5.1, §5.2, §5.6, §5.8;
-- docs/engine-v6/DECISIONS.md A-06, A-08, A-09; gap 06 §2.4–§2.8, §4.3, §4.5). Schema only: every writer and reader is
-- a later task (T-M10-24, T-M10-30 … T-M10-33).
--
-- user_activity_days       one row per user and Tehran day with a session (`session`) and/or a data log (`logged`);
--                          DAU / MAU / logging users (§5.6) and the activity projection read it. Writers upsert
--                          (INSERT … ON DUPLICATE KEY UPDATE session = 1 / logged = 1). PK (day, user_id) for per-day
--                          counts, KEY (user_id, day) for one user's history (and the FK).
-- metric_alert_rules       the automatic dashboard alerts (§5.1), keyed by metric_key. comparator gt | lt |
--                          drop_vs_mean (fraction below the window_days mean) | rising_days (consecutive days rising,
--                          more than threshold); min_volume = the least daily volume a subject needs to be evaluated
--                          (CTR: shown per day). Seeded with the §5.1 rules plus safety_selftest_failed (T-M10-24).
-- metric_alerts            one alert per breach; `subject` = '' or the message rule/version for CTR. `open_flag` is 1
--                          while open and NULL once resolved, so UNIQUE (rule_id, subject, open_flag) allows exactly
--                          one open alert per (rule, subject) and any number of resolved ones (NULLs never collide).
-- admin_user_activity      the nightly user-directory projection (§5.2): last activity, log days in 30 days and in the
--                          recent / prior 14 days, churn (A-08), mode, plan, data_quality, main engine warning.
-- admin_segments           saved user-list filters (§5.2), `filters` JSON as the list endpoint takes them.
-- users.ref_code           the non-reversible reference code (A-09: `RT-` + 6 base32 chars of an HMAC of the id), UNIQUE
--                          for search; NULL until the writer fills it.
-- admin_audit_logs.prev_hash / row_hash, admin_audit_chain_head
--                          the audit hash chain (§5.8, T-M10-33). The writer locks the head's one row (`FOR UPDATE`)
--                          instead of the log's tail, so inserts never take gap locks on the log, computes row_hash
--                          BEFORE the insert (the row cannot be updated afterwards) and moves the head. genesis_* = the
--                          hash of the rows written before the chain started. CHECK (id = 1) keeps it a singleton.
-- safety_scenario_runs     one row per scenario of a nightly safety self-test run (§5.8, T-M10-24); run_id = ULID.
--
-- admin_audit_logs and clinical_config_approvals become append-only in the database: BEFORE UPDATE and BEFORE DELETE
-- triggers SIGNAL. Neither table has a foreign key, and neither may gain one whose action updates or deletes rows
-- (FK actions do not fire triggers, so such a key would silently bypass the guard). Deleting an admin or a user does
-- not touch these rows. TRUNCATE does not fire triggers either (the test reset uses it).
--
-- Indexes for the KPI scans: daily_health_logs (log_date, user_id) for `WHERE log_date >= ?`, cycle_histories
-- (period_start_date) for `WHERE period_start_date BETWEEN ? AND ?`, users (created_at) for sign-ups per day.
--
-- The triggers are created under the session sql_mode and connection collation of Laravel's strict MySQL connection
-- (both restored afterwards), so mariadb-dump prints them the same on both sides (`make schema-diff` compares them).
-- They go through PREPARE because sqlc parses these files as the schema and its parser has no CREATE TRIGGER.
-- Creating a trigger needs the TRIGGER privilege (and SUPER or log_bin_trust_function_creators=1 with binary logging).
--
-- Twin of backend/database/migrations/2026_10_23_000105_create_admin_metrics_audit_chain_tables.php (Laravel owns the
-- prod schema until T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built
-- tables; IF [NOT] EXISTS everywhere so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `user_activity_days` (
  `user_id` bigint(20) unsigned NOT NULL,
  `day` date NOT NULL,
  `session` tinyint(1) NOT NULL DEFAULT 0,
  `logged` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`day`,`user_id`),
  KEY `user_activity_days_user_id_day_index` (`user_id`,`day`),
  CONSTRAINT `user_activity_days_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `metric_alert_rules` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `metric_key` varchar(64) NOT NULL,
  `comparator` varchar(16) NOT NULL,
  `threshold` decimal(12,4) NOT NULL,
  `window_days` smallint(5) unsigned NOT NULL,
  `min_volume` int(10) unsigned DEFAULT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `metric_alert_rules_metric_key_unique` (`metric_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `metric_alerts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `rule_id` bigint(20) unsigned NOT NULL,
  `subject` varchar(64) NOT NULL DEFAULT '',
  `day` date NOT NULL,
  `value` decimal(12,4) NOT NULL,
  `threshold` decimal(12,4) NOT NULL,
  `state` varchar(12) NOT NULL DEFAULT 'open',
  `open_flag` tinyint(1) GENERATED ALWAYS AS (if(`state` = 'open',1,NULL)) STORED,
  `opened_at` timestamp NULL DEFAULT NULL,
  `resolved_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `metric_alerts_rule_id_subject_open_flag_unique` (`rule_id`,`subject`,`open_flag`),
  KEY `metric_alerts_state_opened_at_index` (`state`,`opened_at`),
  CONSTRAINT `metric_alerts_rule_id_foreign` FOREIGN KEY (`rule_id`) REFERENCES `metric_alert_rules` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `admin_user_activity` (
  `user_id` bigint(20) unsigned NOT NULL,
  `last_activity_at` timestamp NULL DEFAULT NULL,
  `log_days_30` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `log_days_recent14` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `log_days_prior14` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `churn` varchar(8) NOT NULL,
  `mode` varchar(16) NOT NULL,
  `plan` varchar(16) NOT NULL,
  `data_quality` varchar(16) DEFAULT NULL,
  `main_warning` varchar(48) DEFAULT NULL,
  `updated_on` date NOT NULL,
  PRIMARY KEY (`user_id`),
  KEY `admin_user_activity_churn_index` (`churn`),
  KEY `admin_user_activity_plan_index` (`plan`),
  KEY `admin_user_activity_mode_index` (`mode`),
  KEY `admin_user_activity_data_quality_index` (`data_quality`),
  KEY `admin_user_activity_last_activity_at_index` (`last_activity_at`),
  CONSTRAINT `admin_user_activity_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `admin_segments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(80) NOT NULL,
  `filters` json NOT NULL,
  `created_by` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `admin_segments_name_unique` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `ref_code` varchar(12) DEFAULT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS `users_ref_code_unique` ON `users` (`ref_code`);
CREATE INDEX IF NOT EXISTS `users_created_at_index` ON `users` (`created_at`);

ALTER TABLE `admin_audit_logs` ADD COLUMN IF NOT EXISTS `prev_hash` char(64) DEFAULT NULL;
ALTER TABLE `admin_audit_logs` ADD COLUMN IF NOT EXISTS `row_hash` char(64) DEFAULT NULL;

CREATE TABLE IF NOT EXISTS `admin_audit_chain_head` (
  `id` tinyint(3) unsigned NOT NULL,
  `last_id` bigint(20) unsigned DEFAULT NULL,
  `last_hash` char(64) DEFAULT NULL,
  `genesis_last_id` bigint(20) unsigned DEFAULT NULL,
  `genesis_hash` char(64) DEFAULT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `admin_audit_chain_head_singleton` CHECK (`id` = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `admin_audit_chain_head` (`id`) VALUES (1);

CREATE TABLE IF NOT EXISTS `safety_scenario_runs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `run_id` char(26) NOT NULL,
  `scenario` varchar(64) NOT NULL,
  `passed` tinyint(1) NOT NULL,
  `detail` json DEFAULT NULL,
  `duration_ms` int(10) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `safety_scenario_runs_created_at_index` (`created_at`),
  KEY `safety_scenario_runs_scenario_created_at_index` (`scenario`,`created_at`),
  KEY `safety_scenario_runs_run_id_index` (`run_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX IF NOT EXISTS `daily_health_logs_log_date_user_id_index` ON `daily_health_logs` (`log_date`,`user_id`);
CREATE INDEX IF NOT EXISTS `cycle_histories_period_start_date_index` ON `cycle_histories` (`period_start_date`);

-- SPEC §5.1 thresholds (+ the T-M10-24 self-test). INSERT IGNORE keeps a rule an admin already edited.
INSERT IGNORE INTO `metric_alert_rules` (`metric_key`, `comparator`, `threshold`, `window_days`, `min_volume`, `enabled`, `created_at`, `updated_at`) VALUES
  ('ocr_correction_rate',    'gt',           0.1200,  7, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('message_ctr',            'drop_vs_mean', 0.3000,  7, 200,  1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('rui_users',              'rising_days',  7.0000,  7, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('engine_mae',             'gt',           2.5000, 30, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('safety_selftest_failed', 'gt',           0.0000,  1, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

SET @admin_audit_sql_mode = @@SESSION.sql_mode;
SET @admin_audit_collation = @@SESSION.collation_connection;
SET SESSION collation_connection = 'utf8mb4_unicode_ci';
SET SESSION sql_mode = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION';

SET @admin_audit_ddl = 'CREATE TRIGGER IF NOT EXISTS `admin_audit_logs_no_update` BEFORE UPDATE ON `admin_audit_logs` FOR EACH ROW\n  SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''admin_audit_logs is append-only''';
PREPARE admin_audit_stmt FROM @admin_audit_ddl;
EXECUTE admin_audit_stmt;
DEALLOCATE PREPARE admin_audit_stmt;

SET @admin_audit_ddl = 'CREATE TRIGGER IF NOT EXISTS `admin_audit_logs_no_delete` BEFORE DELETE ON `admin_audit_logs` FOR EACH ROW\n  SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''admin_audit_logs is append-only''';
PREPARE admin_audit_stmt FROM @admin_audit_ddl;
EXECUTE admin_audit_stmt;
DEALLOCATE PREPARE admin_audit_stmt;

SET @admin_audit_ddl = 'CREATE TRIGGER IF NOT EXISTS `clinical_config_approvals_no_update` BEFORE UPDATE ON `clinical_config_approvals` FOR EACH ROW\n  SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''clinical_config_approvals is append-only''';
PREPARE admin_audit_stmt FROM @admin_audit_ddl;
EXECUTE admin_audit_stmt;
DEALLOCATE PREPARE admin_audit_stmt;

SET @admin_audit_ddl = 'CREATE TRIGGER IF NOT EXISTS `clinical_config_approvals_no_delete` BEFORE DELETE ON `clinical_config_approvals` FOR EACH ROW\n  SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''clinical_config_approvals is append-only''';
PREPARE admin_audit_stmt FROM @admin_audit_ddl;
EXECUTE admin_audit_stmt;
DEALLOCATE PREPARE admin_audit_stmt;

SET SESSION sql_mode = @admin_audit_sql_mode;
SET SESSION collation_connection = @admin_audit_collation;

-- +goose Down
DROP TRIGGER IF EXISTS `clinical_config_approvals_no_delete`;
DROP TRIGGER IF EXISTS `clinical_config_approvals_no_update`;
DROP TRIGGER IF EXISTS `admin_audit_logs_no_delete`;
DROP TRIGGER IF EXISTS `admin_audit_logs_no_update`;
DROP INDEX IF EXISTS `cycle_histories_period_start_date_index` ON `cycle_histories`;
DROP INDEX IF EXISTS `daily_health_logs_log_date_user_id_index` ON `daily_health_logs`;
DROP TABLE IF EXISTS `safety_scenario_runs`;
DROP TABLE IF EXISTS `admin_audit_chain_head`;
ALTER TABLE `admin_audit_logs` DROP COLUMN IF EXISTS `row_hash`;
ALTER TABLE `admin_audit_logs` DROP COLUMN IF EXISTS `prev_hash`;
DROP INDEX IF EXISTS `users_created_at_index` ON `users`;
DROP INDEX IF EXISTS `users_ref_code_unique` ON `users`;
ALTER TABLE `users` DROP COLUMN IF EXISTS `ref_code`;
DROP TABLE IF EXISTS `admin_segments`;
DROP TABLE IF EXISTS `admin_user_activity`;
DROP TABLE IF EXISTS `metric_alerts`;
DROP TABLE IF EXISTS `metric_alert_rules`;
DROP TABLE IF EXISTS `user_activity_days`;

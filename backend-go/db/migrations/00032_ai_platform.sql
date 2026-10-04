-- 00032_ai_platform.sql — AI adapter platform: versioned consents and the usage + cost log (B-N6-05, bloom/N6).
--
--  1. `user_consents.version` — the version of the consent text the user accepted (internal/consent catalog).
--     NULL = granted before texts were versioned (B-N1-12 rows) or never granted; an AI feature whose consent is
--     NULL or older than the current text answers 403 consent_required until the user accepts the current text.
--  2. `ai_usage_logs` — one row per provider call: who (user_id, NULL for system jobs and after account
--     deletion — the cost history must outlive the account for the global daily cap), feature, op, provider,
--     model, token counts, audio / image byte sizes, the estimated cost in micro-USD (config price table),
--     latency and outcome. Never the prompt, the answer, a file or any other content or PII.
--
-- Twin of backend/database/migrations/2026_10_03_000032_create_ai_platform_tables.php (Laravel owns the prod schema
-- until T-M2-27; schema only, no models/routes), so `make schema-diff` stays green. Spelled the way mariadb-dump
-- prints the Laravel-built tables. IF [NOT] EXISTS: a database whose Laravel half already ran must not fail here.

-- +goose Up
ALTER TABLE `user_consents` ADD COLUMN IF NOT EXISTS `version` smallint(5) unsigned DEFAULT NULL AFTER `granted`;

CREATE TABLE IF NOT EXISTS `ai_usage_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `feature` varchar(32) NOT NULL,
  `op` varchar(32) NOT NULL,
  `provider` varchar(16) NOT NULL,
  `model` varchar(64) NOT NULL,
  `input_tokens` int(10) unsigned NOT NULL DEFAULT 0,
  `output_tokens` int(10) unsigned NOT NULL DEFAULT 0,
  `audio_bytes` int(10) unsigned NOT NULL DEFAULT 0,
  `image_bytes` int(10) unsigned NOT NULL DEFAULT 0,
  `cost_micros` bigint(20) unsigned NOT NULL DEFAULT 0,
  `latency_ms` int(10) unsigned NOT NULL DEFAULT 0,
  `ok` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `ai_usage_logs_created_at_index` (`created_at`),
  KEY `ai_usage_logs_user_id_created_at_index` (`user_id`,`created_at`),
  KEY `ai_usage_logs_feature_created_at_index` (`feature`,`created_at`),
  CONSTRAINT `ai_usage_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `ai_usage_logs`;
ALTER TABLE `user_consents` DROP COLUMN IF EXISTS `version`;

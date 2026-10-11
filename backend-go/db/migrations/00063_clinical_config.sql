-- 00063_clinical_config.sql — versioned clinical thresholds & config with two medical approvals (bloom B-N9-11, D-88).
--
-- clinical_config_versions   one proposed / approved / rejected version of a clinical key (internal/clinicalconfig
--                            registry: bp.crisis, glucose.fasting, epds.cutoff.likely, engine.period_hard_cap, …).
--                            Version 1 of every key is the code-held default and is never stored; rows are v2+.
--                            value is a JSON object of named numbers ({"systolic":180,"diastolic":120}); the server
--                            reads the approved version whose effective_from is the latest on or before target_date.
-- clinical_config_approvals  one medical advisor's approval of a version; two distinct advisors (never the proposer)
--                            make it approved.
--
-- Twin of backend/database/migrations/2026_10_11_000063_create_clinical_config_tables.php (Laravel owns the prod
-- schema until T-M2-27), so `make schema-diff` stays green. IF NOT EXISTS so a database whose Laravel half already
-- ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `clinical_config_versions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `config_key` varchar(64) NOT NULL,
  `version` int(10) unsigned NOT NULL,
  `value` text NOT NULL,
  `source` varchar(255) NOT NULL DEFAULT '',
  `note` varchar(500) NOT NULL DEFAULT '',
  `effective_from` date NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `proposed_by` bigint(20) unsigned DEFAULT NULL,
  `decided_by` bigint(20) unsigned DEFAULT NULL,
  `decided_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `clinical_config_versions_config_key_version_unique` (`config_key`,`version`),
  KEY `clinical_config_versions_status_config_key_index` (`status`,`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `clinical_config_approvals` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint(20) unsigned NOT NULL,
  `admin_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `clinical_config_approvals_version_id_admin_id_unique` (`version_id`,`admin_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `clinical_config_approvals`;
DROP TABLE IF EXISTS `clinical_config_versions`;

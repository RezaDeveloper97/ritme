-- 00061_messages_v2.sql — smart messages v2: rule-based messages with levels, A/B variants, medical approval,
-- versioning and events (bloom B-N9-05, D-86).
--
-- message_rules          one rule-based message (head row). content = {a:{title,body,cta_label,cta_target},
--                        b:{…}|null, seen, confidence, contact}; conditions = [{op:"and"|"or", items:[{fact,op,value}]}];
--                        exceptions = [{fact,op,value}]. Any edit of content / rule fields bumps `version` and
--                        withdraws the medical approval (status back to draft).
-- message_rule_versions  immutable snapshot per version (history / revert reference).
-- message_events         impression / click / dismiss / goal per user and rule (no health payload; variant only).
--
-- Twin of backend/database/migrations/2026_10_11_000061_create_message_rules_tables.php. IF NOT EXISTS so a database
-- whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `message_rules` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(160) NOT NULL,
  `level` varchar(16) NOT NULL DEFAULT 'info',
  `kind` varchar(16) NOT NULL DEFAULT 'product',
  `priority` smallint(5) unsigned NOT NULL DEFAULT 1,
  `placement` varchar(32) NOT NULL DEFAULT 'smart_tip',
  `content` text NOT NULL,
  `variants_enabled` tinyint(1) NOT NULL DEFAULT 0,
  `ab_split` tinyint(3) unsigned NOT NULL DEFAULT 50,
  `conditions` text NOT NULL,
  `exceptions` text NOT NULL,
  `cap_count` smallint(5) unsigned DEFAULT NULL,
  `cap_period` varchar(8) DEFAULT NULL,
  `cooldown_days` smallint(5) unsigned NOT NULL DEFAULT 0,
  `hours_from` tinyint(3) unsigned DEFAULT NULL,
  `hours_to` tinyint(3) unsigned DEFAULT NULL,
  `starts_on` date DEFAULT NULL,
  `ends_on` date DEFAULT NULL,
  `audience_percent` tinyint(3) unsigned NOT NULL DEFAULT 100,
  `goal` varchar(120) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'draft',
  `is_active` tinyint(1) NOT NULL DEFAULT 0,
  `version` int(10) unsigned NOT NULL DEFAULT 1,
  `approved_by` bigint(20) unsigned DEFAULT NULL,
  `approved_at` timestamp NULL DEFAULT NULL,
  `created_by` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `message_rules_placement_is_active_priority_index` (`placement`,`is_active`,`priority`),
  KEY `message_rules_status_index` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `message_rule_versions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `rule_id` bigint(20) unsigned NOT NULL,
  `version` int(10) unsigned NOT NULL,
  `snapshot` text NOT NULL,
  `changed_by` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_rule_versions_rule_id_version_unique` (`rule_id`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `message_events` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `rule_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `variant` varchar(1) NOT NULL DEFAULT 'a',
  `event` varchar(12) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `message_events_rule_id_event_created_at_index` (`rule_id`,`event`,`created_at`),
  KEY `message_events_user_id_rule_id_created_at_index` (`user_id`,`rule_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `message_events`;
DROP TABLE IF EXISTS `message_rule_versions`;
DROP TABLE IF EXISTS `message_rules`;

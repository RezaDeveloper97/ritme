-- 00111_message_engine_v6.sql — message engine schema v6: versioned approval, N caps / variants / hour windows,
-- datetime schedules, topics, goal events, locked rules, event version + slot, the daily event rollup, the generic
-- alerts inbox, challenge contraindications (T-M10-02; SPEC v6 §5.3, §5.4, §5.4.1, §5.5; gap 03 §1 rows 1a–1j, 2a–2g;
-- DECISIONS.md M-01, M-02, M-03, M-06, M-07, M-13, M-16, M-17, M-18; deviation D-160). One migration carries every M10
-- message-engine table so T-M10-03…T-M10-10 add no schema.
--
-- message_rules          + rule_key          stable key of seeded and code-referenced rules (UNIQUE, NULL for
--                                            admin-made rules)
--                        + approved_version  the version the medical advisor approved; served while the head waits
--                                            for re-approval (M-03). NULL = never approved.
--                        + topics            ["fertile_window", …] (M-06)
--                        + caps              [{"count":1,"period":"day|week|cycle"}] (several caps per rule)
--                        + variants          [{"key":"a","weight":50}, …] (N weighted variants; texts in content)
--                        + starts_at/ends_at datetime schedule (user-local, M-08)
--                        + hour_windows      [[8,12],[18,22]] user-local hours
--                        + goal_event        server-matched goal from a fixed catalogue (M-17)
--                        + locked            a locked rule cannot be switched off or deleted
--                        The legacy columns (cap_count/cap_period, variants_enabled/ab_split, starts_on/ends_on,
--                        hours_from/hours_to, goal) stay and are read when the new column is NULL.
-- message_rule_versions  unchanged: `snapshot` is the admin JSON of the whole rule (every served field — content,
--                        conditions, exceptions, level, kind, topics, variants — is in it from this task on).
-- message_events         + version, + slot; variant widened to varchar(16) (N variants); event admits `eligible`
--                        (varchar(12) already); KEYs (user_id, created_at) and (created_at, rule_id) for the
--                        per-user reads and the retention / rollup sweep (M-13).
-- message_event_daily    the daily rollup kept forever (M-13): one row per (day, rule, version, variant, slot, event).
-- message_inbox          the alerts inbox (SPEC §5.4.1, M-16): open_flag is 1 for an open row (active, acknowledged,
--                        added_to_visit_note) and NULL for history (resolved, archived), so UNIQUE (user_id, rule_id,
--                        open_flag) allows at most one open row per user and message and any number of history rows.
-- challenge_programs     + contraindications (a rule, SPEC §5.5), + start_phase (on_phase_start).
--
-- Data step (M-18, goose only; the Laravel twin is schema only — production rows are not converted by Laravel):
-- daily_advice + product → content, modal + product → marketing. Every other slot/kind mismatch stays and is skipped
-- by selection with `kind_not_allowed` (T-M10-04). Down maps content → clinical (priority ≥ 100) or product, and
-- marketing → product, so the pre-v6 code never sees a kind it does not know.
--
-- Twin of backend/database/migrations/2026_10_29_000111_add_message_engine_v6_schema.php (Laravel owns the prod schema
-- until T-M2-27), so `make schema-diff` stays green. IF [NOT] EXISTS everywhere so a database whose Laravel half already
-- ran does not fail.

-- +goose Up
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `rule_key` varchar(64) DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `approved_version` int(10) unsigned DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `topics` json DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `caps` json DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `variants` json DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `starts_at` datetime DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `ends_at` datetime DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `hour_windows` json DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `goal_event` varchar(48) DEFAULT NULL;
ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `locked` tinyint(1) NOT NULL DEFAULT 0;
CREATE UNIQUE INDEX IF NOT EXISTS `message_rules_rule_key_unique` ON `message_rules` (`rule_key`);

ALTER TABLE `message_events` MODIFY `variant` varchar(16) NOT NULL DEFAULT 'a';
ALTER TABLE `message_events` ADD COLUMN IF NOT EXISTS `version` int(10) unsigned DEFAULT NULL;
ALTER TABLE `message_events` ADD COLUMN IF NOT EXISTS `slot` varchar(32) DEFAULT NULL;
CREATE INDEX IF NOT EXISTS `message_events_user_id_created_at_index` ON `message_events` (`user_id`,`created_at`);
CREATE INDEX IF NOT EXISTS `message_events_created_at_rule_id_index` ON `message_events` (`created_at`,`rule_id`);

CREATE TABLE IF NOT EXISTS `message_event_daily` (
  `day` date NOT NULL,
  `rule_id` bigint(20) unsigned NOT NULL,
  `version` int(10) unsigned NOT NULL,
  `variant` varchar(16) NOT NULL,
  `slot` varchar(32) NOT NULL,
  `event` varchar(12) NOT NULL,
  `count` int(10) unsigned NOT NULL,
  PRIMARY KEY (`day`,`rule_id`,`version`,`variant`,`slot`,`event`),
  KEY `message_event_daily_rule_id_day_index` (`rule_id`,`day`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `message_inbox` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `rule_id` bigint(20) unsigned NOT NULL,
  `version` int(10) unsigned NOT NULL,
  `rule_key` varchar(64) DEFAULT NULL,
  `slot` varchar(32) NOT NULL DEFAULT 'alerts_inbox',
  `kind` varchar(16) NOT NULL,
  `level` varchar(16) NOT NULL,
  `state` varchar(24) NOT NULL,
  `open_flag` tinyint(1) GENERATED ALWAYS AS (if(`state` in ('active','acknowledged','added_to_visit_note'),1,NULL)) STORED,
  `evidence_key` varchar(96) DEFAULT NULL,
  `hit_count` int(10) unsigned NOT NULL DEFAULT 1,
  `evidence` json DEFAULT NULL,
  `first_at` timestamp NULL DEFAULT NULL,
  `last_at` timestamp NULL DEFAULT NULL,
  `acked_at` timestamp NULL DEFAULT NULL,
  `noted_at` timestamp NULL DEFAULT NULL,
  `resolved_at` timestamp NULL DEFAULT NULL,
  `archived_at` timestamp NULL DEFAULT NULL,
  `legacy_alert_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_inbox_user_id_rule_id_open_flag_unique` (`user_id`,`rule_id`,`open_flag`),
  KEY `message_inbox_user_id_state_level_last_at_index` (`user_id`,`state`,`level`,`last_at`),
  KEY `message_inbox_state_last_at_index` (`state`,`last_at`),
  CONSTRAINT `message_inbox_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `challenge_programs` ADD COLUMN IF NOT EXISTS `contraindications` json DEFAULT NULL;
ALTER TABLE `challenge_programs` ADD COLUMN IF NOT EXISTS `start_phase` varchar(24) DEFAULT NULL;

UPDATE `message_rules` SET `kind` = 'content' WHERE `placement` = 'daily_advice' AND `kind` = 'product';
UPDATE `message_rules` SET `kind` = 'marketing' WHERE `placement` = 'modal' AND `kind` = 'product';

-- +goose Down
UPDATE `message_rules` SET `kind` = IF(`priority` >= 100, 'clinical', 'product') WHERE `kind` = 'content';
UPDATE `message_rules` SET `kind` = 'product' WHERE `kind` = 'marketing';

ALTER TABLE `challenge_programs` DROP COLUMN IF EXISTS `start_phase`;
ALTER TABLE `challenge_programs` DROP COLUMN IF EXISTS `contraindications`;

DROP TABLE IF EXISTS `message_inbox`;
DROP TABLE IF EXISTS `message_event_daily`;

DROP INDEX IF EXISTS `message_events_created_at_rule_id_index` ON `message_events`;
DROP INDEX IF EXISTS `message_events_user_id_created_at_index` ON `message_events`;
ALTER TABLE `message_events` DROP COLUMN IF EXISTS `slot`;
ALTER TABLE `message_events` DROP COLUMN IF EXISTS `version`;
-- A variant key longer than one character (N variants, written after Up) is cut to its first letter.
UPDATE `message_events` SET `variant` = LEFT(`variant`, 1) WHERE CHAR_LENGTH(`variant`) > 1;
ALTER TABLE `message_events` MODIFY `variant` varchar(1) NOT NULL DEFAULT 'a';

DROP INDEX IF EXISTS `message_rules_rule_key_unique` ON `message_rules`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `locked`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `goal_event`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `hour_windows`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `ends_at`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `starts_at`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `variants`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `caps`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `topics`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `approved_version`;
ALTER TABLE `message_rules` DROP COLUMN IF EXISTS `rule_key`;

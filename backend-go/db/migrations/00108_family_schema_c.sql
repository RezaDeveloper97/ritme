-- 00108_family_schema_c.sql — family schema v6, part C: child guardians, actor columns on child / baby-log / to-do
-- writes, the child medication dose log, and the household scope of the to-do list (T-M11-04; SPEC v6 §1.7;
-- docs/engine-v6/family.md §2, §4, §10; gap 05 §1 rows 5, 17, 18, 27, 33, §2.9; DECISIONS.md F-15, F-17, F-24, F-25,
-- F-26; deviation D-192). Schema plus the guardian backfill only: every writer and reader is a later task (T-M11-11
-- co-guardianship, T-M11-14 household to-do and the dose log).
--
-- child_guardians   who may act on a child: level write | limited (limited = the carer's sub_scopes JSON list, e.g.
--                   ["feeding","diaper","sleep"]; NULL for write), status invited | active | left | removed (left /
--                   removed rows stay for the activity trail; access needs status = 'active'), invited_by (FK users,
--                   SET NULL), link_id = the family link the guardianship came through (FK companions, SET NULL: a
--                   hard-deleted link never takes the guardianship with it; T-M11-11 decides what a revoke does).
--                   UNIQUE (child_id, user_id) = one row per person per child; KEY (user_id, status) for "my
--                   children". Both FKs to children and users cascade: a deleted child or account loses its rows.
-- created_by /      the account that wrote the row (family.md §4 "writer name on each entry"), on children,
-- updated_by        child_measurements, child_vaccine_doses, child_milestone_checks, baby_feeds, baby_sleeps,
--                   baby_diapers, todo_tasks, todo_items and baby_med_doses. FK users ON DELETE SET NULL, so a deleted
--                   account leaves the entry and shows "—". Not backfilled: existing rows have no known writer.
-- baby_med_doses    the child dose log «دارو» (F-26): name, free dose text, given_at (local wall clock, like the other
--                   baby logs), note. No schedule, reminder or dosing advice. KEY (child_id, given_at) for the timeline.
-- todo_tasks        + scope personal | household (DEFAULT 'personal': every existing task stays personal), assigned_to
--                   (FK users, SET NULL; «شیفت شب» = a household task assigned to one person, F-24). One household
--                   list per owner (user_id), never merged (F-25). KEY (user_id, scope, done_at) for the list reads.
--
-- children.owner_id keeps ON DELETE CASCADE: the frozen Laravel production code deletes accounts through it, so a sole
-- owner's child still goes with her account. T-M11-11 moves a child to a remaining guardian in Go before the user row
-- is deleted.
--
-- Backfill (idempotent INSERT IGNORE … SELECT, re-runnable): every child's owner becomes an active write guardian, and
-- the spouse of every active spouse link becomes an active write guardian of each child the owner shared with that
-- family (families / family_children, F-17), invited_by = the owner, link_id = the spouse link. A revoked or pending
-- spouse link gives nothing. Rows already present (written by v6 code) are left alone.
--
-- Twin of backend/database/migrations/2026_10_23_000108_add_family_schema_c.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables; IF [NOT]
-- EXISTS everywhere so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `child_guardians` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `child_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `level` varchar(8) NOT NULL,
  `sub_scopes` json DEFAULT NULL,
  `status` varchar(10) NOT NULL,
  `invited_by` bigint(20) unsigned DEFAULT NULL,
  `link_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `child_guardians_child_id_user_id_unique` (`child_id`,`user_id`),
  KEY `child_guardians_user_id_status_index` (`user_id`,`status`),
  KEY `child_guardians_invited_by_foreign` (`invited_by`),
  KEY `child_guardians_link_id_foreign` (`link_id`),
  CONSTRAINT `child_guardians_child_id_foreign` FOREIGN KEY (`child_id`) REFERENCES `children` (`id`) ON DELETE CASCADE,
  CONSTRAINT `child_guardians_invited_by_foreign` FOREIGN KEY (`invited_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `child_guardians_link_id_foreign` FOREIGN KEY (`link_id`) REFERENCES `companions` (`id`) ON DELETE SET NULL,
  CONSTRAINT `child_guardians_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `children`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `children_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `children_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `children_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `children_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

ALTER TABLE `child_measurements`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `child_measurements_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `child_measurements_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `child_measurements_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `child_measurements_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

ALTER TABLE `child_vaccine_doses`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `child_vaccine_doses_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `child_vaccine_doses_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `child_vaccine_doses_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `child_vaccine_doses_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

ALTER TABLE `child_milestone_checks`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `child_milestone_checks_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `child_milestone_checks_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `child_milestone_checks_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `child_milestone_checks_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

ALTER TABLE `baby_feeds`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `baby_feeds_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `baby_feeds_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `baby_feeds_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `baby_feeds_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

ALTER TABLE `baby_sleeps`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `baby_sleeps_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `baby_sleeps_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `baby_sleeps_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `baby_sleeps_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

ALTER TABLE `baby_diapers`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `baby_diapers_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `baby_diapers_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `baby_diapers_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `baby_diapers_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

ALTER TABLE `todo_tasks`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `todo_tasks_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `todo_tasks_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `todo_tasks_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `todo_tasks_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

ALTER TABLE `todo_items`
  ADD COLUMN IF NOT EXISTS `created_by` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `updated_by` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `todo_items_created_by_foreign` (`created_by`),
  ADD KEY IF NOT EXISTS `todo_items_updated_by_foreign` (`updated_by`),
  ADD CONSTRAINT `todo_items_created_by_foreign` FOREIGN KEY IF NOT EXISTS (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `todo_items_updated_by_foreign` FOREIGN KEY IF NOT EXISTS (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS `baby_med_doses` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `child_id` bigint(20) unsigned NOT NULL,
  `name` varchar(80) NOT NULL,
  `dose_text` varchar(40) DEFAULT NULL,
  `given_at` datetime NOT NULL,
  `note` varchar(200) DEFAULT NULL,
  `created_by` bigint(20) unsigned DEFAULT NULL,
  `updated_by` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `baby_med_doses_child_id_given_at_index` (`child_id`,`given_at`),
  KEY `baby_med_doses_created_by_foreign` (`created_by`),
  KEY `baby_med_doses_updated_by_foreign` (`updated_by`),
  CONSTRAINT `baby_med_doses_child_id_foreign` FOREIGN KEY (`child_id`) REFERENCES `children` (`id`) ON DELETE CASCADE,
  CONSTRAINT `baby_med_doses_created_by_foreign` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `baby_med_doses_updated_by_foreign` FOREIGN KEY (`updated_by`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `todo_tasks`
  ADD COLUMN IF NOT EXISTS `scope` varchar(10) NOT NULL DEFAULT 'personal',
  ADD COLUMN IF NOT EXISTS `assigned_to` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `todo_tasks_user_id_scope_done_at_index` (`user_id`,`scope`,`done_at`),
  ADD KEY IF NOT EXISTS `todo_tasks_assigned_to_foreign` (`assigned_to`),
  ADD CONSTRAINT `todo_tasks_assigned_to_foreign` FOREIGN KEY IF NOT EXISTS (`assigned_to`) REFERENCES `users` (`id`) ON DELETE SET NULL;

-- Backfill 1: every child's owner is an active write guardian.
INSERT IGNORE INTO `child_guardians` (`child_id`, `user_id`, `level`, `sub_scopes`, `status`, `invited_by`, `link_id`, `created_at`, `updated_at`)
SELECT c.`id`, c.`owner_id`, 'write', NULL, 'active', NULL, NULL, COALESCE(c.`created_at`, NOW()), NOW()
FROM `children` c;

-- Backfill 2 (F-17): the spouse of an active spouse link co-guards every child the owner shared with that family.
INSERT IGNORE INTO `child_guardians` (`child_id`, `user_id`, `level`, `sub_scopes`, `status`, `invited_by`, `link_id`, `created_at`, `updated_at`)
SELECT c.`id`, f.`spouse_user_id`, 'write', NULL, 'active', f.`owner_id`, f.`companion_id`, COALESCE(fc.`created_at`, NOW()), NOW()
FROM `family_children` fc
JOIN `families` f ON f.`id` = fc.`family_id`
JOIN `children` c ON c.`id` = fc.`child_id` AND c.`owner_id` = f.`owner_id`
JOIN `companions` cm ON cm.`id` = f.`companion_id`
WHERE f.`spouse_user_id` IS NOT NULL AND f.`spouse_user_id` <> f.`owner_id`
  AND cm.`type` = 'spouse' AND cm.`status` = 'active' AND cm.`companion_user_id` = f.`spouse_user_id`;

-- +goose Down
-- children.owner_id is untouched in both directions. Guardian, dose-log and household rows written after Up are lost.
ALTER TABLE `todo_tasks` DROP FOREIGN KEY IF EXISTS `todo_tasks_assigned_to_foreign`;
-- Dropping scope alone would shrink this key to a duplicate of (user_id, done_at).
ALTER TABLE `todo_tasks` DROP INDEX IF EXISTS `todo_tasks_user_id_scope_done_at_index`;
ALTER TABLE `todo_tasks` DROP COLUMN IF EXISTS `assigned_to`, DROP COLUMN IF EXISTS `scope`;

DROP TABLE IF EXISTS `baby_med_doses`;

ALTER TABLE `todo_items` DROP FOREIGN KEY IF EXISTS `todo_items_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `todo_items_created_by_foreign`;
ALTER TABLE `todo_items` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;
ALTER TABLE `todo_tasks` DROP FOREIGN KEY IF EXISTS `todo_tasks_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `todo_tasks_created_by_foreign`;
ALTER TABLE `todo_tasks` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;
ALTER TABLE `baby_diapers` DROP FOREIGN KEY IF EXISTS `baby_diapers_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `baby_diapers_created_by_foreign`;
ALTER TABLE `baby_diapers` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;
ALTER TABLE `baby_sleeps` DROP FOREIGN KEY IF EXISTS `baby_sleeps_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `baby_sleeps_created_by_foreign`;
ALTER TABLE `baby_sleeps` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;
ALTER TABLE `baby_feeds` DROP FOREIGN KEY IF EXISTS `baby_feeds_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `baby_feeds_created_by_foreign`;
ALTER TABLE `baby_feeds` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;
ALTER TABLE `child_milestone_checks` DROP FOREIGN KEY IF EXISTS `child_milestone_checks_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `child_milestone_checks_created_by_foreign`;
ALTER TABLE `child_milestone_checks` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;
ALTER TABLE `child_vaccine_doses` DROP FOREIGN KEY IF EXISTS `child_vaccine_doses_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `child_vaccine_doses_created_by_foreign`;
ALTER TABLE `child_vaccine_doses` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;
ALTER TABLE `child_measurements` DROP FOREIGN KEY IF EXISTS `child_measurements_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `child_measurements_created_by_foreign`;
ALTER TABLE `child_measurements` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;
ALTER TABLE `children` DROP FOREIGN KEY IF EXISTS `children_updated_by_foreign`, DROP FOREIGN KEY IF EXISTS `children_created_by_foreign`;
ALTER TABLE `children` DROP COLUMN IF EXISTS `updated_by`, DROP COLUMN IF EXISTS `created_by`;

DROP TABLE IF EXISTS `child_guardians`;

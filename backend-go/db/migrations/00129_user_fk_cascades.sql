-- 00129_user_fk_cascades.sql — account delete reaches every user row (T-M12-13, finding T-03).
--
-- DELETE /api/v1/account (and the admin user delete) remove the `users` row and rely on ON DELETE CASCADE for the rest.
-- These user columns had no foreign key, so a deleted account left rows behind:
--
--   message_events.user_id                 smart-message impressions / clicks / goals of the user (B-N9-05)
--   admin_health_access_requests.user_id   an admin's request to see THIS user's health data (reason + ticket)
--   lab_jobs.user_id                        lab OCR/analysis queue rows (also reachable through lab_id → lab_reports)
--   oauth_auth_codes.user_id                Passport auth codes (no Ritme flow writes them; kept safe for the future)
--   oauth_device_codes.user_id              Passport device codes (idem; NULL until approved — NULL is allowed)
--   user_profiles_dedupe_backup.user_id     profile rows removed by the 00096 dedupe (health data: birthday, weight, LMP…)
--
-- For each table, in this order:
--   1. orphan rows are deleted: rows whose user_id names no `users` row. They belong to accounts that were already
--      deleted (or never existed), so they are exactly the leftovers this task removes — never another user's data;
--   2. the FK `<table>_user_id_foreign` → users(id) ON DELETE CASCADE is added (+ the user_id index where the table
--      has none leading with user_id: admin_health_access_requests, lab_jobs).
--
-- Big-table safety (message_events grows with every message shown): the orphan delete walks the user_id index and the
-- FK is added with foreign_key_checks = 0, which lets InnoDB add it ALGORITHM=INPLACE, LOCK=NONE (metadata only, no
-- table copy, writes keep flowing). The checks are off only for these ALTERs and the rows were made valid by step 1
-- (a user deleted between steps 1 and 2 can leave at most that user's few rows; the next account delete path would
-- not reach them, so the migration is re-run safe: running it again deletes them and the ALTERs are IF NOT EXISTS).
--
-- Deliberately NOT given a cascade here (docs/quality-2026-11/user-tables.md, the allow-list the schema test reads):
-- oauth_access_tokens (deleted by hand in profile.Service.DeleteAccount), sessions (holds admin ids under Laravel's
-- `admin` guard), oauth_clients (polymorphic owner), ivf_cycles.active_user_id (same row as ivf_cycles.user_id),
-- the SET NULL columns (anonymised on purpose) and plus_admin_actions (billing ledger: pending a user decision).
--
-- The DDL goes through PREPARE so sqlc (which parses the Up sections) never sees MariaDB-only syntax.
-- Laravel twin: backend/database/migrations/2026_11_18_000129_add_user_fk_cascades.php (same statements).
-- Test: internal/profile/delete_complete_int_test.go (schema + behaviour), `make test-int PKG=./internal/profile/...`.

-- +goose Up
DELETE x FROM `message_events` x LEFT JOIN `users` u ON u.`id` = x.`user_id` WHERE u.`id` IS NULL;
DELETE x FROM `admin_health_access_requests` x LEFT JOIN `users` u ON u.`id` = x.`user_id` WHERE u.`id` IS NULL;
DELETE x FROM `lab_jobs` x LEFT JOIN `users` u ON u.`id` = x.`user_id` WHERE u.`id` IS NULL;
DELETE x FROM `oauth_auth_codes` x LEFT JOIN `users` u ON u.`id` = x.`user_id` WHERE u.`id` IS NULL;
DELETE x FROM `oauth_device_codes` x LEFT JOIN `users` u ON u.`id` = x.`user_id` WHERE x.`user_id` IS NOT NULL AND u.`id` IS NULL;
DELETE x FROM `user_profiles_dedupe_backup` x LEFT JOIN `users` u ON u.`id` = x.`user_id` WHERE u.`id` IS NULL;

SET SESSION foreign_key_checks = 0;
PREPARE `fk` FROM 'ALTER TABLE `message_events`
  ADD CONSTRAINT `message_events_user_id_foreign` FOREIGN KEY IF NOT EXISTS (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  ALGORITHM=INPLACE, LOCK=NONE';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `admin_health_access_requests`
  ADD INDEX IF NOT EXISTS `admin_health_access_requests_user_id_index` (`user_id`),
  ADD CONSTRAINT `admin_health_access_requests_user_id_foreign` FOREIGN KEY IF NOT EXISTS (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  ALGORITHM=INPLACE, LOCK=NONE';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `lab_jobs`
  ADD INDEX IF NOT EXISTS `lab_jobs_user_id_index` (`user_id`),
  ADD CONSTRAINT `lab_jobs_user_id_foreign` FOREIGN KEY IF NOT EXISTS (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  ALGORITHM=INPLACE, LOCK=NONE';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `oauth_auth_codes`
  ADD CONSTRAINT `oauth_auth_codes_user_id_foreign` FOREIGN KEY IF NOT EXISTS (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  ALGORITHM=INPLACE, LOCK=NONE';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `oauth_device_codes`
  ADD CONSTRAINT `oauth_device_codes_user_id_foreign` FOREIGN KEY IF NOT EXISTS (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  ALGORITHM=INPLACE, LOCK=NONE';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `user_profiles_dedupe_backup`
  ADD CONSTRAINT `user_profiles_dedupe_backup_user_id_foreign` FOREIGN KEY IF NOT EXISTS (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  ALGORITHM=INPLACE, LOCK=NONE';
EXECUTE `fk`;
DEALLOCATE PREPARE `fk`;
SET SESSION foreign_key_checks = 1;

-- +goose Down
-- Drops the keys only; the orphan rows deleted by Up are gone for good (they were a deleted account's data).
PREPARE `fk` FROM 'ALTER TABLE `message_events` DROP FOREIGN KEY IF EXISTS `message_events_user_id_foreign`';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `admin_health_access_requests` DROP FOREIGN KEY IF EXISTS `admin_health_access_requests_user_id_foreign`';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `admin_health_access_requests` DROP INDEX IF EXISTS `admin_health_access_requests_user_id_index`';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `lab_jobs` DROP FOREIGN KEY IF EXISTS `lab_jobs_user_id_foreign`';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `lab_jobs` DROP INDEX IF EXISTS `lab_jobs_user_id_index`';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `oauth_auth_codes` DROP FOREIGN KEY IF EXISTS `oauth_auth_codes_user_id_foreign`';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `oauth_device_codes` DROP FOREIGN KEY IF EXISTS `oauth_device_codes_user_id_foreign`';
EXECUTE `fk`;
PREPARE `fk` FROM 'ALTER TABLE `user_profiles_dedupe_backup` DROP FOREIGN KEY IF EXISTS `user_profiles_dedupe_backup_user_id_foreign`';
EXECUTE `fk`;
DEALLOCATE PREPARE `fk`;

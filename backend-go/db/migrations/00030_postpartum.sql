-- 00030_postpartum.sql — postpartum mode (bloom B-N5-01, artboards nbl_v15_Main / _Recovery / _MoodCheck): the birth the
-- mode counts from and the EPDS mood checks. Domain logic: internal/postpartum. 00027–00029 belong to parallel tasks.
--
-- Built on what exists, nothing duplicated:
--   * the mode itself is bloom's user_life_profiles.life_mode = 'postpartum' (B-N2-01);
--   * the recovery log (lochia amount / colour, pain & location, breasts, sleep, feeds count) is log taxonomy v2
--     slots in health_log_entries (B-N3-01; internal/healthlog/taxonomy) — no recovery table;
--   * a pregnancy that ends in a birth is closed through pregnancy_profiles.pregnancy_mode = 0 (like
--     /pregnancy/deactivate); "delivered" is recorded here (source = pregnancy), a loss is CB-LOSS-01's own table.
--
-- postpartum_profiles   one row per user (re-activation for a later birth overwrites it): birth_date, delivery_type
--                       vaginal|cesarean (NULL = not told), baby_count 1–4, source pregnancy|direct, the closed
--                       pregnancy_profiles row (SET NULL) and when it was closed.
-- epds_checks           one Edinburgh Postnatal Depression Scale check per (user, kind, day): kind short (EPDS-3,
--                       items 3–5, weekly) | full (10 items, every 2 weeks); answers = JSON {item code: score 0–3};
--                       total, self_harm (item 10 score, full only) and urgent (item 10 > 0 or total ≥ 13) are
--                       computed by the API and stored. Sensitive health data: never logged, never in analytics,
--                       not shared with companions (no grant section reads it).
--
-- Codes are varchar validated in Go. Twin of backend/database/migrations/2026_10_02_000030_create_postpartum_tables.php
-- (Laravel owns the prod schema until T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints
-- the Laravel-built tables; IF NOT EXISTS so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `postpartum_profiles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `birth_date` date NOT NULL,
  `delivery_type` varchar(16) DEFAULT NULL,
  `baby_count` tinyint(3) unsigned NOT NULL DEFAULT 1,
  `source` varchar(16) NOT NULL DEFAULT 'direct',
  `pregnancy_profile_id` bigint(20) unsigned DEFAULT NULL,
  `pregnancy_closed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `postpartum_profiles_user_id_unique` (`user_id`),
  KEY `postpartum_profiles_pregnancy_profile_id_foreign` (`pregnancy_profile_id`),
  CONSTRAINT `postpartum_profiles_pregnancy_profile_id_foreign` FOREIGN KEY (`pregnancy_profile_id`) REFERENCES `pregnancy_profiles` (`id`) ON DELETE SET NULL,
  CONSTRAINT `postpartum_profiles_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `epds_checks` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(8) NOT NULL,
  `taken_on` date NOT NULL,
  `answers` json NOT NULL,
  `total` tinyint(3) unsigned NOT NULL,
  `self_harm` tinyint(3) unsigned DEFAULT NULL,
  `urgent` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `epds_checks_user_id_kind_taken_on_unique` (`user_id`,`kind`,`taken_on`),
  KEY `epds_checks_user_id_taken_on_index` (`user_id`,`taken_on`),
  CONSTRAINT `epds_checks_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `epds_checks`;
DROP TABLE IF EXISTS `postpartum_profiles`;

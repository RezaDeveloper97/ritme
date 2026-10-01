-- 00014_life_profiles.sql — profile & onboarding schema v2 and the six life-stage modes (B-N2-01, bloom/N2).
--
-- `user_life_profiles` — one row per user, written by the v2 onboarding steps (Gender, Goal, Meno, Conditions) and
-- the life-stage switch. Everything the legacy profile already holds stays in `user_profiles` (name on `users`,
-- birthday / height / weight, last period, period / cycle length, user_goal), so existing rows are untouched:
--
--   gender               female | male (NULL = not asked yet)
--   life_mode            cycle | ttc | pregnancy | postpartum | menopause | teen. NULL = the legacy derivation
--                        (active pregnancy profile → pregnancy, user_goal ttc → ttc, else cycle), which is what every
--                        user created before this migration gets — no backfill, no data change.
--   ivf_iui              TTC with IVF/IUI treatment (mode screen switch; roadmap E03-ivf builds on it)
--   track_contraception  «روش پیشگیری را هم پیگیری کن» (mode screen; roadmap E06-contra builds on it)
--   chronic_illnesses    JSON list (diabetes, hypertension, thyroid, asthma, anemia, migraine, other); [] = «هیچ‌کدام»,
--   gyn_conditions       JSON list (pcos, endometriosis, fibroids, recurrent_infections, other);    NULL = skipped
--   medications          JSON list (contraceptive_pill, iud, hormonal_medication)
--   menopause_stage      peri | meno | post | unsure; menopause_last_period = approximate (first of the month),
--   menopause_surgical / menopause_hrt = yes/no (NULL = unanswered). Roadmap E02-meno reuses these columns.
--   onboarding_started_at / onboarding_completed_at — v2 onboarding progress; while started and not completed,
--                        auth's `profile_completed` is false (legacy users: both NULL → legacy rule unchanged).
--
-- Twin of backend/database/migrations/2026_10_01_000005_create_user_life_profiles_table.php (Laravel owns the prod
-- schema until T-M2-27; schema only), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the
-- Laravel-built table. IF NOT EXISTS: a database whose Laravel half already ran must not fail here.

-- +goose Up
CREATE TABLE IF NOT EXISTS `user_life_profiles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `gender` varchar(16) DEFAULT NULL,
  `life_mode` varchar(32) DEFAULT NULL,
  `ivf_iui` tinyint(1) NOT NULL DEFAULT 0,
  `track_contraception` tinyint(1) NOT NULL DEFAULT 0,
  `chronic_illnesses` json DEFAULT NULL,
  `gyn_conditions` json DEFAULT NULL,
  `medications` json DEFAULT NULL,
  `menopause_stage` varchar(16) DEFAULT NULL,
  `menopause_last_period` date DEFAULT NULL,
  `menopause_surgical` tinyint(1) DEFAULT NULL,
  `menopause_hrt` tinyint(1) DEFAULT NULL,
  `onboarding_started_at` timestamp NULL DEFAULT NULL,
  `onboarding_completed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_life_profiles_user_id_unique` (`user_id`),
  CONSTRAINT `user_life_profiles_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `user_life_profiles`;

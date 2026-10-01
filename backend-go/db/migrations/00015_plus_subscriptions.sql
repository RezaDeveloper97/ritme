-- 00015_plus_subscriptions.sql — Ritme Plus subscription domain (B-N2-04, bloom/N2).
--
-- Money: every amount is an integer number of **rials** (IRR, `*_rials` bigint unsigned) — the unit Iranian bank
-- gateways settle in. Clients show toman (= rials / 10). VAT is a per-invoice snapshot in basis points
-- (`vat_rate_bps`, 1000 = 10 %) taken from config (PLUS_VAT_RATE_BPS) at checkout.
--
--   plus_plans                admin data (B-N2-09 CRUD). title/badge are translatable JSON keyed by language code;
--                             monthly_display_rials NULL = price / duration (the «تومان / ماه» line).
--   plus_trials               one row per user, ever (UNIQUE user_id = the double-trial / race guard).
--   plus_discount_codes       percent (value 1–100) or amount (value in rials); max_redemptions / per_user_limit NULL =
--                             unlimited; plan_ids JSON list NULL = every plan; starts_at / expires_at window.
--                             A redemption is an invoice that carries the code and is paid, or pending and not expired
--                             (counted under a row lock at checkout) — no separate counter to drift.
--   plus_invoices             one checkout. reference = public id (random, user-scoped). status pending | paid |
--                             failed | expired | refunded. authority = the gateway's payment id (UNIQUE).
--   plus_receipts             a verified payment (UNIQUE invoice_id; UNIQUE gateway+ref_id stops a replayed bank
--                             reference from paying a second invoice).
--   plus_subscriptions        an entitlement period. status active | canceled (auto-renew off, still valid until
--                             ends_at) | refunded; "expired" is derived from ends_at. source purchase | admin.
--   plus_usage_counters       per user, feature key and month (period_start = first day, Tehran) for quotas
--                             (plus.lab_ai 10/month, …) and the trial sheet's usage lines (B-N2-06).
--
-- Seed: the three plans of nbl_Prem_Plans (1 / 3 / 6 months: 99,000 / 237,000 / 390,000 toman) — admin data from
-- here on; INSERT IGNORE never overwrites an admin edit.
--
-- Twin of backend/database/migrations/2026_10_01_000006_create_plus_tables.php (Laravel owns the prod schema until
-- T-M2-27; schema + the same seed rows), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the
-- Laravel-built tables. IF NOT EXISTS: a database whose Laravel half already ran must not fail here.

-- +goose Up
CREATE TABLE IF NOT EXISTS `plus_plans` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `title` json NOT NULL,
  `badge` json DEFAULT NULL,
  `duration_months` smallint(5) unsigned NOT NULL,
  `price_rials` bigint(20) unsigned NOT NULL,
  `monthly_display_rials` bigint(20) unsigned DEFAULT NULL,
  `is_highlighted` tinyint(1) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(10) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_plans_code_unique` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plus_trials` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `started_at` datetime NOT NULL,
  `ends_at` datetime NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_trials_user_id_unique` (`user_id`),
  CONSTRAINT `plus_trials_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plus_discount_codes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `kind` varchar(16) NOT NULL,
  `value` bigint(20) unsigned NOT NULL,
  `max_redemptions` int(10) unsigned DEFAULT NULL,
  `per_user_limit` int(10) unsigned DEFAULT 1,
  `plan_ids` json DEFAULT NULL,
  `starts_at` datetime DEFAULT NULL,
  `expires_at` datetime DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_discount_codes_code_unique` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plus_invoices` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `reference` varchar(32) NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `plan_id` bigint(20) unsigned DEFAULT NULL,
  `duration_months` smallint(5) unsigned NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `currency` varchar(3) NOT NULL DEFAULT 'IRR',
  `subtotal_rials` bigint(20) unsigned NOT NULL,
  `discount_rials` bigint(20) unsigned NOT NULL DEFAULT 0,
  `vat_rate_bps` int(10) unsigned NOT NULL,
  `vat_rials` bigint(20) unsigned NOT NULL,
  `total_rials` bigint(20) unsigned NOT NULL,
  `discount_code_id` bigint(20) unsigned DEFAULT NULL,
  `discount_code` varchar(64) DEFAULT NULL,
  `gateway` varchar(32) DEFAULT NULL,
  `authority` varchar(191) DEFAULT NULL,
  `expires_at` datetime NOT NULL,
  `paid_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_invoices_reference_unique` (`reference`),
  UNIQUE KEY `plus_invoices_authority_unique` (`authority`),
  KEY `plus_invoices_plan_id_foreign` (`plan_id`),
  KEY `plus_invoices_user_id_status_index` (`user_id`,`status`),
  KEY `plus_invoices_discount_code_id_status_index` (`discount_code_id`,`status`),
  CONSTRAINT `plus_invoices_discount_code_id_foreign` FOREIGN KEY (`discount_code_id`) REFERENCES `plus_discount_codes` (`id`) ON DELETE SET NULL,
  CONSTRAINT `plus_invoices_plan_id_foreign` FOREIGN KEY (`plan_id`) REFERENCES `plus_plans` (`id`) ON DELETE SET NULL,
  CONSTRAINT `plus_invoices_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plus_receipts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `invoice_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `gateway` varchar(32) NOT NULL,
  `ref_id` varchar(191) NOT NULL,
  `card_pan` varchar(32) DEFAULT NULL,
  `amount_rials` bigint(20) unsigned NOT NULL,
  `paid_at` datetime NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_receipts_gateway_ref_id_unique` (`gateway`,`ref_id`),
  UNIQUE KEY `plus_receipts_invoice_id_unique` (`invoice_id`),
  KEY `plus_receipts_user_id_foreign` (`user_id`),
  CONSTRAINT `plus_receipts_invoice_id_foreign` FOREIGN KEY (`invoice_id`) REFERENCES `plus_invoices` (`id`) ON DELETE CASCADE,
  CONSTRAINT `plus_receipts_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plus_subscriptions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `plan_id` bigint(20) unsigned DEFAULT NULL,
  `invoice_id` bigint(20) unsigned DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'active',
  `source` varchar(16) NOT NULL DEFAULT 'purchase',
  `starts_at` datetime NOT NULL,
  `ends_at` datetime NOT NULL,
  `auto_renew` tinyint(1) NOT NULL DEFAULT 1,
  `canceled_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_subscriptions_invoice_id_unique` (`invoice_id`),
  KEY `plus_subscriptions_plan_id_foreign` (`plan_id`),
  KEY `plus_subscriptions_user_id_ends_at_index` (`user_id`,`ends_at`),
  CONSTRAINT `plus_subscriptions_invoice_id_foreign` FOREIGN KEY (`invoice_id`) REFERENCES `plus_invoices` (`id`) ON DELETE SET NULL,
  CONSTRAINT `plus_subscriptions_plan_id_foreign` FOREIGN KEY (`plan_id`) REFERENCES `plus_plans` (`id`) ON DELETE SET NULL,
  CONSTRAINT `plus_subscriptions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plus_usage_counters` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `feature` varchar(64) NOT NULL,
  `period_start` date NOT NULL,
  `used` int(10) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_usage_counters_user_id_feature_period_start_unique` (`user_id`,`feature`,`period_start`),
  CONSTRAINT `plus_usage_counters_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Plan seed (nbl_Prem_Plans): prices in rials; the 3-month plan is the highlighted «محبوب‌ترین» one.
INSERT IGNORE INTO `plus_plans` (`code`, `title`, `badge`, `duration_months`, `price_rials`, `monthly_display_rials`, `is_highlighted`, `is_active`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('plus_1m', '{"fa":"۱ ماهه","en":"1 month"}', NULL, 1, 990000, NULL, 0, 1, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('plus_3m', '{"fa":"۳ ماهه","en":"3 months"}', '{"fa":"محبوب‌ترین","en":"Most popular"}', 3, 2370000, NULL, 1, 1, 2,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('plus_6m', '{"fa":"۶ ماهه","en":"6 months"}', NULL, 6, 3900000, NULL, 0, 1, 3,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DROP TABLE IF EXISTS `plus_usage_counters`;
DROP TABLE IF EXISTS `plus_subscriptions`;
DROP TABLE IF EXISTS `plus_receipts`;
DROP TABLE IF EXISTS `plus_invoices`;
DROP TABLE IF EXISTS `plus_discount_codes`;
DROP TABLE IF EXISTS `plus_trials`;
DROP TABLE IF EXISTS `plus_plans`;

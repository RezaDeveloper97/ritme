-- 00125_ai_credits.sql — AI credits: every AI feature spends «اعتبار» (credits) instead of a monthly use count.
--
-- Rules (owner's decisions, 2026-11-12):
--   * a subscription period (or the trial) carries credits; they are usable only while that period runs and are
--     gone with it. The next period starts from its own amount again;
--   * extra credits (bought packs, admin gifts) never expire and are never zeroed — but nothing can be spent
--     without a running subscription or trial: they wait, untouched, for the next one;
--   * spending takes the period's credits first, then the extra ones;
--   * one use of a feature costs a fixed number of credits (ai_features.credit_cost), whatever the token count.
--
--   plus_plans.ai_credits     credits of one purchased period of the plan (the whole period, not per month).
--   plus_invoices.kind        `plan` (a subscription period) | `credit_pack` (credit_pack_id + the `credits`
--                             snapshot it grants). One checkout / verify / receipt path for both.
--   ai_models                 the model catalogue: provider (`gemini` | `arvan`) + the name sent upstream, what it
--                             can do (capabilities: chat, vision, audio, json) and what it costs us (milli-USD per
--                             1M tokens — the cost estimate of ai_usage_logs; never shown to users).
--   ai_features               one row per AI feature slot (the key is declared in code, internal/ai): which model
--                             serves it, the fallback model, and how many credits one use costs.
--   ai_credit_packs           purchasable extra credit (price in rials, like plus_plans).
--   ai_credit_grants          a bucket of credits. kind `plan` (subscription_id) and `trial` (trial_id) are valid
--                             while that row's period covers now — read by JOIN, never copied, so an admin extend
--                             or refund needs no second write and there is no expiry job. kind `extra` (invoice_id
--                             of the pack) and `admin` (a manual gift) have no end. UNIQUE on each source makes
--                             granting idempotent.
--   ai_credit_spends          one row per reserved use: who, which feature, how many credits and from which
--                             grants (`allocations` JSON [{grant_id, credits}]) so a refund returns them exactly.
--                             status `spent` | `refunded`.
--
-- Settings (plus_settings): trial_ai_credits — credits of the trial (seeded 20).
-- Seed: plan credits (1m 100, 3m 300, 6m 600, mtest 10), the Gemini model the platform already used, the four
-- existing feature slots and three packs. All admin data from here on (INSERT IGNORE / guarded UPDATE).
--
-- Laravel twin: 2026_11_12_000125_create_ai_credits_tables.php (schema + the same seed rows).

-- +goose Up
ALTER TABLE `plus_plans` ADD COLUMN IF NOT EXISTS `ai_credits` int(10) unsigned NOT NULL DEFAULT 0 AFTER `monthly_display_rials`;

CREATE TABLE IF NOT EXISTS `ai_models` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `provider` varchar(16) NOT NULL,
  `upstream_model` varchar(128) NOT NULL,
  `title` varchar(128) NOT NULL,
  `capabilities` json NOT NULL,
  `input_price_milli` int(10) unsigned NOT NULL DEFAULT 0,
  `output_price_milli` int(10) unsigned NOT NULL DEFAULT 0,
  `audio_price_milli` int(10) unsigned NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ai_models_provider_upstream_model_unique` (`provider`,`upstream_model`),
  UNIQUE KEY `ai_models_code_unique` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ai_features` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(64) NOT NULL,
  `title` json NOT NULL,
  `model_id` bigint(20) unsigned DEFAULT NULL,
  `fallback_model_id` bigint(20) unsigned DEFAULT NULL,
  `credit_cost` int(10) unsigned NOT NULL DEFAULT 1,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(10) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ai_features_key_unique` (`key`),
  KEY `ai_features_model_id_foreign` (`model_id`),
  KEY `ai_features_fallback_model_id_foreign` (`fallback_model_id`),
  CONSTRAINT `ai_features_fallback_model_id_foreign` FOREIGN KEY (`fallback_model_id`) REFERENCES `ai_models` (`id`) ON DELETE SET NULL,
  CONSTRAINT `ai_features_model_id_foreign` FOREIGN KEY (`model_id`) REFERENCES `ai_models` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ai_credit_packs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `title` json NOT NULL,
  `badge` json DEFAULT NULL,
  `credits` int(10) unsigned NOT NULL,
  `price_rials` bigint(20) unsigned NOT NULL,
  `is_highlighted` tinyint(1) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(10) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ai_credit_packs_code_unique` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `plus_invoices` ADD COLUMN IF NOT EXISTS `kind` varchar(16) NOT NULL DEFAULT 'plan' AFTER `plan_id`;
ALTER TABLE `plus_invoices` ADD COLUMN IF NOT EXISTS `credit_pack_id` bigint(20) unsigned DEFAULT NULL AFTER `kind`;
ALTER TABLE `plus_invoices` ADD COLUMN IF NOT EXISTS `credits` int(10) unsigned DEFAULT NULL AFTER `credit_pack_id`;
ALTER TABLE `plus_invoices` ADD KEY IF NOT EXISTS `plus_invoices_credit_pack_id_foreign` (`credit_pack_id`);
ALTER TABLE `plus_invoices` ADD CONSTRAINT `plus_invoices_credit_pack_id_foreign` FOREIGN KEY IF NOT EXISTS (`credit_pack_id`) REFERENCES `ai_credit_packs` (`id`) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS `ai_credit_grants` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(16) NOT NULL,
  `subscription_id` bigint(20) unsigned DEFAULT NULL,
  `trial_id` bigint(20) unsigned DEFAULT NULL,
  `invoice_id` bigint(20) unsigned DEFAULT NULL,
  `amount` int(10) unsigned NOT NULL,
  `remaining` int(10) unsigned NOT NULL,
  `note` varchar(255) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ai_credit_grants_subscription_id_unique` (`subscription_id`),
  UNIQUE KEY `ai_credit_grants_trial_id_unique` (`trial_id`),
  UNIQUE KEY `ai_credit_grants_invoice_id_unique` (`invoice_id`),
  KEY `ai_credit_grants_user_id_kind_index` (`user_id`,`kind`),
  CONSTRAINT `ai_credit_grants_invoice_id_foreign` FOREIGN KEY (`invoice_id`) REFERENCES `plus_invoices` (`id`) ON DELETE SET NULL,
  CONSTRAINT `ai_credit_grants_subscription_id_foreign` FOREIGN KEY (`subscription_id`) REFERENCES `plus_subscriptions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ai_credit_grants_trial_id_foreign` FOREIGN KEY (`trial_id`) REFERENCES `plus_trials` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ai_credit_grants_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ai_credit_spends` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `feature` varchar(64) NOT NULL,
  `credits` int(10) unsigned NOT NULL,
  `plan_credits` int(10) unsigned NOT NULL DEFAULT 0,
  `extra_credits` int(10) unsigned NOT NULL DEFAULT 0,
  `allocations` json NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'spent',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `ai_credit_spends_user_id_created_at_index` (`user_id`,`created_at`),
  KEY `ai_credit_spends_feature_created_at_index` (`feature`,`created_at`),
  CONSTRAINT `ai_credit_spends_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seed. Plan credits only where the admin has not set any yet.
UPDATE `plus_plans` SET `ai_credits` = 100 WHERE `code` = 'plus_1m' AND `ai_credits` = 0;
UPDATE `plus_plans` SET `ai_credits` = 300 WHERE `code` = 'plus_3m' AND `ai_credits` = 0;
UPDATE `plus_plans` SET `ai_credits` = 600 WHERE `code` = 'plus_6m' AND `ai_credits` = 0;
UPDATE `plus_plans` SET `ai_credits` = 10 WHERE `code` = 'mtest' AND `ai_credits` = 0;

INSERT IGNORE INTO `plus_settings` (`key`, `value`, `created_at`, `updated_at`) VALUES
  ('trial_ai_credits', '20', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

INSERT IGNORE INTO `ai_models` (`code`, `provider`, `upstream_model`, `title`, `capabilities`, `input_price_milli`, `output_price_milli`, `audio_price_milli`, `is_active`, `created_at`, `updated_at`) VALUES
  ('gemini-flash', 'gemini', 'gemini-flash-latest', 'Gemini Flash', '["chat","vision","audio","json"]', 300, 2500, 1000, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

INSERT IGNORE INTO `ai_features` (`key`, `title`, `model_id`, `fallback_model_id`, `credit_cost`, `is_active`, `sort_order`, `created_at`, `updated_at`)
  SELECT f.`key`, f.`title`, m.`id`, NULL, f.`credit_cost`, 1, f.`sort_order`,
         CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
  FROM (
    SELECT 'assistant' AS `key`, '{"fa":"دستیار سلامت","en":"Health assistant"}' AS `title`, 1 AS `credit_cost`, 1 AS `sort_order`
    UNION ALL SELECT 'voice_log', '{"fa":"ثبت صوتی","en":"Voice logging"}', 1, 2
    UNION ALL SELECT 'lab_analysis', '{"fa":"تحلیل آزمایش","en":"Lab analysis"}', 5, 3
    UNION ALL SELECT 'doc_extract', '{"fa":"خواندن مدارک","en":"Document reading"}', 3, 4
  ) f LEFT JOIN `ai_models` m ON m.`code` = 'gemini-flash';

INSERT IGNORE INTO `ai_credit_packs` (`code`, `title`, `badge`, `credits`, `price_rials`, `is_highlighted`, `is_active`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('credits_50', '{"fa":"۵۰ اعتبار","en":"50 credits"}', NULL, 50, 490000, 0, 1, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('credits_150', '{"fa":"۱۵۰ اعتبار","en":"150 credits"}', '{"fa":"محبوب‌ترین","en":"Most popular"}', 150, 1290000, 1, 1, 2,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('credits_400', '{"fa":"۴۰۰ اعتبار","en":"400 credits"}', NULL, 400, 2990000, 0, 1, 3,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DROP TABLE IF EXISTS `ai_credit_spends`;
DROP TABLE IF EXISTS `ai_credit_grants`;
ALTER TABLE `plus_invoices` DROP FOREIGN KEY IF EXISTS `plus_invoices_credit_pack_id_foreign`;
ALTER TABLE `plus_invoices` DROP KEY IF EXISTS `plus_invoices_credit_pack_id_foreign`;
ALTER TABLE `plus_invoices` DROP COLUMN IF EXISTS `credits`;
ALTER TABLE `plus_invoices` DROP COLUMN IF EXISTS `credit_pack_id`;
ALTER TABLE `plus_invoices` DROP COLUMN IF EXISTS `kind`;
DROP TABLE IF EXISTS `ai_credit_packs`;
DROP TABLE IF EXISTS `ai_features`;
DROP TABLE IF EXISTS `ai_models`;
DELETE FROM `plus_settings` WHERE `key` = 'trial_ai_credits';
ALTER TABLE `plus_plans` DROP COLUMN IF EXISTS `ai_credits`;

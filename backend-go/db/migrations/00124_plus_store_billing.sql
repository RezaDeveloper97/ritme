-- 00124_plus_store_billing.sql — Ritme Plus through the Android stores (Café Bazaar, Myket) next to the web gateway
-- (Zarinpal). The web keeps checkout → gateway → verify; a store build buys through the store's in-app billing and
-- the server checks the store-signed purchase (RSA) before it grants anything.
--
--   plus_store_products   store SKU → what it grants: kind `plan` (plan_id) or `trial` (the one Plus trial,
--                         plus_trials). UNIQUE(store, sku). The client never names a plan or a price.
--   plus_store_purchases  one row per store purchase token: who claimed it (a token pays one account only), its SKU
--                         and the latest purchase_time seen (a renewal is a newer purchase_time).
--   plus_plans.duration_minutes / plus_invoices.duration_minutes  NULL = duration_months applies; set only for
--                         developer test plans (`mtest`, 5 minutes).
--
-- Seed: the `mtest` plan (20,000 rials, 5 minutes) and the Bazaar SKUs trial_subscription (trial) and mtest.
--
-- Laravel twin: 2026_11_12_000124_create_plus_store_billing_tables.php (schema + the same seed rows).

-- +goose Up
ALTER TABLE `plus_plans` ADD COLUMN IF NOT EXISTS `duration_minutes` int(10) unsigned DEFAULT NULL AFTER `duration_months`;
ALTER TABLE `plus_invoices` ADD COLUMN IF NOT EXISTS `duration_minutes` int(10) unsigned DEFAULT NULL AFTER `duration_months`;

CREATE TABLE IF NOT EXISTS `plus_store_products` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `store` varchar(16) NOT NULL,
  `sku` varchar(100) NOT NULL,
  `kind` varchar(16) NOT NULL,
  `plan_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_store_products_store_sku_unique` (`store`,`sku`),
  KEY `plus_store_products_plan_id_foreign` (`plan_id`),
  CONSTRAINT `plus_store_products_plan_id_foreign` FOREIGN KEY (`plan_id`) REFERENCES `plus_plans` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plus_store_purchases` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `store` varchar(16) NOT NULL,
  `sku` varchar(100) NOT NULL,
  `purchase_token` varchar(191) NOT NULL,
  `order_id` varchar(191) DEFAULT NULL,
  `purchase_time` datetime NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `plus_store_purchases_store_purchase_token_unique` (`store`,`purchase_token`),
  KEY `plus_store_purchases_user_id_foreign` (`user_id`),
  CONSTRAINT `plus_store_purchases_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `plus_plans` (`code`, `title`, `badge`, `duration_months`, `duration_minutes`, `price_rials`, `monthly_display_rials`, `is_highlighted`, `is_active`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('mtest', '{"fa":"تست ۵ دقیقه‌ای","en":"5-minute test"}', NULL, 1, 5, 20000, NULL, 0, 1, 4,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

INSERT IGNORE INTO `plus_store_products` (`store`, `sku`, `kind`, `plan_id`, `created_at`, `updated_at`) VALUES
  ('bazaar', 'trial_subscription', 'trial', NULL,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));
INSERT IGNORE INTO `plus_store_products` (`store`, `sku`, `kind`, `plan_id`, `created_at`, `updated_at`)
  SELECT 'bazaar', 'mtest', 'plan', `id`,
         CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
  FROM `plus_plans` WHERE `code` = 'mtest';

-- +goose Down
DROP TABLE IF EXISTS `plus_store_purchases`;
DROP TABLE IF EXISTS `plus_store_products`;
DELETE FROM `plus_plans` WHERE `code` = 'mtest' AND NOT EXISTS (SELECT 1 FROM `plus_invoices` WHERE `plan_id` = `plus_plans`.`id`);
ALTER TABLE `plus_invoices` DROP COLUMN IF EXISTS `duration_minutes`;
ALTER TABLE `plus_plans` DROP COLUMN IF EXISTS `duration_minutes`;

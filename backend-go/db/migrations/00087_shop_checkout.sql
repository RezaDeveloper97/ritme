-- 00087_shop_checkout.sql — what the cart / COD checkout API (CB-SHOP-03) needs on top of 00080:
--
-- shop_discount_codes   admin-defined discount codes (CB-SHOP-05 edits them): percent or fixed toman amount, optional
--                       cap, minimum items total, global and per-user use limits, optional validity window.
--                       `code` uses the table's case-insensitive collation, so «Welcome» and «WELCOME» are one code.
-- shop_carts   +discount_code_id   the code the user applied to her cart (re-validated at every read and at checkout).
-- shop_orders  +discount_code_id   the code an order used (per-user limit, released on cancel).
--              +idempotency_key    client-generated key of the checkout request; UNIQUE per user, so a retried or
--                                  double-tapped «ثبت سفارش» returns the first order instead of placing a second one.
--
-- No health data. Twin of backend/database/migrations/2026_10_11_000087_add_shop_checkout_tables.php.

-- +goose Up
CREATE TABLE IF NOT EXISTS `shop_discount_codes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(32) NOT NULL,
  `kind` varchar(10) NOT NULL DEFAULT 'percent',
  `value` bigint(20) unsigned NOT NULL,
  `max_discount` bigint(20) unsigned DEFAULT NULL,
  `min_items_total` bigint(20) unsigned NOT NULL DEFAULT 0,
  `max_uses` int(10) unsigned DEFAULT NULL,
  `used_count` int(10) unsigned NOT NULL DEFAULT 0,
  `per_user_limit` smallint(5) unsigned DEFAULT 1,
  `starts_at` timestamp NULL DEFAULT NULL,
  `ends_at` timestamp NULL DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_discount_codes_code_unique` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `shop_carts`
  ADD COLUMN IF NOT EXISTS `discount_code_id` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `shop_carts_discount_code_id_foreign` (`discount_code_id`),
  ADD CONSTRAINT `shop_carts_discount_code_id_foreign` FOREIGN KEY IF NOT EXISTS (`discount_code_id`) REFERENCES `shop_discount_codes` (`id`) ON DELETE SET NULL;

ALTER TABLE `shop_orders`
  ADD COLUMN IF NOT EXISTS `discount_code_id` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `idempotency_key` varchar(64) DEFAULT NULL,
  ADD KEY IF NOT EXISTS `shop_orders_discount_code_id_foreign` (`discount_code_id`),
  ADD CONSTRAINT `shop_orders_discount_code_id_foreign` FOREIGN KEY IF NOT EXISTS (`discount_code_id`) REFERENCES `shop_discount_codes` (`id`) ON DELETE SET NULL;

-- A separate statement: sqlc's parser rejects `ADD UNIQUE KEY IF NOT EXISTS` inside ALTER TABLE.
CREATE UNIQUE INDEX IF NOT EXISTS `shop_orders_user_id_idempotency_key_unique` ON `shop_orders` (`user_id`,`idempotency_key`);

-- +goose Down
ALTER TABLE `shop_orders` DROP FOREIGN KEY IF EXISTS `shop_orders_discount_code_id_foreign`;
ALTER TABLE `shop_orders`
  DROP KEY IF EXISTS `shop_orders_discount_code_id_foreign`,
  DROP KEY IF EXISTS `shop_orders_user_id_idempotency_key_unique`,
  DROP COLUMN IF EXISTS `idempotency_key`,
  DROP COLUMN IF EXISTS `discount_code_id`;
ALTER TABLE `shop_carts` DROP FOREIGN KEY IF EXISTS `shop_carts_discount_code_id_foreign`;
ALTER TABLE `shop_carts`
  DROP KEY IF EXISTS `shop_carts_discount_code_id_foreign`,
  DROP COLUMN IF EXISTS `discount_code_id`;
DROP TABLE IF EXISTS `shop_discount_codes`;

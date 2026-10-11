-- 00080_shop_commerce.sql — the internal shop: catalog depth, cart, pay-on-delivery orders, returns, layette
-- checklist (CB-SHOP-01, DECISIONS #13 MVP without online payment, #15 internal shop supersedes partner link-out).
--
-- Built ON bloom's 00071 (B-N10-03): shop_categories / shop_products / shop_click_counts are only EXTENDED here
-- (additive nullable/defaulted columns, never recreated or renamed). A product with `external_url` set and no variants
-- stays a link-out row (price_text, click counter, interstitial) until an admin migrates it; a product with variants is
-- internal. Money columns are integer toman (bigint unsigned), never floats.
--
-- shop_sellers               verified sellers: per-seller shipping days/fee, free-shipping threshold, COD ceiling,
--                            discreet packaging, return window. One shipment per seller per order.
-- shop_categories  +parent_id sub-categories under the two seeded roots (layette, beauty).
-- shop_products    +seller_id, subtitle/description/images/attributes (material, wash, size guide, ingredients… as
--                            translatable JSON), irc_code + expires_on + skin_types (beauty), badges, discreet_packaging,
--                            rating cache (rating_avg/rating_count) and purchase_count.
-- shop_product_variants      size / color / volume (translatable JSON), price, compare-at price, stock.
-- shop_carts / _cart_items   one cart per user; lines point at a variant.
-- shop_addresses             the user's delivery addresses (shop data only).
-- shop_delivery_slots        per-seller dated delivery windows with optional capacity.
-- shop_orders                COD only in the MVP (payment_method 'cod'); recipient/address SNAPSHOT so later address
--                            edits never rewrite history; discreet flag; totals.
-- shop_order_shipments       per-seller shipment: slot snapshot, status, fee, tracking, delivered_at.
-- shop_order_items           title / variant label SNAPSHOT, unit price, quantity, line total.
-- shop_order_events          status timeline (order- or shipment-level).
-- shop_returns               return requests per order item (window enforced by the service).
-- shop_product_reviews       verified-purchase reviews (unique per order item), moderated (status).
-- shop_layette_items         admin layette checklist template (group, essential / first-days flags, «see» category).
-- shop_user_checklist_items  the user's own checklist state per template item, or a custom row (NULL template).
-- shop_supply_reminders      period-supply reminder opt-in (lead days, optional product). The reminder SERVICE reads
--                            cycle predictions through Go interfaces; no SQL here joins a health table.
--
-- No shop table references a health table and no health table references a shop table. User rows cascade on account
-- deletion. Twin of backend/database/migrations/2026_10_11_000080_create_shop_commerce_tables.php.
-- IF [NOT] EXISTS so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `shop_sellers` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `slug` varchar(64) NOT NULL,
  `name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`name`)),
  `logo_path` varchar(255) DEFAULT NULL,
  `is_verified` tinyint(1) NOT NULL DEFAULT 0,
  `ship_min_days` tinyint(3) unsigned NOT NULL DEFAULT 1,
  `ship_max_days` tinyint(3) unsigned NOT NULL DEFAULT 3,
  `discreet_packaging` tinyint(1) NOT NULL DEFAULT 0,
  `shipping_fee` bigint(20) unsigned NOT NULL DEFAULT 0,
  `free_shipping_over` bigint(20) unsigned DEFAULT NULL,
  `cod_max_total` bigint(20) unsigned DEFAULT NULL,
  `return_days` tinyint(3) unsigned NOT NULL DEFAULT 7,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_sellers_slug_unique` (`slug`),
  KEY `shop_sellers_is_active_sort_order_index` (`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `shop_categories`
  ADD COLUMN IF NOT EXISTS `parent_id` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `shop_categories_parent_id_foreign` (`parent_id`),
  ADD CONSTRAINT `shop_categories_parent_id_foreign` FOREIGN KEY IF NOT EXISTS (`parent_id`) REFERENCES `shop_categories` (`id`) ON DELETE SET NULL;

ALTER TABLE `shop_products`
  ADD COLUMN IF NOT EXISTS `seller_id` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `subtitle` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`subtitle`)),
  ADD COLUMN IF NOT EXISTS `description` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`description`)),
  ADD COLUMN IF NOT EXISTS `images` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`images`)),
  ADD COLUMN IF NOT EXISTS `attributes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`attributes`)),
  ADD COLUMN IF NOT EXISTS `irc_code` varchar(40) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `expires_on` date DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `skin_types` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`skin_types`)),
  ADD COLUMN IF NOT EXISTS `badges` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`badges`)),
  ADD COLUMN IF NOT EXISTS `discreet_packaging` tinyint(1) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS `rating_avg` decimal(3,2) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `rating_count` int(10) unsigned NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS `purchase_count` int(10) unsigned NOT NULL DEFAULT 0,
  ADD KEY IF NOT EXISTS `shop_products_seller_id_foreign` (`seller_id`),
  ADD CONSTRAINT `shop_products_seller_id_foreign` FOREIGN KEY IF NOT EXISTS (`seller_id`) REFERENCES `shop_sellers` (`id`) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS `shop_product_variants` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `product_id` bigint(20) unsigned NOT NULL,
  `sku` varchar(64) DEFAULT NULL,
  `size` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`size`)),
  `color` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`color`)),
  `volume` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`volume`)),
  `price` bigint(20) unsigned NOT NULL,
  `compare_at_price` bigint(20) unsigned DEFAULT NULL,
  `stock` int(10) unsigned NOT NULL DEFAULT 0,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_product_variants_sku_unique` (`sku`),
  KEY `shop_product_variants_product_id_is_active_sort_order_index` (`product_id`,`is_active`,`sort_order`),
  CONSTRAINT `shop_product_variants_product_id_foreign` FOREIGN KEY (`product_id`) REFERENCES `shop_products` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_carts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_carts_user_id_unique` (`user_id`),
  CONSTRAINT `shop_carts_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_cart_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `cart_id` bigint(20) unsigned NOT NULL,
  `variant_id` bigint(20) unsigned NOT NULL,
  `quantity` smallint(5) unsigned NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_cart_items_cart_id_variant_id_unique` (`cart_id`,`variant_id`),
  KEY `shop_cart_items_variant_id_foreign` (`variant_id`),
  CONSTRAINT `shop_cart_items_cart_id_foreign` FOREIGN KEY (`cart_id`) REFERENCES `shop_carts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_cart_items_variant_id_foreign` FOREIGN KEY (`variant_id`) REFERENCES `shop_product_variants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_addresses` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `label` varchar(40) DEFAULT NULL,
  `recipient_name` varchar(120) NOT NULL,
  `recipient_phone` varchar(20) NOT NULL,
  `province` varchar(60) DEFAULT NULL,
  `city` varchar(60) NOT NULL,
  `address_line` varchar(500) NOT NULL,
  `postal_code` varchar(10) DEFAULT NULL,
  `is_default` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `shop_addresses_user_id_foreign` (`user_id`),
  CONSTRAINT `shop_addresses_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_delivery_slots` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `seller_id` bigint(20) unsigned NOT NULL,
  `slot_date` date NOT NULL,
  `starts_at` time NOT NULL,
  `ends_at` time NOT NULL,
  `capacity` smallint(5) unsigned DEFAULT NULL,
  `booked` smallint(5) unsigned NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_delivery_slots_seller_id_slot_date_starts_at_unique` (`seller_id`,`slot_date`,`starts_at`),
  CONSTRAINT `shop_delivery_slots_seller_id_foreign` FOREIGN KEY (`seller_id`) REFERENCES `shop_sellers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_orders` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(16) NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `address_id` bigint(20) unsigned DEFAULT NULL,
  `recipient_name` varchar(120) NOT NULL,
  `recipient_phone` varchar(20) NOT NULL,
  `city` varchar(60) NOT NULL,
  `address_text` varchar(600) NOT NULL,
  `postal_code` varchar(10) DEFAULT NULL,
  `payment_method` varchar(16) NOT NULL DEFAULT 'cod',
  `payment_status` varchar(16) NOT NULL DEFAULT 'unpaid',
  `status` varchar(20) NOT NULL DEFAULT 'placed',
  `discreet` tinyint(1) NOT NULL DEFAULT 0,
  `items_total` bigint(20) unsigned NOT NULL,
  `shipping_total` bigint(20) unsigned NOT NULL DEFAULT 0,
  `discount_total` bigint(20) unsigned NOT NULL DEFAULT 0,
  `payable_total` bigint(20) unsigned NOT NULL,
  `placed_at` timestamp NULL DEFAULT NULL,
  `cancelled_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_orders_code_unique` (`code`),
  KEY `shop_orders_address_id_foreign` (`address_id`),
  KEY `shop_orders_user_id_created_at_index` (`user_id`,`created_at`),
  KEY `shop_orders_status_created_at_index` (`status`,`created_at`),
  CONSTRAINT `shop_orders_address_id_foreign` FOREIGN KEY (`address_id`) REFERENCES `shop_addresses` (`id`) ON DELETE SET NULL,
  CONSTRAINT `shop_orders_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_order_shipments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint(20) unsigned NOT NULL,
  `seller_id` bigint(20) unsigned NOT NULL,
  `delivery_slot_id` bigint(20) unsigned DEFAULT NULL,
  `slot_date` date DEFAULT NULL,
  `slot_starts_at` time DEFAULT NULL,
  `slot_ends_at` time DEFAULT NULL,
  `status` varchar(20) NOT NULL DEFAULT 'awaiting_seller',
  `items_total` bigint(20) unsigned NOT NULL,
  `shipping_fee` bigint(20) unsigned NOT NULL DEFAULT 0,
  `tracking_code` varchar(64) DEFAULT NULL,
  `delivered_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `shop_order_shipments_order_id_foreign` (`order_id`),
  KEY `shop_order_shipments_delivery_slot_id_foreign` (`delivery_slot_id`),
  KEY `shop_order_shipments_seller_id_status_index` (`seller_id`,`status`),
  CONSTRAINT `shop_order_shipments_delivery_slot_id_foreign` FOREIGN KEY (`delivery_slot_id`) REFERENCES `shop_delivery_slots` (`id`) ON DELETE SET NULL,
  CONSTRAINT `shop_order_shipments_order_id_foreign` FOREIGN KEY (`order_id`) REFERENCES `shop_orders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_order_shipments_seller_id_foreign` FOREIGN KEY (`seller_id`) REFERENCES `shop_sellers` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_order_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint(20) unsigned NOT NULL,
  `shipment_id` bigint(20) unsigned NOT NULL,
  `product_id` bigint(20) unsigned DEFAULT NULL,
  `variant_id` bigint(20) unsigned DEFAULT NULL,
  `title` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`title`)),
  `variant_label` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`variant_label`)),
  `unit_price` bigint(20) unsigned NOT NULL,
  `quantity` smallint(5) unsigned NOT NULL,
  `line_total` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `shop_order_items_order_id_foreign` (`order_id`),
  KEY `shop_order_items_shipment_id_foreign` (`shipment_id`),
  KEY `shop_order_items_product_id_foreign` (`product_id`),
  KEY `shop_order_items_variant_id_foreign` (`variant_id`),
  CONSTRAINT `shop_order_items_order_id_foreign` FOREIGN KEY (`order_id`) REFERENCES `shop_orders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_order_items_product_id_foreign` FOREIGN KEY (`product_id`) REFERENCES `shop_products` (`id`) ON DELETE SET NULL,
  CONSTRAINT `shop_order_items_shipment_id_foreign` FOREIGN KEY (`shipment_id`) REFERENCES `shop_order_shipments` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_order_items_variant_id_foreign` FOREIGN KEY (`variant_id`) REFERENCES `shop_product_variants` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_order_events` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint(20) unsigned NOT NULL,
  `shipment_id` bigint(20) unsigned DEFAULT NULL,
  `status` varchar(20) NOT NULL,
  `note` varchar(255) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `shop_order_events_shipment_id_foreign` (`shipment_id`),
  KEY `shop_order_events_order_id_created_at_index` (`order_id`,`created_at`),
  CONSTRAINT `shop_order_events_order_id_foreign` FOREIGN KEY (`order_id`) REFERENCES `shop_orders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_order_events_shipment_id_foreign` FOREIGN KEY (`shipment_id`) REFERENCES `shop_order_shipments` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_returns` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint(20) unsigned NOT NULL,
  `order_item_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `quantity` smallint(5) unsigned NOT NULL DEFAULT 1,
  `reason` varchar(32) NOT NULL,
  `note` varchar(1000) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'requested',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `shop_returns_order_id_foreign` (`order_id`),
  KEY `shop_returns_order_item_id_foreign` (`order_item_id`),
  KEY `shop_returns_user_id_foreign` (`user_id`),
  KEY `shop_returns_status_created_at_index` (`status`,`created_at`),
  CONSTRAINT `shop_returns_order_id_foreign` FOREIGN KEY (`order_id`) REFERENCES `shop_orders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_returns_order_item_id_foreign` FOREIGN KEY (`order_item_id`) REFERENCES `shop_order_items` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_returns_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_product_reviews` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `product_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `order_item_id` bigint(20) unsigned DEFAULT NULL,
  `author_name` varchar(60) NOT NULL,
  `variant_label` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`variant_label`)),
  `rating` tinyint(3) unsigned NOT NULL,
  `body` varchar(2000) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_product_reviews_order_item_id_unique` (`order_item_id`),
  KEY `shop_product_reviews_user_id_foreign` (`user_id`),
  KEY `shop_product_reviews_product_id_status_created_at_index` (`product_id`,`status`,`created_at`),
  CONSTRAINT `shop_product_reviews_order_item_id_foreign` FOREIGN KEY (`order_item_id`) REFERENCES `shop_order_items` (`id`) ON DELETE SET NULL,
  CONSTRAINT `shop_product_reviews_product_id_foreign` FOREIGN KEY (`product_id`) REFERENCES `shop_products` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_product_reviews_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_layette_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `group_key` varchar(32) NOT NULL,
  `title` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`title`)),
  `note` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`note`)),
  `suggested_qty` smallint(5) unsigned DEFAULT NULL,
  `is_essential` tinyint(1) NOT NULL DEFAULT 0,
  `is_first_days` tinyint(1) NOT NULL DEFAULT 0,
  `category_id` bigint(20) unsigned DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `shop_layette_items_category_id_foreign` (`category_id`),
  KEY `shop_layette_items_is_active_group_key_sort_order_index` (`is_active`,`group_key`,`sort_order`),
  CONSTRAINT `shop_layette_items_category_id_foreign` FOREIGN KEY (`category_id`) REFERENCES `shop_categories` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_user_checklist_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `layette_item_id` bigint(20) unsigned DEFAULT NULL,
  `custom_title` varchar(120) DEFAULT NULL,
  `group_key` varchar(32) NOT NULL,
  `state` varchar(16) NOT NULL DEFAULT 'todo',
  `quantity` smallint(5) unsigned DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_user_checklist_items_user_id_layette_item_id_unique` (`user_id`,`layette_item_id`),
  KEY `shop_user_checklist_items_layette_item_id_foreign` (`layette_item_id`),
  CONSTRAINT `shop_user_checklist_items_layette_item_id_foreign` FOREIGN KEY (`layette_item_id`) REFERENCES `shop_layette_items` (`id`) ON DELETE CASCADE,
  CONSTRAINT `shop_user_checklist_items_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_supply_reminders` (
  `user_id` bigint(20) unsigned NOT NULL,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 0,
  `lead_days` tinyint(3) unsigned NOT NULL DEFAULT 3,
  `product_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`user_id`),
  KEY `shop_supply_reminders_product_id_foreign` (`product_id`),
  CONSTRAINT `shop_supply_reminders_product_id_foreign` FOREIGN KEY (`product_id`) REFERENCES `shop_products` (`id`) ON DELETE SET NULL,
  CONSTRAINT `shop_supply_reminders_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `shop_supply_reminders`;
DROP TABLE IF EXISTS `shop_user_checklist_items`;
DROP TABLE IF EXISTS `shop_layette_items`;
DROP TABLE IF EXISTS `shop_product_reviews`;
DROP TABLE IF EXISTS `shop_returns`;
DROP TABLE IF EXISTS `shop_order_events`;
DROP TABLE IF EXISTS `shop_order_items`;
DROP TABLE IF EXISTS `shop_order_shipments`;
DROP TABLE IF EXISTS `shop_orders`;
DROP TABLE IF EXISTS `shop_delivery_slots`;
DROP TABLE IF EXISTS `shop_addresses`;
DROP TABLE IF EXISTS `shop_cart_items`;
DROP TABLE IF EXISTS `shop_carts`;
DROP TABLE IF EXISTS `shop_product_variants`;
ALTER TABLE `shop_products` DROP FOREIGN KEY IF EXISTS `shop_products_seller_id_foreign`;
ALTER TABLE `shop_products`
  DROP KEY IF EXISTS `shop_products_seller_id_foreign`,
  DROP COLUMN IF EXISTS `purchase_count`,
  DROP COLUMN IF EXISTS `rating_count`,
  DROP COLUMN IF EXISTS `rating_avg`,
  DROP COLUMN IF EXISTS `discreet_packaging`,
  DROP COLUMN IF EXISTS `badges`,
  DROP COLUMN IF EXISTS `skin_types`,
  DROP COLUMN IF EXISTS `expires_on`,
  DROP COLUMN IF EXISTS `irc_code`,
  DROP COLUMN IF EXISTS `attributes`,
  DROP COLUMN IF EXISTS `images`,
  DROP COLUMN IF EXISTS `description`,
  DROP COLUMN IF EXISTS `subtitle`,
  DROP COLUMN IF EXISTS `seller_id`;
ALTER TABLE `shop_categories` DROP FOREIGN KEY IF EXISTS `shop_categories_parent_id_foreign`;
ALTER TABLE `shop_categories`
  DROP KEY IF EXISTS `shop_categories_parent_id_foreign`,
  DROP COLUMN IF EXISTS `parent_id`;
DROP TABLE IF EXISTS `shop_sellers`;

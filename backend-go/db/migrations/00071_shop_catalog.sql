-- 00071_shop_catalog.sql — shop entry and link-out catalog (bloom B-N10-03, D-91).
--
-- shop_categories    admin-managed categories («سیسمونی و نوزاد», «آرایشی و بهداشتی»): translatable `title`, stable `slug`.
-- shop_products      admin-managed products: translatable `title`, public-disk `image_path`, free-text `price_text` and
--                    `badge`, `partner_label`, and a NULLABLE `external_url` (link-out). A NULL `external_url` is a
--                    product without a link-out (CB-SHOP-01 extends this table with the internal COD shop; link-out
--                    rows must keep rendering). No cart, no payment, no stock here.
-- shop_click_counts  per product and Tehran day: how many link-out clicks. NO user id, no session, no history.
--
-- The shop never references a health table and no health table references it. Only the two categories are seeded;
-- products are admin data. Twin of backend/database/migrations/2026_10_11_000071_create_shop_catalog_tables.php.
-- IF NOT EXISTS / INSERT IGNORE so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `shop_categories` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `slug` varchar(64) NOT NULL,
  `title` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`title`)),
  `icon` varchar(32) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `shop_categories_slug_unique` (`slug`),
  KEY `shop_categories_is_active_sort_order_index` (`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_products` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `category_id` bigint(20) unsigned NOT NULL,
  `title` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`title`)),
  `image_path` varchar(255) DEFAULT NULL,
  `price_text` varchar(80) DEFAULT NULL,
  `badge` varchar(60) DEFAULT NULL,
  `external_url` varchar(1000) DEFAULT NULL,
  `partner_label` varchar(120) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `shop_products_category_id_is_active_sort_order_index` (`category_id`,`is_active`,`sort_order`),
  CONSTRAINT `shop_products_category_id_foreign` FOREIGN KEY (`category_id`) REFERENCES `shop_categories` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `shop_click_counts` (
  `product_id` bigint(20) unsigned NOT NULL,
  `day` date NOT NULL,
  `clicks` int(10) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`product_id`,`day`),
  CONSTRAINT `shop_click_counts_product_id_foreign` FOREIGN KEY (`product_id`) REFERENCES `shop_products` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `shop_categories` (`slug`, `title`, `icon`, `sort_order`, `is_active`, `created_at`, `updated_at`) VALUES
  ('layette', '{"fa":"سیسمونی و نوزاد","en":"Layette & baby"}', 'bottle', 1, 1, NOW(), NOW()),
  ('beauty', '{"fa":"آرایشی و بهداشتی","en":"Beauty & hygiene"}', 'sparkle', 2, 1, NOW(), NOW());

-- +goose Down
DROP TABLE IF EXISTS `shop_click_counts`;
DROP TABLE IF EXISTS `shop_products`;
DROP TABLE IF EXISTS `shop_categories`;

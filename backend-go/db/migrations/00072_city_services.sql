-- 00072_city_services.sql — city services: classes, pools, playhouses and simple booking requests (bloom B-N10-05, D-92).
--
-- city_service_cities      admin-managed cities (translatable `name`, stable `slug`).
-- city_service_categories  admin-editable categories; the three seeded codes are class / pool / playhouse.
-- city_service_listings    admin-managed venues: name, category, city, district, address, phone, description, price and
--                          hours as free text, public-disk `photos` (JSON list of paths), `booking_mode`
--                          link | request | phone and a NULLABLE `booking_url`. CB-DIR-01 extends this table (map,
--                          services, slots, reviews); rows must keep rendering.
-- city_service_bookings    a user's booking REQUEST for a `request`-mode listing: contact name + phone, preferred day,
--                          optional child AGE RANGE (typed by the user) and a short note. Status requested →
--                          confirmed | rejected | cancelled | completed, moved by the admin team (they confirm by
--                          phone) or cancelled by the owner. NEVER a health / medical field, never cycle or pregnancy
--                          state. CB-DIR-09 supersedes the simple «رزروها» list over this table.
--
-- Only the three categories are seeded; cities and listings are admin data. Twin of
-- backend/database/migrations/2026_10_11_000072_create_city_services_tables.php.
-- IF NOT EXISTS / INSERT IGNORE so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `city_service_cities` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `slug` varchar(64) NOT NULL,
  `name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`name`)),
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `city_service_cities_slug_unique` (`slug`),
  KEY `city_service_cities_is_active_sort_order_index` (`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `city_service_categories` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `slug` varchar(64) NOT NULL,
  `name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`name`)),
  `icon` varchar(32) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `city_service_categories_slug_unique` (`slug`),
  KEY `city_service_categories_is_active_sort_order_index` (`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `city_service_listings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `category_id` bigint(20) unsigned NOT NULL,
  `city_id` bigint(20) unsigned NOT NULL,
  `name` varchar(160) NOT NULL,
  `district` varchar(120) DEFAULT NULL,
  `address` varchar(300) DEFAULT NULL,
  `phone` varchar(32) DEFAULT NULL,
  `description` text DEFAULT NULL,
  `price_text` varchar(120) DEFAULT NULL,
  `hours_text` varchar(200) DEFAULT NULL,
  `photos` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`photos`)),
  `booking_mode` varchar(12) NOT NULL DEFAULT 'phone',
  `booking_url` varchar(1000) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `city_service_listings_category_id_foreign` (`category_id`),
  KEY `city_service_listings_city_id_category_id_is_active_index` (`city_id`,`category_id`,`is_active`),
  CONSTRAINT `city_service_listings_category_id_foreign` FOREIGN KEY (`category_id`) REFERENCES `city_service_categories` (`id`),
  CONSTRAINT `city_service_listings_city_id_foreign` FOREIGN KEY (`city_id`) REFERENCES `city_service_cities` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `city_service_bookings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `listing_id` bigint(20) unsigned NOT NULL,
  `contact_name` varchar(120) NOT NULL,
  `contact_phone` varchar(32) NOT NULL,
  `preferred_date` date NOT NULL,
  `child_age_range` varchar(16) DEFAULT NULL,
  `note` varchar(500) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'requested',
  `status_note` varchar(300) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `city_service_bookings_listing_id_foreign` (`listing_id`),
  KEY `city_service_bookings_user_id_created_at_index` (`user_id`,`created_at`),
  KEY `city_service_bookings_status_created_at_index` (`status`,`created_at`),
  CONSTRAINT `city_service_bookings_listing_id_foreign` FOREIGN KEY (`listing_id`) REFERENCES `city_service_listings` (`id`),
  CONSTRAINT `city_service_bookings_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `city_service_categories` (`slug`, `name`, `icon`, `sort_order`, `is_active`, `created_at`, `updated_at`) VALUES
  ('class', '{"fa":"کلاس","en":"Classes"}', 'book', 1, 1, NOW(), NOW()),
  ('pool', '{"fa":"استخر","en":"Pools"}', 'drop', 2, 1, NOW(), NOW()),
  ('playhouse', '{"fa":"خانه بازی","en":"Playhouses"}', 'home', 3, 1, NOW(), NOW());

-- +goose Down
DROP TABLE IF EXISTS `city_service_bookings`;
DROP TABLE IF EXISTS `city_service_listings`;
DROP TABLE IF EXISTS `city_service_categories`;
DROP TABLE IF EXISTS `city_service_cities`;

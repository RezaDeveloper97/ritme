-- 00081_directory.sql — the mother & child directory (CB-DIR-01, docs/canvas-build/directory.md) ON TOP of bloom's
-- city services (00072, B-N10-05). Bloom's four tables are EXTENDED (ALTER, never recreated or renamed) and the new
-- tables keep the `city_service_` prefix, so a directory "place" IS a `city_service_listings` row and a directory
-- booking IS a `city_service_bookings` row. Existing link / request / phone listings and bloom's simple bookings keep
-- working unchanged: every added column is NULLable or defaulted, `status` defaults to 'published'.
--
-- city_service_listings      + status, verified_at, owner_user_id, lat/lng, about, age_ranges, amenities, weekly_hours,
--                            photo_file_ids, website_url, cancel_free_hours, cancel_policy, rules, rating_avg/count.
--                            booking_mode gains the value 'online' (varchar(12) already fits): DECISIONS #13 — it is
--                            still a REQUEST the Ritme team confirms in admin, never auto-confirmed, no payment fields.
--                            bloom's booking_mode / hours_text / price_text / photos stay as legacy fallbacks.
-- city_service_services      a place's services and packages (duration, age range in months, capacity, price shown
--                            as the venue declared it — MVP has no online payment).
-- city_service_slots         open times of a place (optionally of one service), capacity / booked count.
-- city_service_bookings      + code (user-visible), service_id, slot_id, child_id (bloom `children`), price snapshot,
--                            status timestamps and who cancelled. contact_name / contact_phone / preferred_date stay.
-- city_service_reviews       one review per COMPLETED booking (enforced by the service, CB-DIR-03), overall rating +
--                            cleanliness / instructor / amenities sub-scores.
-- city_service_reports       «گزارش اطلاعات نادرست» — a wrong-information report about a place, worked in admin.
-- city_service_applications  the 4-step business join draft (CB-DIR-04): step 1 intro, 2 place + photos, 3 services,
--                            4 booking mode + PRIVATE documents (files purpose place_licence, CB-CORE-05) → tracking
--                            code on submit. An approved application points at the listing it created.
--
-- Privacy: none of these tables holds a health / medical field, cycle or pregnancy state. A booking references a
-- child row only so the venue can be told the child's first name and age (Dir_Book: «فقط نام تو، نام و سن کودک و
-- شماره تماست»); reviews and reports carry the author's user id for moderation and are never sent to a venue.
--
-- Seeds (data, INSERT IGNORE so an admin edit is never overwritten):
--   city_service_categories  the board's other join types next to bloom's class / pool / playhouse (the categories
--                            stay in bloom's admin-editable table; no parallel `dir_categories` catalog group).
--   catalog_items            groups dir_amenities and dir_age_ranges (docs/canvas-build/catalog.md §5).
--
-- Twin of backend/database/migrations/2026_10_11_000081_extend_city_services_for_directory.php.
-- IF [NOT] EXISTS so a database whose Laravel half already ran migrates cleanly.

-- +goose Up
ALTER TABLE `city_service_listings`
  ADD COLUMN IF NOT EXISTS `status` varchar(16) NOT NULL DEFAULT 'published',
  ADD COLUMN IF NOT EXISTS `verified_at` timestamp NULL DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `owner_user_id` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `lat` decimal(9,6) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `lng` decimal(9,6) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `about` text DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `age_ranges` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`age_ranges`)),
  ADD COLUMN IF NOT EXISTS `amenities` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`amenities`)),
  ADD COLUMN IF NOT EXISTS `weekly_hours` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`weekly_hours`)),
  ADD COLUMN IF NOT EXISTS `photo_file_ids` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`photo_file_ids`)),
  ADD COLUMN IF NOT EXISTS `website_url` varchar(1000) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `cancel_free_hours` smallint(5) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `cancel_policy` varchar(300) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `rules` text DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `rating_avg` decimal(3,2) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `rating_count` int(10) unsigned NOT NULL DEFAULT 0,
  ADD KEY IF NOT EXISTS `city_service_listings_owner_user_id_foreign` (`owner_user_id`),
  ADD KEY IF NOT EXISTS `city_service_listings_status_city_id_category_id_index` (`status`,`city_id`,`category_id`),
  ADD KEY IF NOT EXISTS `city_service_listings_lat_lng_index` (`lat`,`lng`),
  ADD CONSTRAINT `city_service_listings_owner_user_id_foreign` FOREIGN KEY IF NOT EXISTS (`owner_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS `city_service_services` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `listing_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(12) NOT NULL DEFAULT 'session',
  `name` varchar(160) NOT NULL,
  `description` varchar(300) DEFAULT NULL,
  `format` varchar(12) DEFAULT NULL,
  `duration_minutes` smallint(5) unsigned DEFAULT NULL,
  `age_min_months` smallint(5) unsigned DEFAULT NULL,
  `age_max_months` smallint(5) unsigned DEFAULT NULL,
  `capacity` smallint(5) unsigned DEFAULT NULL,
  `price_toman` int(10) unsigned DEFAULT NULL,
  `package_sessions` smallint(5) unsigned DEFAULT NULL,
  `package_validity_days` smallint(5) unsigned DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `city_service_services_listing_id_is_active_sort_order_index` (`listing_id`,`is_active`,`sort_order`),
  CONSTRAINT `city_service_services_listing_id_foreign` FOREIGN KEY (`listing_id`) REFERENCES `city_service_listings` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `city_service_slots` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `listing_id` bigint(20) unsigned NOT NULL,
  `service_id` bigint(20) unsigned DEFAULT NULL,
  `starts_at` datetime NOT NULL,
  `ends_at` datetime DEFAULT NULL,
  `capacity` smallint(5) unsigned NOT NULL DEFAULT 1,
  `booked_count` smallint(5) unsigned NOT NULL DEFAULT 0,
  `status` varchar(12) NOT NULL DEFAULT 'open',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `city_service_slots_listing_id_starts_at_index` (`listing_id`,`starts_at`),
  KEY `city_service_slots_service_id_starts_at_index` (`service_id`,`starts_at`),
  CONSTRAINT `city_service_slots_listing_id_foreign` FOREIGN KEY (`listing_id`) REFERENCES `city_service_listings` (`id`) ON DELETE CASCADE,
  CONSTRAINT `city_service_slots_service_id_foreign` FOREIGN KEY (`service_id`) REFERENCES `city_service_services` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `city_service_bookings`
  ADD COLUMN IF NOT EXISTS `code` varchar(16) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `service_id` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `slot_id` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `child_id` bigint(20) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `price_toman` int(10) unsigned DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `confirmed_at` timestamp NULL DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `cancelled_at` timestamp NULL DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `cancelled_by` varchar(8) DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS `completed_at` timestamp NULL DEFAULT NULL,
  ADD UNIQUE KEY IF NOT EXISTS `city_service_bookings_code_unique` (`code`),
  ADD CONSTRAINT `city_service_bookings_service_id_foreign` FOREIGN KEY IF NOT EXISTS (`service_id`) REFERENCES `city_service_services` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `city_service_bookings_slot_id_foreign` FOREIGN KEY IF NOT EXISTS (`slot_id`) REFERENCES `city_service_slots` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `city_service_bookings_child_id_foreign` FOREIGN KEY IF NOT EXISTS (`child_id`) REFERENCES `children` (`id`) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS `city_service_reviews` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `listing_id` bigint(20) unsigned NOT NULL,
  `booking_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `rating` tinyint(3) unsigned NOT NULL,
  `score_cleanliness` tinyint(3) unsigned DEFAULT NULL,
  `score_instructor` tinyint(3) unsigned DEFAULT NULL,
  `score_amenities` tinyint(3) unsigned DEFAULT NULL,
  `body` varchar(1000) DEFAULT NULL,
  `status` varchar(12) NOT NULL DEFAULT 'published',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `city_service_reviews_booking_id_unique` (`booking_id`),
  KEY `city_service_reviews_user_id_foreign` (`user_id`),
  KEY `city_service_reviews_listing_id_status_created_at_index` (`listing_id`,`status`,`created_at`),
  CONSTRAINT `city_service_reviews_booking_id_foreign` FOREIGN KEY (`booking_id`) REFERENCES `city_service_bookings` (`id`) ON DELETE CASCADE,
  CONSTRAINT `city_service_reviews_listing_id_foreign` FOREIGN KEY (`listing_id`) REFERENCES `city_service_listings` (`id`) ON DELETE CASCADE,
  CONSTRAINT `city_service_reviews_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `city_service_reports` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `listing_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `reason` varchar(32) NOT NULL,
  `note` varchar(500) DEFAULT NULL,
  `status` varchar(12) NOT NULL DEFAULT 'open',
  `resolved_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `city_service_reports_listing_id_foreign` (`listing_id`),
  KEY `city_service_reports_user_id_foreign` (`user_id`),
  KEY `city_service_reports_status_created_at_index` (`status`,`created_at`),
  CONSTRAINT `city_service_reports_listing_id_foreign` FOREIGN KEY (`listing_id`) REFERENCES `city_service_listings` (`id`) ON DELETE CASCADE,
  CONSTRAINT `city_service_reports_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `city_service_applications` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `tracking_code` varchar(16) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'draft',
  `step` tinyint(3) unsigned NOT NULL DEFAULT 1,
  `business_name` varchar(160) DEFAULT NULL,
  `category_slugs` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`category_slugs`)),
  `city_id` bigint(20) unsigned DEFAULT NULL,
  `district` varchar(120) DEFAULT NULL,
  `contact_name` varchar(120) DEFAULT NULL,
  `contact_phone` varchar(32) DEFAULT NULL,
  `owner_confirmed` tinyint(1) NOT NULL DEFAULT 0,
  `lat` decimal(9,6) DEFAULT NULL,
  `lng` decimal(9,6) DEFAULT NULL,
  `address` varchar(300) DEFAULT NULL,
  `about` text DEFAULT NULL,
  `age_ranges` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`age_ranges`)),
  `amenities` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`amenities`)),
  `weekly_hours` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`weekly_hours`)),
  `photo_file_ids` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`photo_file_ids`)),
  `services` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`services`)),
  `booking_mode` varchar(12) DEFAULT NULL,
  `documents` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`documents`)),
  `terms_accepted_at` timestamp NULL DEFAULT NULL,
  `submitted_at` timestamp NULL DEFAULT NULL,
  `reviewed_at` timestamp NULL DEFAULT NULL,
  `review_note` varchar(500) DEFAULT NULL,
  `listing_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `city_service_applications_tracking_code_unique` (`tracking_code`),
  KEY `city_service_applications_city_id_foreign` (`city_id`),
  KEY `city_service_applications_listing_id_foreign` (`listing_id`),
  KEY `city_service_applications_user_id_status_index` (`user_id`,`status`),
  KEY `city_service_applications_status_submitted_at_index` (`status`,`submitted_at`),
  CONSTRAINT `city_service_applications_city_id_foreign` FOREIGN KEY (`city_id`) REFERENCES `city_service_cities` (`id`) ON DELETE SET NULL,
  CONSTRAINT `city_service_applications_listing_id_foreign` FOREIGN KEY (`listing_id`) REFERENCES `city_service_listings` (`id`) ON DELETE SET NULL,
  CONSTRAINT `city_service_applications_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `city_service_categories` (`slug`, `name`, `icon`, `sort_order`, `is_active`, `created_at`, `updated_at`) VALUES
  ('music_art', '{"fa":"موسیقی و هنر","en":"Music & art"}', 'mic', 4, 1, NOW(), NOW()),
  ('postpartum_fitness', '{"fa":"ورزش پس از زایمان","en":"Postpartum fitness"}', 'walk', 5, 1, NOW(), NOW()),
  ('baby_massage', '{"fa":"ماساژ نوزاد","en":"Baby massage"}', 'hand', 6, 1, NOW(), NOW()),
  ('parenting_workshop', '{"fa":"کارگاه فرزندپروری","en":"Parenting workshops"}', 'users', 7, 1, NOW(), NOW()),
  ('other', '{"fa":"دیگر","en":"Other"}', 'grid', 8, 1, NOW(), NOW());

INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('dir_amenities', 'nursing_room', 1, 1, NULL, '{"fa":"اتاق شیردهی","en":"Nursing room"}', NULL, '{"icon":"heart"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_amenities', 'changing_table', 2, 1, NULL, '{"fa":"میز تعویض","en":"Changing table"}', NULL, '{"icon":"box"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_amenities', 'female_instructor', 3, 1, NULL, '{"fa":"مربی خانم","en":"Female instructor"}', NULL, '{"icon":"female"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_amenities', 'stroller_parking', 4, 1, NULL, '{"fa":"جای کالسکه","en":"Stroller parking"}', NULL, '{"icon":"home"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_amenities', 'parking', 5, 1, NULL, '{"fa":"پارکینگ","en":"Parking"}', NULL, '{"icon":"mapPin"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_amenities', 'elevator', 6, 1, NULL, '{"fa":"آسانسور","en":"Elevator"}', NULL, '{"icon":"arrowR"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_amenities', 'wheelchair_access', 7, 1, NULL, '{"fa":"دسترسی ویلچر","en":"Wheelchair access"}', NULL, '{"icon":"user"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_age_ranges', 'pregnancy', 1, 1, NULL, '{"fa":"بارداری","en":"Pregnancy"}', NULL, '{"pregnancy":true}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_age_ranges', 'm0_6', 2, 1, NULL, '{"fa":"۰ تا ۶ ماه","en":"0–6 months"}', NULL, '{"min_months":0,"max_months":6}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_age_ranges', 'm6_12', 3, 1, NULL, '{"fa":"۶ تا ۱۲ ماه","en":"6–12 months"}', NULL, '{"min_months":6,"max_months":12}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_age_ranges', 'y1_2', 4, 1, NULL, '{"fa":"۱ تا ۲ سال","en":"1–2 years"}', NULL, '{"min_months":12,"max_months":24}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_age_ranges', 'y2_4', 5, 1, NULL, '{"fa":"۲ تا ۴ سال","en":"2–4 years"}', NULL, '{"min_months":24,"max_months":48}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dir_age_ranges', 'y4_6', 6, 1, NULL, '{"fa":"۴ تا ۶ سال","en":"4–6 years"}', NULL, '{"min_months":48,"max_months":72}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'dir_amenities'
  AND `code` IN ('nursing_room', 'changing_table', 'female_instructor', 'stroller_parking', 'parking', 'elevator', 'wheelchair_access');
DELETE FROM `catalog_items` WHERE `group` = 'dir_age_ranges' AND `code` IN ('pregnancy', 'm0_6', 'm6_12', 'y1_2', 'y2_4', 'y4_6');
DELETE FROM `city_service_categories` WHERE `slug` IN ('music_art', 'postpartum_fitness', 'baby_massage', 'parenting_workshop', 'other')
  AND `id` NOT IN (SELECT `category_id` FROM `city_service_listings`);
DROP TABLE IF EXISTS `city_service_applications`;
DROP TABLE IF EXISTS `city_service_reports`;
DROP TABLE IF EXISTS `city_service_reviews`;
ALTER TABLE `city_service_bookings`
  DROP FOREIGN KEY IF EXISTS `city_service_bookings_child_id_foreign`,
  DROP FOREIGN KEY IF EXISTS `city_service_bookings_slot_id_foreign`,
  DROP FOREIGN KEY IF EXISTS `city_service_bookings_service_id_foreign`;
ALTER TABLE `city_service_bookings`
  DROP KEY IF EXISTS `city_service_bookings_child_id_foreign`,
  DROP KEY IF EXISTS `city_service_bookings_slot_id_foreign`,
  DROP KEY IF EXISTS `city_service_bookings_service_id_foreign`,
  DROP KEY IF EXISTS `city_service_bookings_code_unique`,
  DROP COLUMN IF EXISTS `completed_at`,
  DROP COLUMN IF EXISTS `cancelled_by`,
  DROP COLUMN IF EXISTS `cancelled_at`,
  DROP COLUMN IF EXISTS `confirmed_at`,
  DROP COLUMN IF EXISTS `price_toman`,
  DROP COLUMN IF EXISTS `child_id`,
  DROP COLUMN IF EXISTS `slot_id`,
  DROP COLUMN IF EXISTS `service_id`,
  DROP COLUMN IF EXISTS `code`;
DROP TABLE IF EXISTS `city_service_slots`;
DROP TABLE IF EXISTS `city_service_services`;
ALTER TABLE `city_service_listings`
  DROP FOREIGN KEY IF EXISTS `city_service_listings_owner_user_id_foreign`;
ALTER TABLE `city_service_listings`
  DROP KEY IF EXISTS `city_service_listings_owner_user_id_foreign`,
  DROP KEY IF EXISTS `city_service_listings_lat_lng_index`,
  DROP KEY IF EXISTS `city_service_listings_status_city_id_category_id_index`,
  DROP COLUMN IF EXISTS `rating_count`,
  DROP COLUMN IF EXISTS `rating_avg`,
  DROP COLUMN IF EXISTS `rules`,
  DROP COLUMN IF EXISTS `cancel_policy`,
  DROP COLUMN IF EXISTS `cancel_free_hours`,
  DROP COLUMN IF EXISTS `website_url`,
  DROP COLUMN IF EXISTS `photo_file_ids`,
  DROP COLUMN IF EXISTS `weekly_hours`,
  DROP COLUMN IF EXISTS `amenities`,
  DROP COLUMN IF EXISTS `age_ranges`,
  DROP COLUMN IF EXISTS `about`,
  DROP COLUMN IF EXISTS `lng`,
  DROP COLUMN IF EXISTS `lat`,
  DROP COLUMN IF EXISTS `owner_user_id`,
  DROP COLUMN IF EXISTS `verified_at`,
  DROP COLUMN IF EXISTS `status`;

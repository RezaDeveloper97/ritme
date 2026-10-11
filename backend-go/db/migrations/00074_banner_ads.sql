-- 00074_banner_ads.sql — home ad slots on the banner system (bloom B-N10-02, D-93).
--
-- banners             + kind ('banner' = the existing slideshow, 'ad' = a sponsored slot), advertiser (admin-only label,
--                     never sent to the app), cta_label (json {fa,en}) and modes (json list of life modes it targets;
--                     NULL/[] = every mode). Targeting is by app mode only, never by health data.
-- banner_daily_stats  per-banner, per-Tehran-day impression/click counters. NO user id, no device, no health context.
--
-- Twin of backend/database/migrations/2026_10_11_000074_add_ads_to_banners.php. IF [NOT] EXISTS so a database that
-- already carries the columns (an earlier run of this task, or the Laravel twin) migrates cleanly.

-- +goose Up
ALTER TABLE `banners`
  ADD COLUMN IF NOT EXISTS `kind` varchar(16) NOT NULL DEFAULT 'banner' AFTER `sort_order`,
  ADD COLUMN IF NOT EXISTS `advertiser` varchar(120) DEFAULT NULL AFTER `kind`,
  ADD COLUMN IF NOT EXISTS `cta_label` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`cta_label`)) AFTER `advertiser`,
  ADD COLUMN IF NOT EXISTS `modes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`modes`)) AFTER `cta_label`,
  ADD KEY IF NOT EXISTS `banners_kind_position_is_active_index` (`kind`,`position`,`is_active`);

CREATE TABLE IF NOT EXISTS `banner_daily_stats` (
  `banner_id` bigint(20) unsigned NOT NULL,
  `day` date NOT NULL,
  `impressions` int(10) unsigned NOT NULL DEFAULT 0,
  `clicks` int(10) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`banner_id`,`day`),
  CONSTRAINT `banner_daily_stats_banner_id_foreign` FOREIGN KEY (`banner_id`) REFERENCES `banners` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `banner_daily_stats`;
ALTER TABLE `banners`
  DROP KEY IF EXISTS `banners_kind_position_is_active_index`,
  DROP COLUMN IF EXISTS `modes`,
  DROP COLUMN IF EXISTS `cta_label`,
  DROP COLUMN IF EXISTS `advertiser`,
  DROP COLUMN IF EXISTS `kind`;

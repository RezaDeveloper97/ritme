-- 00044_record_sharing.sql — record sharing (canvas-build CB-REC-03; artboards nbl_Rec_Share / nbl_Rec_Emergency,
-- deviations.md D-71 / D-72). Domain logic: internal/sharelinks (24h doctor code, access log) and internal/emergency
-- (emergency card).
--
-- health_share_links (bloom B-N6-04, 00042) gains a second kind of link:
--   kind          report = bloom's 7-day doctor report link (unchanged); summary = the 24h record summary «دسترسی موقت
--                 با کد» the doctor opens by scanning the QR (the token URL) or by typing the short code.
--   label         the owner's own note for whom she made it («دکتر …», optional; never shown to the viewer).
--   code_hash     HMAC-SHA256 (server pepper SHARE_CODE_PEPPER) of the short human code — the code lookup key. The code
--                 itself is never stored.
--   code_payload  the link token sealed with AES-256-GCM under a key derived from pepper ‖ code, so the code (not the
--                 server alone, not a DB leak alone) opens the snapshot. Wiped on revoke / expiry like `payload`.
--
-- health_share_link_views  the access log «سابقه دسترسی»: one row per successful public open of a link of either
--                          kind — when, through what (link | code) and a coarse client class (device + browser
--                          family). No IP address, no user agent, no location: nothing that tracks the viewer.
--                          Rows go with their link (cascade; links are deleted 30 days after expiry).
--
-- emergency_cards  «کارت اضطراری»: the owner's card settings (show on lock screen, show pregnancy status), her
--                  emergency contact and a masked insurance placeholder (label + last 4 digits only — never a full
--                  number). The card's health data (name, blood group, allergies, conditions, permanent meds,
--                  pregnancy) is read live from the record. `public_token_hash` is the SHA-256 of the optional public
--                  card link token (256 random bits; only while she turns it on).
--
-- Health data: owner-only routes; the public reads go by token / code only and are throttled. Never logged. Twin of
-- backend/database/migrations/2026_10_06_000044_create_record_sharing_tables.php so `make schema-diff` stays green;
-- IF NOT EXISTS / IF EXISTS everywhere so it is a no-op where the Laravel twin already ran (stamped databases).

-- +goose Up
ALTER TABLE `health_share_links` ADD COLUMN IF NOT EXISTS `kind` varchar(16) NOT NULL DEFAULT 'report' AFTER `user_id`;
ALTER TABLE `health_share_links` ADD COLUMN IF NOT EXISTS `label` varchar(60) DEFAULT NULL AFTER `kind`;
ALTER TABLE `health_share_links` ADD COLUMN IF NOT EXISTS `code_hash` char(64) DEFAULT NULL AFTER `token_hash`;
ALTER TABLE `health_share_links` ADD COLUMN IF NOT EXISTS `code_payload` text DEFAULT NULL AFTER `code_hash`;
CREATE UNIQUE INDEX IF NOT EXISTS `health_share_links_code_hash_unique` ON `health_share_links` (`code_hash`);
CREATE INDEX IF NOT EXISTS `health_share_links_user_id_kind_index` ON `health_share_links` (`user_id`,`kind`);

CREATE TABLE IF NOT EXISTS `health_share_link_views` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `share_link_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `via` varchar(8) NOT NULL,
  `device` varchar(16) NOT NULL,
  `browser` varchar(16) NOT NULL,
  `viewed_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `health_share_link_views_user_id_viewed_at_index` (`user_id`,`viewed_at`),
  KEY `health_share_link_views_share_link_id_viewed_at_index` (`share_link_id`,`viewed_at`),
  CONSTRAINT `health_share_link_views_share_link_id_foreign` FOREIGN KEY (`share_link_id`) REFERENCES `health_share_links` (`id`) ON DELETE CASCADE,
  CONSTRAINT `health_share_link_views_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `emergency_cards` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `show_on_lock_screen` tinyint(1) NOT NULL DEFAULT 0,
  `show_pregnancy` tinyint(1) NOT NULL DEFAULT 0,
  `contact_name` varchar(60) DEFAULT NULL,
  `contact_relation` varchar(30) DEFAULT NULL,
  `contact_phone` varchar(20) DEFAULT NULL,
  `insurance_label` varchar(60) DEFAULT NULL,
  `insurance_last4` char(4) DEFAULT NULL,
  `public_token_hash` char(64) DEFAULT NULL,
  `public_enabled_at` timestamp NULL DEFAULT NULL,
  `public_view_count` int(10) unsigned NOT NULL DEFAULT 0,
  `public_last_viewed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `emergency_cards_user_id_unique` (`user_id`),
  UNIQUE KEY `emergency_cards_public_token_hash_unique` (`public_token_hash`),
  CONSTRAINT `emergency_cards_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `emergency_cards`;
DROP TABLE IF EXISTS `health_share_link_views`;
DROP INDEX IF EXISTS `health_share_links_user_id_kind_index` ON `health_share_links`;
DROP INDEX IF EXISTS `health_share_links_code_hash_unique` ON `health_share_links`;
ALTER TABLE `health_share_links` DROP COLUMN IF EXISTS `code_payload`;
ALTER TABLE `health_share_links` DROP COLUMN IF EXISTS `code_hash`;
ALTER TABLE `health_share_links` DROP COLUMN IF EXISTS `label`;
ALTER TABLE `health_share_links` DROP COLUMN IF EXISTS `kind`;

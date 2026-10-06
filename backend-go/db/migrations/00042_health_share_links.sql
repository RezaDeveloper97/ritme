-- 00042_health_share_links.sql — the doctor report's 7-day share link (bloom B-N6-04; artboards nbl_Record_Export /
-- nbl_Record_Preview in c-health-record, deviations.md D-65). Domain logic: internal/sharelinks.
--
-- health_share_links  one row per link the owner created from «گزارش برای پزشک». The report is a snapshot of the
--                     health record built for the `share` audience (internal/healthrecord: no ids, no free-text notes,
--                     losses only as a count) plus the owner's optional question, frozen at creation and stored
--                     encrypted in `payload`: "v1:" + base64(nonce ‖ AES-256-GCM(json) ‖ tag) with a key derived from
--                     the link token (HKDF-SHA256), so the server alone cannot read a stored report. The token itself
--                     (256 random bits, base64url) is never stored — only its SHA-256 hex in `token_hash` (lookup).
--                     A link lives 7 days (`expires_at`); revoking or expiry clears `payload` (revoked_at / the purge
--                     loop), and rows are deleted 30 days after they stopped working. `sections` lists the included
--                     section keys and range_from / range_to the report window (metadata for the owner's list in
--                     Privacy & security — no health value). view_count / last_viewed_at count public opens.
--
-- Health data: owner-only list / revoke, the public read is by token only. Never logged. Twin of
-- backend/database/migrations/2026_10_06_000042_create_health_share_links_table.php (Laravel owns the prod schema
-- until T-M2-27), so `make schema-diff` stays green. Numbered 00042 (after canvas
-- 00040 files) so no environment sees an out-of-order version.

-- +goose Up
CREATE TABLE IF NOT EXISTS `health_share_links` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `token_hash` char(64) NOT NULL,
  `payload` mediumtext DEFAULT NULL,
  `sections` json NOT NULL,
  `range_from` date NOT NULL,
  `range_to` date NOT NULL,
  `expires_at` timestamp NULL DEFAULT NULL,
  `revoked_at` timestamp NULL DEFAULT NULL,
  `view_count` int(10) unsigned NOT NULL DEFAULT 0,
  `last_viewed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `health_share_links_token_hash_unique` (`token_hash`),
  KEY `health_share_links_user_id_created_at_index` (`user_id`,`created_at`),
  KEY `health_share_links_expires_at_index` (`expires_at`),
  CONSTRAINT `health_share_links_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `health_share_links`;

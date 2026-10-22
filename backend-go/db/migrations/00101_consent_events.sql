-- 00101_consent_events.sql — the consent ledger (T-M10-23, D-174; SPEC v6 §4.2; DECISIONS.md H-13, H-18).
--
-- consent_events   append-only: one row per grant or revoke of a consent, the legal evidence of what the user agreed
--                  to and when. `user_consents` stays the current-state projection (one row per user and code, read
--                  by every consent gate); internal/consent writes the event and the projection in one transaction.
--                  consent   catalog code (internal/consent) or an evidence-only scoped code (telemed_share, share_link,
--                            assistant_profile_items)
--                  version   the text version granted or revoked (0 = granted before texts were versioned, B-N1-12)
--                  action    grant | revoke
--                  scope     JSON, NULL for a whole consent: {items} for assistant switches, {booking_id, doctor_id,
--                            items} for a telemed share, {link_id, expires_at, items} for a share link
--                  source    signup | settings | feature | family | telemed | sharelink | backfill | admin
--                  created_at Tehran wall-clock of the act
--
-- Append-only is enforced by the database: BEFORE UPDATE and BEFORE DELETE triggers SIGNAL. Deleting the user still
-- removes her events — a foreign-key cascade does not fire triggers (account deletion keeps working; proven in
-- db/migrations/consent_events_int_test.go).
--
-- Backfill: one `grant` event per granted (or once granted) user_consents row at its granted_at and a `revoke` event
-- at its revoked_at, source `backfill`, oldest first. A (user, code) pair that already has events is skipped, so a
-- database where the Laravel twin ran first (prod at cutover) is not backfilled twice.
--
-- The triggers are created under the session sql_mode and connection collation of Laravel's strict MySQL connection
-- (both restored afterwards), so mariadb-dump prints them the same on both sides. They go through PREPARE because
-- sqlc parses these files as the schema and its parser has no CREATE TRIGGER.
--
-- Twin of backend/database/migrations/2026_10_22_000101_create_consent_events_table.php (Laravel owns the prod schema
-- until T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built table.

-- +goose Up
CREATE TABLE IF NOT EXISTS `consent_events` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `consent` varchar(64) NOT NULL,
  `version` int(10) unsigned NOT NULL,
  `action` varchar(8) NOT NULL,
  `scope` json DEFAULT NULL,
  `source` varchar(24) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `consent_events_user_id_consent_id_index` (`user_id`,`consent`,`id`),
  CONSTRAINT `consent_events_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @consent_events_sql_mode = @@SESSION.sql_mode;
SET @consent_events_collation = @@SESSION.collation_connection;
SET SESSION collation_connection = 'utf8mb4_unicode_ci';
SET SESSION sql_mode = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION';

SET @consent_events_ddl = 'CREATE TRIGGER IF NOT EXISTS `consent_events_no_update` BEFORE UPDATE ON `consent_events` FOR EACH ROW\n  SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''consent_events is append-only''';
PREPARE consent_events_stmt FROM @consent_events_ddl;
EXECUTE consent_events_stmt;
DEALLOCATE PREPARE consent_events_stmt;

SET @consent_events_ddl = 'CREATE TRIGGER IF NOT EXISTS `consent_events_no_delete` BEFORE DELETE ON `consent_events` FOR EACH ROW\n  SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''consent_events is append-only''';
PREPARE consent_events_stmt FROM @consent_events_ddl;
EXECUTE consent_events_stmt;
DEALLOCATE PREPARE consent_events_stmt;

SET SESSION sql_mode = @consent_events_sql_mode;
SET SESSION collation_connection = @consent_events_collation;

INSERT INTO `consent_events` (`user_id`, `consent`, `version`, `action`, `scope`, `source`, `created_at`)
SELECT b.user_id, b.consent, b.version, b.action, NULL, 'backfill', b.at
FROM (
  SELECT uc.user_id, uc.consent, COALESCE(uc.version, 0) AS version, 'grant' AS action,
         COALESCE(uc.granted_at, uc.created_at) AS at, 0 AS seq, uc.id
  FROM `user_consents` uc
  WHERE uc.granted = 1 OR uc.granted_at IS NOT NULL
  UNION ALL
  SELECT uc.user_id, uc.consent, COALESCE(uc.version, 0), 'revoke', uc.revoked_at, 1, uc.id
  FROM `user_consents` uc
  WHERE uc.revoked_at IS NOT NULL
) b
WHERE NOT EXISTS (SELECT 1 FROM `consent_events` e WHERE e.user_id = b.user_id AND e.consent = b.consent)
ORDER BY b.at, b.id, b.seq;

-- +goose Down
DROP TRIGGER IF EXISTS `consent_events_no_delete`;
DROP TRIGGER IF EXISTS `consent_events_no_update`;
DROP TABLE IF EXISTS `consent_events`;

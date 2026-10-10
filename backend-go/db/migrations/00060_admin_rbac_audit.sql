-- 00060_admin_rbac_audit.sql — admin roles & permissions, DB audit log, temporary health-data access (bloom B-N9-02, D-84).
--
-- admin_roles          a super admin's edit of one role's permission matrix ({capability: "view"|"manage"} JSON).
--                      No row = the code defaults (httpadmin.DefaultPermissions). super / editor are never stored.
-- admin_audit_logs     one row per httpadmin.Audit call: who, action, target, ip, small scalar detail (ids / flags,
--                      never health payloads or free text). Append-only; the admin API only lists and exports it.
-- admin_health_access_requests
--                      a clinical-support admin asks to see ONE user's health data (reason + ticket); another super
--                      approves; the grant expires 30 minutes later (expires_at). Every read is audited.
--
-- Twin of backend/database/migrations/2026_10_10_000060_create_admin_rbac_audit_tables.php (Laravel owns the prod
-- schema until T-M2-27), so `make schema-diff` stays green. IF NOT EXISTS so a database whose Laravel half already
-- ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `admin_roles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `role` varchar(32) NOT NULL,
  `permissions` text NOT NULL,
  `updated_by` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `admin_roles_role_unique` (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `admin_audit_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `admin_id` bigint(20) unsigned DEFAULT NULL,
  `action` varchar(64) NOT NULL,
  `target_type` varchar(64) NOT NULL,
  `target_id` bigint(20) unsigned NOT NULL DEFAULT 0,
  `ip` varchar(45) NOT NULL DEFAULT '',
  `detail` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `admin_audit_logs_created_at_index` (`created_at`),
  KEY `admin_audit_logs_admin_id_created_at_index` (`admin_id`,`created_at`),
  KEY `admin_audit_logs_target_type_target_id_index` (`target_type`,`target_id`),
  KEY `admin_audit_logs_action_index` (`action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `admin_health_access_requests` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `admin_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `reason` varchar(500) NOT NULL,
  `ticket` varchar(64) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `decided_by` bigint(20) unsigned DEFAULT NULL,
  `decided_at` timestamp NULL DEFAULT NULL,
  `expires_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `admin_health_access_requests_admin_id_user_id_status_index` (`admin_id`,`user_id`,`status`),
  KEY `admin_health_access_requests_status_created_at_index` (`status`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `admin_health_access_requests`;
DROP TABLE IF EXISTS `admin_audit_logs`;
DROP TABLE IF EXISTS `admin_roles`;

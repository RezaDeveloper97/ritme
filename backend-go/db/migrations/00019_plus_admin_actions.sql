-- 00019_plus_admin_actions.sql — the Ritme Plus admin ledger (B-N2-09, bloom/N2).
--
--   plus_admin_actions   one row per admin mutation in the «اشتراک‌ها و پرداخت» module: plan / discount-code /
--                        settings changes, refunds (through the gateway or marked manually) and subscription
--                        extensions. It is the durable audit trail of money-changing actions (the slog "admin audit"
--                        line is written too) and the only place a manual refund's note and a gateway refund id live.
--
-- Columns:
--   admin_id          the acting admin (no FK: the row outlives a deleted admin account).
--   action            plan.create | plan.update | plan.delete | plan.deactivate | discount.create | discount.update |
--                     discount.delete | discount.deactivate | settings.update | invoice.refund |
--                     invoice.refund_manual | subscription.extend
--   target_type/_id   plan | discount_code | settings | invoice | subscription, and the row id (0 for settings).
--   user_id           the subscriber an invoice / subscription action concerns (no FK: kept for the money trail).
--   amount_rials      refunded amount; days = days added by an extension.
--   gateway / gateway_ref   provider id and its refund id (gateway refunds only).
--   note              the admin's reason (required for manual refunds and extensions); never card data.
--   details           small JSON of the changed fields (old → new price, percent, …).
--
-- Twin of backend/database/migrations/2026_10_01_000019_create_plus_admin_actions_table.php so `make schema-diff`
-- stays green. 00016 is intentionally unused, 00017 belongs to another work stream.

-- +goose Up
CREATE TABLE IF NOT EXISTS `plus_admin_actions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `admin_id` bigint(20) unsigned DEFAULT NULL,
  `action` varchar(48) NOT NULL,
  `target_type` varchar(32) NOT NULL,
  `target_id` bigint(20) unsigned NOT NULL DEFAULT 0,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `amount_rials` bigint(20) unsigned DEFAULT NULL,
  `days` int(10) unsigned DEFAULT NULL,
  `gateway` varchar(32) DEFAULT NULL,
  `gateway_ref` varchar(191) DEFAULT NULL,
  `note` varchar(500) DEFAULT NULL,
  `details` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `plus_admin_actions_target_type_target_id_index` (`target_type`,`target_id`),
  KEY `plus_admin_actions_user_id_index` (`user_id`),
  KEY `plus_admin_actions_created_at_index` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `plus_admin_actions`;

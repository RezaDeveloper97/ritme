-- 00090_share_include_losses.sql — the per-share loss opt-in (user decision 2026-10-12, CB-REC-06 M2 follow-up).
--
-- health_share_links  +include_losses   whether the owner explicitly chose «شامل سابقه‌ی از دست دادن بارداری» for this
--                                       share (7-day report link or 24h doctor code). Off by default: a share then
--                                       carries no loss-derived «ended» pregnancy row. Even when on, a loss is only a
--                                       neutral «ended» row (no type, date, mood or note). Kept so the owner's list
--                                       shows what each share carries and the public read can enforce it.
--
-- Health data (a flag only). Twin of backend/database/migrations/2026_10_12_000090_add_include_losses_to_health_share_links.php
-- so `make schema-diff` stays green; IF NOT EXISTS / IF EXISTS so it is a no-op where the Laravel twin already ran.

-- +goose Up
ALTER TABLE `health_share_links` ADD COLUMN IF NOT EXISTS `include_losses` tinyint(1) NOT NULL DEFAULT 0 AFTER `sections`;

-- +goose Down
ALTER TABLE `health_share_links` DROP COLUMN IF EXISTS `include_losses`;

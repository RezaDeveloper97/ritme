-- 00117_companion_preferences.sql — «ریتمی همراه» v2: the owner's per-link preferences for her companion.
--
-- companions  + goal          what the owner wants from the link: cycle | support | low_energy | quality_time | info |
--                             unknown (NULL = unknown, the default; codes validated in Go, companion.Goals)
--             + share_pain    separate consent per day signal (default off — privacy by default): the companion home
--             + share_mood    shows a coarse «today» yes/no derived from the owner's health log only for a flag that
--             + share_energy  is on, never a count, code, severity or text
--             + share_sleep
--             + action_prefs  the owner's chosen suggestion kinds, comma-separated in the order she picked them
--                             (food,gift,time,calm,help; ≤ 5 distinct codes, ≤ 24 chars); NULL / '' = none. A plain
--                             list rather than JSON: it is only ever read whole, never queried by element.
--
-- Go-only feature (the Laravel API never serves it), but Laravel owns the prod schema until T-M2-27, so the twin
-- backend/database/migrations/2026_11_08_000117_add_companion_preferences.php adds the same columns and
-- `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built table; IF [NOT] EXISTS so a
-- database whose Laravel half already ran does not fail. Additive: every current reader and writer keeps working.

-- +goose Up
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `goal` varchar(16) DEFAULT NULL;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `share_pain` tinyint(1) NOT NULL DEFAULT 0;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `share_mood` tinyint(1) NOT NULL DEFAULT 0;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `share_energy` tinyint(1) NOT NULL DEFAULT 0;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `share_sleep` tinyint(1) NOT NULL DEFAULT 0;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `action_prefs` varchar(64) DEFAULT NULL;

-- +goose Down
ALTER TABLE `companions` DROP COLUMN IF EXISTS `action_prefs`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `share_sleep`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `share_energy`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `share_mood`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `share_pain`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `goal`;

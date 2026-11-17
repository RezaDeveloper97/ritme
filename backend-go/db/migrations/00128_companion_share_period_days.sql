-- 00128_companion_share_period_days.sql — «ریتمی همراه»: the owner's consent to show her companion the days her period
-- is likely to fall on (his calendar).
--
-- companions  + share_period_days  off by default (privacy by default). A separate, explicit consent on top of the
--                                  cycle grant: F-01 keeps the cycle card to the phase word and «period soon»; this
--                                  switch is the one place a partner sees predicted days, never logged history, cycle
--                                  day/length, lateness or confidence (docs/engine-v6/family.md §4, period_forecast).
--
-- Go-only feature. Laravel owns the prod schema until T-M2-27, so the twin
-- backend/database/migrations/2026_11_17_000128_add_companion_share_period_days.php adds the same column and
-- `make schema-diff` stays green. IF [NOT] EXISTS so a database whose Laravel half already ran does not fail.

-- +goose Up
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `share_period_days` tinyint(1) NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE `companions` DROP COLUMN IF EXISTS `share_period_days`;

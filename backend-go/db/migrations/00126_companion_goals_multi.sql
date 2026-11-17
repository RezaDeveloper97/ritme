-- 00126_companion_goals_multi.sql — «ریتمی همراه»: the owner may pick several goals for a link.
--
-- companions.goal  varchar(16) → varchar(64): now the goals she picked, comma-separated in her order
--                  (cycle,support,low_energy,quality_time — at most 37 chars — or the lone `info`); NULL = none
--                  (unknown). A row written before this holds one code and reads the same, so no data change. A plain
--                  list rather than JSON, like action_prefs: it is only ever read whole. Codes validated in Go
--                  (companion.NormalizeGoals).
--
-- Laravel twin: 2026_11_15_000126_widen_companions_goal.php (schema only; the Laravel API never serves the column).

-- +goose Up
ALTER TABLE `companions` MODIFY `goal` varchar(64) DEFAULT NULL;

-- +goose Down
-- A list written after Up is cut to its first goal (what a single-goal reader would show).
UPDATE `companions` SET `goal` = SUBSTRING_INDEX(`goal`, ',', 1) WHERE `goal` LIKE '%,%';
ALTER TABLE `companions` MODIFY `goal` varchar(16) DEFAULT NULL;

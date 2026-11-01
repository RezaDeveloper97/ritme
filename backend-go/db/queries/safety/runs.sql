-- Persisted nightly safety self-test runs (migration 00105, T-M10-25; SPEC v6 §5.8; written by T-M10-24). One row per
-- scenario; run_id (ULID) groups the rows of one run.

-- name: InsertSafetyScenarioRun :exec
INSERT INTO `safety_scenario_runs` (`run_id`, `scenario`, `passed`, `detail`, `duration_ms`, `created_at`)
VALUES (sqlc.arg(run_id), sqlc.arg(scenario), sqlc.arg(passed), sqlc.narg(detail), sqlc.arg(duration_ms), sqlc.arg(created_at));

-- name: ListRecentSafetyRunIDs :many
-- The latest runs, newest first, with their scenario and failure counts.
SELECT `run_id`, MIN(`created_at`) AS started_at, COUNT(*) AS scenarios,
       CAST(SUM(`passed` = 0) AS SIGNED) AS failed
FROM `safety_scenario_runs`
GROUP BY `run_id`
ORDER BY started_at DESC, `run_id` DESC
LIMIT ?;

-- name: ListSafetyScenarioRunsByRun :many
SELECT * FROM `safety_scenario_runs` WHERE `run_id` = ? ORDER BY `scenario`, `id`;

-- name: ListSafetyScenarioHistory :many
-- One scenario's latest results, newest first.
SELECT * FROM `safety_scenario_runs` WHERE `scenario` = ? ORDER BY `created_at` DESC, `id` DESC LIMIT ?;

-- name: CountSafetyRunsBetween :one
-- Rows written in [from_at, to_at): the nightly job's per-day idempotency key (T-M10-24).
SELECT COUNT(*) FROM `safety_scenario_runs` WHERE `created_at` >= sqlc.arg(from_at) AND `created_at` < sqlc.arg(to_at);

-- name: ListSafetyActiveLanguages :many
-- The active languages (default first): the languages every emergency copy must cover (T-M10-24).
SELECT `code`, `is_default` FROM `languages` WHERE `is_active` = 1 ORDER BY `is_default` DESC, `sort_order`, `id`;

-- name: ListSafetyRulesByKeys :many
-- The message_rules rows of the locked emergency rules, active or not (T-M10-24 live configuration check).
SELECT `id`, `rule_key`, `kind`, `level`, `is_active`, `locked`, `status`, `version`, `approved_version`
FROM `message_rules` WHERE `rule_key` IN (sqlc.slice(rule_keys)) ORDER BY `rule_key`, `id`;

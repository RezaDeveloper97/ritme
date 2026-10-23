-- Automatic dashboard alerts (migration 00105, T-M10-25; SPEC v6 §5.1). Rules are keyed by metric_key; at most one
-- open alert per (rule, subject) — the UNIQUE (rule_id, subject, open_flag) key, open_flag = 1 only while open.

-- name: ListMetricAlertRules :many
SELECT * FROM `metric_alert_rules` ORDER BY `metric_key`;

-- name: ListEnabledMetricAlertRules :many
SELECT * FROM `metric_alert_rules` WHERE `enabled` = 1 ORDER BY `metric_key`;

-- name: GetMetricAlertRuleByKey :one
SELECT * FROM `metric_alert_rules` WHERE `metric_key` = ? LIMIT 1;

-- name: UpsertMetricAlertRule :exec
INSERT INTO `metric_alert_rules` (`metric_key`, `comparator`, `threshold`, `window_days`, `min_volume`, `enabled`,
                                  `created_at`, `updated_at`)
VALUES (sqlc.arg(metric_key), sqlc.arg(comparator), sqlc.arg(threshold), sqlc.arg(window_days), sqlc.narg(min_volume),
        sqlc.arg(enabled), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE `comparator` = VALUES(`comparator`), `threshold` = VALUES(`threshold`),
  `window_days` = VALUES(`window_days`), `min_volume` = VALUES(`min_volume`), `enabled` = VALUES(`enabled`),
  `updated_at` = VALUES(`updated_at`);

-- name: UpdateMetricAlertRuleSettings :execrows
-- The admin edit (T-M10-31): threshold and enabled only.
UPDATE `metric_alert_rules`
SET `threshold` = sqlc.arg(threshold), `enabled` = sqlc.arg(enabled), `updated_at` = sqlc.arg(now)
WHERE `metric_key` = sqlc.arg(metric_key);

-- name: OpenMetricAlert :exec
-- Opens an alert, or refreshes the one already open for (rule, subject) with the latest day and value.
INSERT INTO `metric_alerts` (`rule_id`, `subject`, `day`, `value`, `threshold`, `state`, `opened_at`)
VALUES (sqlc.arg(rule_id), sqlc.arg(subject), sqlc.arg(day), sqlc.arg(value), sqlc.arg(threshold), 'open', sqlc.arg(now))
ON DUPLICATE KEY UPDATE `day` = VALUES(`day`), `value` = VALUES(`value`), `threshold` = VALUES(`threshold`);

-- name: ResolveMetricAlert :execrows
UPDATE `metric_alerts` SET `state` = 'resolved', `resolved_at` = sqlc.arg(now)
WHERE `rule_id` = sqlc.arg(rule_id) AND `subject` = sqlc.arg(subject) AND `state` = 'open';

-- name: ListOpenMetricAlerts :many
-- Newest first, with the rule's key and comparator.
SELECT a.id, a.rule_id, r.metric_key, r.comparator, a.subject, a.day, a.value, a.threshold, a.opened_at
FROM `metric_alerts` a
JOIN `metric_alert_rules` r ON r.id = a.rule_id
WHERE a.state = 'open'
ORDER BY a.opened_at DESC, a.id DESC;

-- name: ListResolvedMetricAlerts :many
SELECT a.id, a.rule_id, r.metric_key, r.comparator, a.subject, a.day, a.value, a.threshold, a.opened_at, a.resolved_at
FROM `metric_alerts` a
JOIN `metric_alert_rules` r ON r.id = a.rule_id
WHERE a.state = 'resolved'
ORDER BY a.resolved_at DESC, a.id DESC
LIMIT ?;

-- name: ListOpenMetricAlertSubjects :many
-- The open (rule, subject) pairs of one rule, so the evaluator resolves those whose condition no longer holds.
SELECT `subject` FROM `metric_alerts` WHERE `rule_id` = ? AND `state` = 'open' ORDER BY `subject`;

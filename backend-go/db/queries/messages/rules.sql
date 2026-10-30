-- Smart messages v2 (bloom B-N9-05, migration 00061): rule-based messages, versions, events.

-- name: CountMessageRules :one
SELECT COUNT(*) FROM `message_rules`
WHERE (sqlc.arg(placement) = '' OR placement = sqlc.arg(placement))
  AND (sqlc.arg(kind) = '' OR kind = sqlc.arg(kind))
  AND (sqlc.arg(status) = '' OR status = sqlc.arg(status));

-- name: ListMessageRules :many
SELECT * FROM `message_rules`
WHERE (sqlc.arg(placement) = '' OR placement = sqlc.arg(placement))
  AND (sqlc.arg(kind) = '' OR kind = sqlc.arg(kind))
  AND (sqlc.arg(status) = '' OR status = sqlc.arg(status))
ORDER BY priority DESC, id ASC
LIMIT ? OFFSET ?;

-- name: GetMessageRule :one
SELECT * FROM `message_rules` WHERE id = ? LIMIT 1;

-- name: InsertMessageRule :execresult
INSERT INTO `message_rules` (title, level, kind, priority, placement, content, variants_enabled, ab_split, conditions,
  exceptions, cap_count, cap_period, cooldown_days, hours_from, hours_to, starts_on, ends_on, audience_percent, goal,
  rule_key, topics, caps, variants, starts_at, ends_at, hour_windows, goal_event,
  status, is_active, version, created_by, created_at, updated_at)
VALUES (sqlc.arg(title), sqlc.arg(level), sqlc.arg(kind), sqlc.arg(priority), sqlc.arg(placement), sqlc.arg(content),
  sqlc.arg(variants_enabled), sqlc.arg(ab_split), sqlc.arg(conditions), sqlc.arg(exceptions), sqlc.narg(cap_count),
  sqlc.narg(cap_period), sqlc.arg(cooldown_days), sqlc.narg(hours_from), sqlc.narg(hours_to), sqlc.narg(starts_on),
  sqlc.narg(ends_on), sqlc.arg(audience_percent), sqlc.narg(goal),
  sqlc.narg(rule_key), sqlc.narg(topics), sqlc.narg(caps), sqlc.narg(variants), sqlc.narg(starts_at), sqlc.narg(ends_at),
  sqlc.narg(hour_windows), sqlc.narg(goal_event),
  'draft', 0, 1, sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateMessageRule :exec
-- T-M10-02 (M-02, M-03): the version always grows; the caller decides the approval state — a text / condition /
-- level / kind / topic / variant-content edit of a medical rule leaves it pending while approved_version keeps
-- serving, an ops-only edit (priority, audience, schedule, hours, caps) keeps the approval.
UPDATE `message_rules` SET title = sqlc.arg(title), level = sqlc.arg(level), kind = sqlc.arg(kind),
  priority = sqlc.arg(priority), placement = sqlc.arg(placement), content = sqlc.arg(content),
  variants_enabled = sqlc.arg(variants_enabled), ab_split = sqlc.arg(ab_split), conditions = sqlc.arg(conditions),
  exceptions = sqlc.arg(exceptions), cap_count = sqlc.narg(cap_count), cap_period = sqlc.narg(cap_period),
  cooldown_days = sqlc.arg(cooldown_days), hours_from = sqlc.narg(hours_from), hours_to = sqlc.narg(hours_to),
  starts_on = sqlc.narg(starts_on), ends_on = sqlc.narg(ends_on), audience_percent = sqlc.arg(audience_percent),
  goal = sqlc.narg(goal), rule_key = sqlc.narg(rule_key), topics = sqlc.narg(topics), caps = sqlc.narg(caps),
  variants = sqlc.narg(variants), starts_at = sqlc.narg(starts_at), ends_at = sqlc.narg(ends_at),
  hour_windows = sqlc.narg(hour_windows), goal_event = sqlc.narg(goal_event),
  status = sqlc.arg(status), approved_by = sqlc.narg(approved_by), approved_at = sqlc.narg(approved_at),
  approved_version = sqlc.narg(approved_version), version = version + 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetMessageRuleStatus :exec
-- approved_version NULL keeps the current one (submit, reject: the last approved version stays live, M-03).
UPDATE `message_rules` SET status = sqlc.arg(status), approved_by = sqlc.narg(approved_by),
  approved_at = sqlc.narg(approved_at), approved_version = COALESCE(sqlc.narg(approved_version), approved_version),
  updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: GetMessageRuleIDByKey :one
SELECT id FROM `message_rules` WHERE rule_key = ? LIMIT 1;

-- name: SetMessageRuleActive :exec
UPDATE `message_rules` SET is_active = sqlc.arg(is_active), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteMessageRule :exec
DELETE FROM `message_rules` WHERE id = ?;

-- name: InsertMessageRuleVersion :exec
INSERT INTO `message_rule_versions` (rule_id, version, snapshot, changed_by, created_at)
VALUES (sqlc.arg(rule_id), sqlc.arg(version), sqlc.arg(snapshot), sqlc.narg(changed_by), sqlc.arg(now));

-- name: GetMessageRuleVersion :one
SELECT * FROM `message_rule_versions` WHERE rule_id = ? AND version = ? LIMIT 1;

-- name: ListServedMessageRuleSnapshots :many
-- T-M10-02 (M-03): the approved snapshots of the active rules whose head waits for re-approval (approved_version is
-- set and is not the head version); rules.ActiveRows serves them instead of the head.
SELECT v.rule_id, v.version, v.snapshot FROM `message_rule_versions` v
JOIN `message_rules` r ON r.id = v.rule_id AND v.version = r.approved_version
WHERE r.is_active = 1 AND r.approved_version IS NOT NULL AND r.approved_version <> r.version;

-- name: ListMessageRuleVersions :many
SELECT * FROM `message_rule_versions` WHERE rule_id = ? ORDER BY version DESC LIMIT 50;

-- name: ListActiveMessageRules :many
-- Active rules a user may be served: medical rules only when approved. Placement '' = every placement.
SELECT * FROM `message_rules`
WHERE is_active = 1 AND (sqlc.arg(placement) = '' OR placement = sqlc.arg(placement))
ORDER BY priority DESC, id ASC;

-- name: InsertMessageEvent :exec
INSERT INTO `message_events` (rule_id, user_id, variant, event, created_at)
VALUES (sqlc.arg(rule_id), sqlc.arg(user_id), sqlc.arg(variant), sqlc.arg(event), sqlc.arg(now));

-- name: CountUserMessageEvents :one
SELECT COUNT(*) FROM `message_events`
WHERE user_id = ? AND rule_id = ? AND event = ? AND created_at >= ?;

-- name: LastUserMessageEvent :one
SELECT created_at FROM `message_events`
WHERE user_id = ? AND rule_id = ? AND event = ? ORDER BY created_at DESC LIMIT 1;

-- name: MessageRuleEventStats :many
SELECT rule_id, event, COUNT(*) AS total FROM `message_events`
WHERE created_at >= ? GROUP BY rule_id, event;

-- name: MessageRuleVariantStats :many
SELECT variant, event, COUNT(*) AS total FROM `message_events`
WHERE rule_id = ? AND created_at >= ? GROUP BY variant, event;

-- name: GetMessageUserName :one
SELECT name FROM `users` WHERE id = ? LIMIT 1;

-- name: ListUserMessageEventStats :many
-- One grouped read of a user's history (T-M9-13, replaces the per-rule LastUserMessageEvent + CountUserMessageEvents
-- probes; extended in T-M10-04): the latest event and the counts in the three cap windows (day, week, cycle) per rule,
-- event and slot, with the rule's kind. rules.Select (legacy) reads its candidate rules (`all_rules` = 0); selection v6
-- reads every rule of the user (`all_rules` = 1) for the per-slot counts and the last safety message shown. A legacy
-- event without a slot counts for the rule's placement; a deleted rule has kind ''.
SELECT e.rule_id, e.event,
  COALESCE(e.slot, r.placement, '') AS slot,
  COALESCE(r.kind, '') AS kind,
  CAST(MAX(e.created_at) AS DATETIME) AS last_at,
  CAST(SUM(e.created_at >= sqlc.arg(day_since)) AS SIGNED) AS day_count,
  CAST(SUM(e.created_at >= sqlc.arg(week_since)) AS SIGNED) AS week_count,
  CAST(SUM(e.created_at >= sqlc.arg(cycle_since)) AS SIGNED) AS cycle_count
FROM `message_events` e
LEFT JOIN `message_rules` r ON r.id = e.rule_id
WHERE e.user_id = sqlc.arg(user_id) AND (sqlc.arg(all_rules) = 1 OR e.rule_id IN (sqlc.slice(rule_ids)))
GROUP BY e.rule_id, e.event, COALESCE(e.slot, r.placement, ''), COALESCE(r.kind, '');

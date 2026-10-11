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
  status, is_active, version, created_by, created_at, updated_at)
VALUES (sqlc.arg(title), sqlc.arg(level), sqlc.arg(kind), sqlc.arg(priority), sqlc.arg(placement), sqlc.arg(content),
  sqlc.arg(variants_enabled), sqlc.arg(ab_split), sqlc.arg(conditions), sqlc.arg(exceptions), sqlc.narg(cap_count),
  sqlc.narg(cap_period), sqlc.arg(cooldown_days), sqlc.narg(hours_from), sqlc.narg(hours_to), sqlc.narg(starts_on),
  sqlc.narg(ends_on), sqlc.arg(audience_percent), sqlc.narg(goal), 'draft', 0, 1, sqlc.arg(created_by),
  sqlc.arg(now), sqlc.arg(now));

-- name: UpdateMessageRule :exec
UPDATE `message_rules` SET title = sqlc.arg(title), level = sqlc.arg(level), kind = sqlc.arg(kind),
  priority = sqlc.arg(priority), placement = sqlc.arg(placement), content = sqlc.arg(content),
  variants_enabled = sqlc.arg(variants_enabled), ab_split = sqlc.arg(ab_split), conditions = sqlc.arg(conditions),
  exceptions = sqlc.arg(exceptions), cap_count = sqlc.narg(cap_count), cap_period = sqlc.narg(cap_period),
  cooldown_days = sqlc.arg(cooldown_days), hours_from = sqlc.narg(hours_from), hours_to = sqlc.narg(hours_to),
  starts_on = sqlc.narg(starts_on), ends_on = sqlc.narg(ends_on), audience_percent = sqlc.arg(audience_percent),
  goal = sqlc.narg(goal), status = 'draft', approved_by = NULL, approved_at = NULL, version = version + 1,
  updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetMessageRuleStatus :exec
UPDATE `message_rules` SET status = sqlc.arg(status), approved_by = sqlc.narg(approved_by),
  approved_at = sqlc.narg(approved_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetMessageRuleActive :exec
UPDATE `message_rules` SET is_active = sqlc.arg(is_active), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteMessageRule :exec
DELETE FROM `message_rules` WHERE id = ?;

-- name: InsertMessageRuleVersion :exec
INSERT INTO `message_rule_versions` (rule_id, version, snapshot, changed_by, created_at)
VALUES (sqlc.arg(rule_id), sqlc.arg(version), sqlc.arg(snapshot), sqlc.narg(changed_by), sqlc.arg(now));

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

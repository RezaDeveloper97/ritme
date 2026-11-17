-- Alerts inbox (T-M10-07, D-164; SPEC §5.4.1, DECISIONS.md M-09, M-16): the open rows the evaluation diffs with, the
-- lifecycle writes (merge, resolve, ack, visit note, dismiss), the list pages, the daily sweep and the push slot.
-- Every user read and write is scoped by user_id in the query itself.

-- name: ListUserOpenInbox :many
-- The user's open rows (active, acknowledged, added_to_visit_note): at most one per rule (UNIQUE user, rule, open_flag).
SELECT id, user_id, rule_id, version, rule_key, kind, level, state, evidence_key, hit_count, evidence,
  first_at, last_at, acked_at, noted_at
FROM `message_inbox`
WHERE user_id = sqlc.arg(user_id) AND open_flag = 1
ORDER BY id;

-- name: InsertInboxItem :execresult
-- A new match: an active row. INSERT IGNORE: a concurrent request that already opened the row for the same rule wins
-- (the unique open key), and this one changes nothing.
INSERT IGNORE INTO `message_inbox`
  (user_id, rule_id, version, rule_key, slot, kind, level, state, evidence_key, hit_count, evidence, first_at, last_at,
   created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(rule_id), sqlc.arg(version), sqlc.narg(rule_key), 'alerts_inbox', sqlc.arg(kind),
   sqlc.arg(level), 'active', sqlc.arg(evidence_key), 1, sqlc.arg(evidence), sqlc.arg(now), sqlc.arg(now),
   sqlc.arg(now), sqlc.arg(now));

-- name: MergeInboxItem :execrows
-- The same message fired again with new evidence (M-16): count + 1, last_at, the evidence and the served version
-- move; the state comes from the caller (an acknowledged urgent row returns to active and loses its ack).
UPDATE `message_inbox`
SET hit_count = hit_count + 1, last_at = sqlc.arg(now), evidence_key = sqlc.arg(evidence_key),
  evidence = sqlc.arg(evidence), version = sqlc.arg(version), level = sqlc.arg(level), state = sqlc.arg(state),
  acked_at = sqlc.narg(acked_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND open_flag = 1;

-- name: ResolveInboxItem :execrows
-- The condition no longer holds (all its facts were known): the row becomes history.
UPDATE `message_inbox`
SET state = 'resolved', resolved_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND open_flag = 1;

-- name: GetUserInboxItem :one
SELECT id, user_id, rule_id, version, rule_key, kind, level, state, evidence_key, hit_count, evidence,
  first_at, last_at, acked_at, noted_at, resolved_at, archived_at
FROM `message_inbox`
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
LIMIT 1;

-- name: AckInboxItem :execrows
-- «دیدم، ممنون»: an open row becomes acknowledged.
UPDATE `message_inbox`
SET state = 'acknowledged', acked_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND open_flag = 1;

-- name: DismissInboxItem :execrows
-- Close a row now (open or resolved) → archived. The urgent ack rule is checked by the caller.
UPDATE `message_inbox`
SET state = 'archived', archived_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND state IN ('active', 'acknowledged', 'added_to_visit_note', 'resolved');

-- name: ListUserInboxOpenPage :many
-- GET /alerts?state=open: urgent → follow_up → suggestion → info, then last_at DESC, id DESC; keyset after
-- (after_rank, after_at, after_id) when has_cursor = 1.
SELECT id, user_id, rule_id, version, rule_key, kind, level, state, evidence_key, hit_count, evidence,
  first_at, last_at, acked_at, noted_at, resolved_at, archived_at,
  CAST(CASE level WHEN 'follow_soon' THEN 0 WHEN 'worth_followup' THEN 1 WHEN 'suggestion' THEN 2 ELSE 3 END AS SIGNED) AS level_rank
FROM `message_inbox`
WHERE user_id = sqlc.arg(user_id) AND open_flag = 1
  AND (CAST(sqlc.arg(has_cursor) AS SIGNED) = 0
    OR CASE level WHEN 'follow_soon' THEN 0 WHEN 'worth_followup' THEN 1 WHEN 'suggestion' THEN 2 ELSE 3 END > CAST(sqlc.arg(after_rank) AS SIGNED)
    OR (CASE level WHEN 'follow_soon' THEN 0 WHEN 'worth_followup' THEN 1 WHEN 'suggestion' THEN 2 ELSE 3 END = CAST(sqlc.arg(after_rank) AS SIGNED)
      AND (last_at < sqlc.arg(after_at) OR (last_at = sqlc.arg(after_at) AND id < sqlc.arg(after_id)))))
ORDER BY level_rank, last_at DESC, id DESC
LIMIT ?;

-- name: ListUserInboxHistoryPage :many
-- GET /alerts?state=history: resolved and archived rows, same order and keyset as the open page.
SELECT id, user_id, rule_id, version, rule_key, kind, level, state, evidence_key, hit_count, evidence,
  first_at, last_at, acked_at, noted_at, resolved_at, archived_at,
  CAST(CASE level WHEN 'follow_soon' THEN 0 WHEN 'worth_followup' THEN 1 WHEN 'suggestion' THEN 2 ELSE 3 END AS SIGNED) AS level_rank
FROM `message_inbox`
WHERE user_id = sqlc.arg(user_id) AND state IN ('resolved', 'archived')
  AND (CAST(sqlc.arg(has_cursor) AS SIGNED) = 0
    OR CASE level WHEN 'follow_soon' THEN 0 WHEN 'worth_followup' THEN 1 WHEN 'suggestion' THEN 2 ELSE 3 END > CAST(sqlc.arg(after_rank) AS SIGNED)
    OR (CASE level WHEN 'follow_soon' THEN 0 WHEN 'worth_followup' THEN 1 WHEN 'suggestion' THEN 2 ELSE 3 END = CAST(sqlc.arg(after_rank) AS SIGNED)
      AND (last_at < sqlc.arg(after_at) OR (last_at = sqlc.arg(after_at) AND id < sqlc.arg(after_id)))))
ORDER BY level_rank, last_at DESC, id DESC
LIMIT ?;

-- name: ListOpenInboxUsers :many
-- messages.inbox_sweep: one keyset page of the users with an open row.
SELECT DISTINCT user_id FROM `message_inbox`
WHERE open_flag = 1 AND user_id > sqlc.arg(after_id)
ORDER BY user_id
LIMIT ?;

-- name: ArchiveStaleInbox :execrows
-- messages.inbox_sweep: resolved, acknowledged and noted rows whose last hit is before the cutoff → archived, in chunks.
UPDATE `message_inbox`
SET state = 'archived', archived_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE state IN ('resolved', 'acknowledged', 'added_to_visit_note') AND last_at < sqlc.arg(cutoff)
LIMIT 5000;

-- name: ListActivePushUsers :many
-- messages.push_slot: one page (keyset on users.id) of the users active in the last 30 days (A-06: a token issued or
-- refreshed, a session day, a daily log, a v2 log entry, a period row or an analytics event) with their stored zone.
-- Same definition as ListActiveSnapshotUsers (engine snapshots).
SELECT u.id AS user_id, p.timezone
FROM `users` u
LEFT JOIN `user_profiles` p ON p.user_id = u.id
WHERE u.id > sqlc.arg(after_id)
  AND u.id IN (
    SELECT t.user_id FROM `oauth_access_tokens` t WHERE t.user_id IS NOT NULL AND t.updated_at >= sqlc.arg(since)
    UNION SELECT a.user_id FROM `user_activity_days` a WHERE a.`day` >= sqlc.arg(since_day)
    UNION SELECT l.user_id FROM `daily_health_logs` l WHERE l.log_date >= sqlc.arg(since_day)
    UNION SELECT e.user_id FROM `health_log_entries` e WHERE e.updated_at >= sqlc.arg(since)
    UNION SELECT c.user_id FROM `cycle_histories` c WHERE c.updated_at >= sqlc.arg(since)
    UNION SELECT v.user_id FROM `analytics_events` v WHERE v.occurred_on >= sqlc.arg(since_day)
  )
ORDER BY u.id
LIMIT ?;

-- name: InsertPushSlotEvent :exec
-- The push slot's shown row, written synchronously so the slot caps (1 a day, 3 a week) see it at once.
INSERT INTO `message_events` (rule_id, user_id, version, variant, slot, event, created_at)
VALUES (sqlc.arg(rule_id), sqlc.arg(user_id), sqlc.arg(version), sqlc.arg(variant), 'push', 'impression', sqlc.arg(now));

-- name: InsertPushNotification :exec
-- The push slot's in-app notification (M-09): neutral title/body {lang: text}, never a health detail.
INSERT INTO `user_notifications` (user_id, type, title, body, action_url, data, created_at, updated_at)
VALUES (sqlc.arg(user_id), 'message_push', sqlc.arg(title), sqlc.arg(body), sqlc.narg(action_url), sqlc.arg(data),
  sqlc.arg(now), sqlc.arg(now));

-- The v1 pregnancy alert endpoints (PregnancyAlertController: GET /pregnancy/alerts, /alerts/summary, /alerts/{id},
-- POST /alerts/{id}/read|dismiss, /alerts/read-all) over two sources (T-M10-08b, D-165):
--   * the alerts inbox: every pregnancy alert since T-M10-08 is a message_inbox row of a preg.* rule (one open row per
--     rule; read = acknowledged, dismiss = archived; resolved rows are history the v1 list still shows, as Laravel
--     showed every alert not dismissed);
--   * the legacy pregnancy_alerts rows (PregnancyAlert, App\Models\PregnancyAlert) the retired v1 rules wrote before
--     T-M10-08 (Laravel included). Nothing inserts them any more; they keep their read / dismiss lifecycle so an
--     unread legacy emergency is never hidden. v2 rows (alert_type 'v2:<rule>') were moved into the inbox by goose
--     00113 and are left out here, so no alert is listed twice.
-- Every query is scoped by user_id. Legacy scopes: active = is_dismissed 0, unread = is_read 0; Laravel's
-- orderBy('created_at', 'desc') (no tie-breaker).

-- ---------------- legacy pregnancy_alerts (read / lifecycle only) ----------------

-- name: ListActiveAlerts :many
SELECT * FROM `pregnancy_alerts`
WHERE user_id = sqlc.arg(user_id) AND is_dismissed = 0 AND alert_type NOT LIKE 'v2:%'
  AND (sqlc.arg(level) = '' OR alert_level = sqlc.arg(level))
  AND (sqlc.arg(unread_only) = 0 OR is_read = 0)
ORDER BY created_at DESC;

-- name: CountActiveAlerts :one
SELECT
  COUNT(*) AS total,
  CAST(COALESCE(SUM(is_read = 0), 0) AS SIGNED) AS unread,
  CAST(COALESCE(SUM(alert_level = 'emergency'), 0) AS SIGNED) AS emergency,
  CAST(COALESCE(SUM(alert_level = 'emergency' AND is_read = 0), 0) AS SIGNED) AS unread_emergency,
  CAST(COALESCE(SUM(alert_level = 'warning'), 0) AS SIGNED) AS warning,
  CAST(COALESCE(SUM(alert_level = 'info'), 0) AS SIGNED) AS info
FROM `pregnancy_alerts` WHERE user_id = ? AND is_dismissed = 0 AND alert_type NOT LIKE 'v2:%';

-- name: LatestUnreadEmergencyAlert :one
SELECT * FROM `pregnancy_alerts`
WHERE user_id = ? AND alert_level = 'emergency' AND is_read = 0 AND is_dismissed = 0 AND alert_type NOT LIKE 'v2:%'
ORDER BY created_at DESC LIMIT 1;

-- name: GetAlert :one
SELECT * FROM `pregnancy_alerts` WHERE user_id = ? AND id = ? AND alert_type NOT LIKE 'v2:%' LIMIT 1;

-- name: MarkAlertRead :exec
UPDATE `pregnancy_alerts` SET is_read = 1, read_at = ?, updated_at = ? WHERE id = ?;

-- name: DismissAlert :exec
UPDATE `pregnancy_alerts` SET is_dismissed = 1, dismissed_at = ?, updated_at = ? WHERE id = ?;

-- name: MarkAllAlertsRead :execrows
UPDATE `pregnancy_alerts` SET is_read = 1, read_at = ?, updated_at = ?
WHERE user_id = ? AND is_read = 0 AND alert_type NOT LIKE 'v2:%';

-- name: DismissOpenAlerts :execrows
-- A pregnancy loss (CB-LOSS-01) closes every open legacy alert: none may resurface as a badge or nudge.
UPDATE `pregnancy_alerts`
SET is_read = 1, read_at = COALESCE(read_at, sqlc.arg(now)), is_dismissed = 1, dismissed_at = sqlc.arg(now),
    updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id) AND is_dismissed = 0;

-- ---------------- the alerts inbox (preg.* rows) ----------------

-- name: ListPregnancyInboxAlerts :many
-- The user's preg.* inbox rows the v1 list shows: open (active, acknowledged, added_to_visit_note) and resolved —
-- everything not archived (= not dismissed). Newest activity first.
SELECT id, rule_id, rule_key, level, state, evidence, first_at, last_at, acked_at, noted_at, resolved_at, archived_at
FROM `message_inbox`
WHERE user_id = sqlc.arg(user_id) AND rule_key LIKE 'preg.%'
  AND state IN ('active', 'acknowledged', 'added_to_visit_note', 'resolved')
ORDER BY last_at DESC, id DESC;

-- name: GetPregnancyInboxAlert :one
SELECT id, rule_id, rule_key, level, state, evidence, first_at, last_at, acked_at, noted_at, resolved_at, archived_at
FROM `message_inbox`
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND rule_key LIKE 'preg.%'
LIMIT 1;

-- name: ReadPregnancyInboxAlert :execrows
-- v1 read: an active row becomes acknowledged (the inbox ack); a row already read stays as it is.
UPDATE `message_inbox`
SET state = 'acknowledged', acked_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND rule_key LIKE 'preg.%' AND state = 'active';

-- name: ReadAllPregnancyInboxAlerts :execrows
-- v1 read-all: every active preg.* row becomes acknowledged.
UPDATE `message_inbox`
SET state = 'acknowledged', acked_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id) AND rule_key LIKE 'preg.%' AND state = 'active';

-- name: DismissPregnancyInboxAlert :execrows
-- v1 dismiss: the row is archived. The user dismissed it herself, so an unacknowledged row is acknowledged with it
-- (the inbox's urgent-ack rule is met by the explicit action). Should the condition still hold, the next evaluation
-- opens a new row (M-16: archived is history).
UPDATE `message_inbox`
SET state = 'archived', acked_at = COALESCE(acked_at, sqlc.arg(now)), archived_at = sqlc.arg(now),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND rule_key LIKE 'preg.%'
  AND state IN ('active', 'acknowledged', 'added_to_visit_note', 'resolved');

-- name: ArchivePregnancyInboxAlerts :execrows
-- A pregnancy loss (CB-LOSS-01): every preg.* row not yet archived is archived, so no pregnancy alert outlives the
-- pregnancy as a badge, modal or nudge.
UPDATE `message_inbox`
SET state = 'archived', archived_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id) AND rule_key LIKE 'preg.%'
  AND state IN ('active', 'acknowledged', 'added_to_visit_note', 'resolved');

-- name: GetPregnancyAlertRule :one
-- The message rule of an inbox row (its texts, whether it is served).
SELECT * FROM `message_rules` WHERE id = sqlc.arg(id) LIMIT 1;

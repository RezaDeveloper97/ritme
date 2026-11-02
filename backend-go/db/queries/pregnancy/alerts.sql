-- The v1 pregnancy alert endpoints (PregnancyAlertController: GET /pregnancy/alerts, /alerts/summary, /alerts/{id},
-- POST /alerts/{id}/read|dismiss, /alerts/read-all) over the alerts inbox (T-M10-08b, D-165): every pregnancy alert is a
-- message_inbox row of a preg.* rule (one open row per rule; read = acknowledged, dismiss = archived; resolved rows are
-- history the v1 list still shows, as Laravel showed every alert not dismissed). The legacy pregnancy_alerts rows the
-- retired v1 and v2 rules wrote were moved into the inbox (goose 00113 the v2 rows, 00114 the v1 rows, T-M10-08c):
-- nothing reads or writes that table any more. Every query is scoped by user_id.

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

-- Pregnancy v2 read model (T-M7-02, internal/pregnancy/v2): the Today screen's extras and the message
-- rows it reads. Every user-scoped query filters by user_id.

-- name: GetV2TodayExtras :one
-- One row for the Today screen: the unread (not dismissed) alert count, the next upcoming non-cancelled
-- M3 appointment at or after `now`, and the first active care-plan item whose week window has not ended
-- (the next-visit fallback). The users row only anchors the single result row.
SELECT
  (SELECT COUNT(*) FROM `pregnancy_alerts` pa
    WHERE pa.user_id = sqlc.arg(user_id) AND pa.is_dismissed = 0 AND pa.is_read = 0) AS unread_alerts,
  a.id AS appointment_id,
  a.title AS appointment_title,
  a.scheduled_at AS appointment_at,
  a.meta AS appointment_meta,
  ci.`key` AS care_item_key,
  ci.title AS care_item_title,
  ci.week_from AS care_item_week_from,
  ci.week_to AS care_item_week_to
FROM `users` u
LEFT JOIN `reminders` a ON a.id = (
  SELECT r.id FROM `reminders` r
  WHERE r.user_id = sqlc.arg(user_id) AND r.`type` = 'appointment' AND r.is_active = 1
    AND r.scheduled_at >= sqlc.arg(now)
    AND COALESCE(JSON_UNQUOTE(JSON_EXTRACT(r.meta, '$.status')), 'scheduled') <> 'cancelled'
  ORDER BY r.scheduled_at, r.id
  LIMIT 1
)
LEFT JOIN `pregnancy_care_items` ci ON ci.id = (
  SELECT c.id FROM `pregnancy_care_items` c
  WHERE c.is_active = 1 AND c.week_to >= sqlc.arg(current_week)
  ORDER BY c.week_from, c.sort_order, c.id
  LIMIT 1
)
WHERE u.id = sqlc.arg(user_id);

-- name: ListV2MessagePayloads :many
-- The live message_contents rows of one group/item in the given locales (request locale + default
-- language, for the fallback) — the week tip and the setup templates.
SELECT locale, payload FROM `message_contents`
WHERE `group` = sqlc.arg(message_group) AND item_key = sqlc.arg(item_key)
  AND is_active = 1 AND is_approved = 1 AND locale IN (sqlc.slice(locales));

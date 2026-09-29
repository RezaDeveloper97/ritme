-- Pregnancy v2 calendar (T-M7-05, internal/pregnancy/v2/calendar): the user's M3 appointments that are
-- not cancelled, soonest first. Month filtering and care-item linking (meta.care_item_key) happen in Go.

-- name: ListV2CalendarAppointments :many
-- is_active is the reminder bell: off → the visit stays, but it has no «remind before» (QA 2026-09-29-c L4).
SELECT id, title, scheduled_at, meta, is_active FROM `reminders`
WHERE user_id = sqlc.arg(user_id) AND `type` = 'appointment' AND scheduled_at IS NOT NULL
  AND COALESCE(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.status')), 'scheduled') <> 'cancelled'
ORDER BY scheduled_at, id;

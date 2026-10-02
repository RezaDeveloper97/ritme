-- Pregnancy loss path (CB-LOSS-01, goose 00028): /api/v1/loss*, Go only. Every statement is scoped by user_id.

-- name: GetLatestLoss :one
-- The user's newest loss event (the one the follow-up screens work on).
SELECT * FROM `pregnancy_losses`
WHERE user_id = ?
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: LockLossOwner :one
-- Serialises POST /loss per user (a double tap or two tabs): the second transaction waits on the user row and then
-- sees the first one's record as today's, so it corrects it instead of inserting a second loss.
SELECT id FROM `users` WHERE id = ? FOR UPDATE;

-- name: DeleteLoss :execrows
-- DELETE /loss: the record goes with its moods (FK cascade) and its encrypted note. Pregnancy stays stopped.
DELETE FROM `pregnancy_losses` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: CountLosses :one
SELECT COUNT(*) FROM `pregnancy_losses` WHERE user_id = ?;

-- name: HasLoss :one
SELECT EXISTS(SELECT 1 FROM `pregnancy_losses` WHERE user_id = ?) AS has_loss;

-- name: InsertLoss :execlastid
INSERT INTO `pregnancy_losses` (
  user_id, loss_type, occurred_on, notify_companion, content_stopped_at, paused_reminders, created_at, updated_at
) VALUES (
  sqlc.arg(user_id), sqlc.arg(loss_type), sqlc.narg(occurred_on), sqlc.arg(notify_companion),
  sqlc.arg(now), sqlc.narg(paused_reminders), sqlc.arg(now), sqlc.arg(now)
);

-- name: UpdateLossEvent :exec
-- A second POST /loss on the same day corrects the event instead of recording another loss.
UPDATE `pregnancy_losses`
SET loss_type = sqlc.arg(loss_type), occurred_on = sqlc.narg(occurred_on), notify_companion = sqlc.arg(notify_companion),
    content_stopped_at = sqlc.arg(now), paused_reminders = sqlc.narg(paused_reminders), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: MarkCompanionNotified :exec
UPDATE `pregnancy_losses` SET companion_notified_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: UpdateLossFollowup :exec
UPDATE `pregnancy_losses`
SET bleeding_stopped_on = sqlc.narg(bleeding_stopped_on), beta_next_on = sqlc.narg(beta_next_on),
    beta_negative_on = sqlc.narg(beta_negative_on), beta_reminder_id = sqlc.narg(beta_reminder_id),
    visit_reminder_id = sqlc.narg(visit_reminder_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: SetLossNextStep :exec
UPDATE `pregnancy_losses` SET next_step = sqlc.arg(next_step), next_step_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: SetLossNote :exec
-- private_note is ciphertext (internal/loss/note.go); NULL clears it.
UPDATE `pregnancy_losses`
SET private_note = sqlc.narg(private_note), note_updated_at = sqlc.narg(note_updated_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: UpsertLossMood :exec
INSERT INTO `pregnancy_loss_moods` (user_id, loss_id, log_date, mood, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(loss_id), sqlc.arg(log_date), sqlc.arg(mood), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE mood = VALUES(mood), updated_at = VALUES(updated_at);

-- name: ListLossMoodsSince :many
-- The loss's mood check-ins on or after date_from, newest first.
SELECT log_date, mood FROM `pregnancy_loss_moods`
WHERE user_id = sqlc.arg(user_id) AND loss_id = sqlc.arg(loss_id) AND log_date >= sqlc.arg(date_from)
ORDER BY log_date DESC;

-- name: ListPregnancyReminders :many
-- The reminders that only make sense while pregnant: active medications with duration «تا پایان بارداری»
-- (pregnancy_end) and upcoming, not cancelled pregnancy-visit appointments (linked to a care item, T-M7-05).
SELECT id, `type` FROM `reminders`
WHERE user_id = sqlc.arg(user_id) AND (
  (`type` = 'medication' AND is_active = 1
    AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.duration')) = 'pregnancy_end')
  OR (`type` = 'appointment' AND scheduled_at >= sqlc.arg(now)
    AND JSON_TYPE(JSON_EXTRACT(meta, '$.care_item_key')) = 'STRING'
    AND COALESCE(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.status')), 'scheduled') <> 'cancelled')
)
ORDER BY id;

-- name: PauseMedicationReminder :exec
-- The medication stays in the list, switched off (the user may switch it on again).
UPDATE `reminders` SET is_active = 0, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND `type` = 'medication';

-- name: CancelAppointmentReminder :exec
-- A pregnancy visit (or a loss follow-up appointment no longer needed) is cancelled, its bell switched off.
UPDATE `reminders`
SET is_active = 0, meta = JSON_SET(COALESCE(meta, '{}'), '$.status', 'cancelled'), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND `type` = 'appointment';

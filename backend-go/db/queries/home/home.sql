-- Home page (backend/app/Services/HomePage/**, DailyChallengeService, HomeController).
-- The cycle inputs (profile, cycle_histories, the daily-log window) come from the cycle service
-- (db/queries/cycle/engine.sql); these are the home-only reads and the toggle / notification writes.
-- `whereDate(col, $d)` on a DATE column is plain equality; ties that Laravel leaves to MariaDB come
-- back in primary-key order and are made explicit with `id`.

-- name: CountUnreadNotifications :one
-- $user->appNotifications()->unread()->count().
SELECT COUNT(*) FROM `user_notifications` WHERE user_id = ? AND read_at IS NULL;

-- name: ListTaskTemplatesForPhase :many
-- TaskTemplate::active()->forPhase($phase): cycle_phase IS NULL, or = $phase when $phase is truthy.
SELECT id, `key`, title, description, category, icon
FROM `task_templates`
WHERE is_active = 1
  AND (cycle_phase IS NULL OR (CAST(sqlc.arg(has_phase) AS SIGNED) = 1 AND cycle_phase = sqlc.arg(phase)))
ORDER BY sort_order, id;

-- name: ListTaskCompletionIDsOn :many
-- $user->taskCompletions()->whereDate('completion_date', $date)->pluck('task_template_id').
SELECT task_template_id FROM `user_task_completions`
WHERE user_id = ? AND completion_date = ?
ORDER BY id;

-- name: ListAffirmationsForPhase :many
-- Affirmation::active()->forPhase($phase)->orderBy('sort_order')->orderBy('id').
SELECT id, `text`
FROM `affirmations`
WHERE is_active = 1
  AND (cycle_phase IS NULL OR (CAST(sqlc.arg(has_phase) AS SIGNED) = 1 AND cycle_phase = sqlc.arg(phase)))
ORDER BY sort_order, id;

-- name: ListActiveChallenges :many
-- DailyChallengeService::challengeFor(): Challenge::active()->orderBy('sort_order')->orderBy('id').
SELECT id, title, description, cycle_day_from, cycle_day_to, category
FROM `challenges`
WHERE is_active = 1
ORDER BY sort_order, id;

-- name: ListChallengeIDsCompletedBetween :many
-- withoutRecentlyCompleted(): completions with from <= completion_date < before.
SELECT challenge_id FROM `user_challenge_completions`
WHERE user_id = sqlc.arg(user_id)
  AND completion_date >= sqlc.arg(from_date)
  AND completion_date < sqlc.arg(before_date);

-- name: ChallengeCompletedOn :one
-- DailyChallengeService::isCompleted().
SELECT EXISTS (
    SELECT 1 FROM `user_challenge_completions`
    WHERE user_id = ? AND challenge_id = ? AND completion_date = ?
) AS completed;

-- name: GetDoctorReminder :one
-- $user->reminders()->active()->ofType('doctor')->relevantOn($date)->orderBy('scheduled_at')->first().
SELECT id, title, subtitle, notes, scheduled_at, recurrence, recurrence_time, meta
FROM `reminders`
WHERE user_id = sqlc.arg(user_id) AND is_active = 1 AND `type` = 'doctor'
  AND ((recurrence <> 'none'
        AND (starts_on IS NULL OR starts_on <= sqlc.arg(day))
        AND (ends_on IS NULL OR ends_on >= sqlc.arg(day)))
    OR (recurrence = 'none' AND scheduled_at IS NOT NULL AND scheduled_at >= sqlc.arg(day_start)))
ORDER BY scheduled_at, id
LIMIT 1;

-- name: GetMedicationReminder :one
-- ...->ofType('medication')->relevantOn($date)->orderByRaw('COALESCE(recurrence_time, TIME(scheduled_at))')->first().
SELECT id, title, subtitle, notes, scheduled_at, recurrence, recurrence_time, meta
FROM `reminders`
WHERE user_id = sqlc.arg(user_id) AND is_active = 1 AND `type` = 'medication'
  AND ((recurrence <> 'none'
        AND (starts_on IS NULL OR starts_on <= sqlc.arg(day))
        AND (ends_on IS NULL OR ends_on >= sqlc.arg(day)))
    OR (recurrence = 'none' AND scheduled_at IS NOT NULL AND scheduled_at >= sqlc.arg(day_start)))
ORDER BY COALESCE(recurrence_time, TIME(scheduled_at)), id
LIMIT 1;

-- name: ListHomeArticles :many
-- Article::published()->forPhase($candidates)->limit(6): untagged articles, or tagged with any
-- candidate (orWhereJsonContains per candidate = JSON_OVERLAPS with the candidate list);
-- ordered `cycle_phases is null`, sort_order.
SELECT id, slug, title, excerpt, cycle_phases, category, read_time_minutes, image_url, image_path
FROM `articles`
WHERE is_published = 1
  AND (cycle_phases IS NULL
    OR (CAST(sqlc.arg(has_phases) AS SIGNED) = 1 AND JSON_OVERLAPS(cycle_phases, sqlc.arg(phases))))
ORDER BY (cycle_phases IS NULL), sort_order, id
LIMIT 6;

-- name: GetTaskTemplateID :one
-- Route-model binding {task}: TaskTemplate::where('id', $value)->firstOrFail().
SELECT id FROM `task_templates` WHERE id = ? LIMIT 1;

-- name: GetTaskCompletionOn :one
SELECT id FROM `user_task_completions`
WHERE user_id = ? AND task_template_id = ? AND completion_date = ?
ORDER BY id
LIMIT 1;

-- name: DeleteTaskCompletion :exec
DELETE FROM `user_task_completions` WHERE id = ?;

-- name: InsertTaskCompletion :exec
INSERT INTO `user_task_completions` (user_id, task_template_id, completion_date, completed_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetChallengeID :one
-- Route-model binding {challenge}.
SELECT id FROM `challenges` WHERE id = ? LIMIT 1;

-- name: GetChallengeCompletionOn :one
SELECT id FROM `user_challenge_completions`
WHERE user_id = ? AND challenge_id = ? AND completion_date = ?
ORDER BY id
LIMIT 1;

-- name: DeleteChallengeCompletion :exec
DELETE FROM `user_challenge_completions` WHERE id = ?;

-- name: InsertChallengeCompletion :exec
INSERT INTO `user_challenge_completions` (user_id, challenge_id, completion_date, completed_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: CountUserNotifications :one
SELECT COUNT(*) FROM `user_notifications` WHERE user_id = ?;

-- name: ListUserNotifications :many
-- $user->appNotifications()->orderByDesc('created_at')->paginate($perPage).
SELECT id, `type`, title, body, action_url, read_at, created_at
FROM `user_notifications`
WHERE user_id = ?
ORDER BY created_at DESC, id
LIMIT ? OFFSET ?;

-- name: GetNotificationOwner :one
-- Route-model binding {notification}.
SELECT id, user_id, read_at FROM `user_notifications` WHERE id = ? LIMIT 1;

-- name: MarkNotificationRead :exec
-- $notification->update(['read_at' => now()]) (touches updated_at).
UPDATE `user_notifications` SET read_at = ?, updated_at = ? WHERE id = ?;

-- name: MarkAllNotificationsRead :execrows
-- $user->appNotifications()->unread()->update(['read_at' => now()]) (Eloquent adds updated_at).
UPDATE `user_notifications` SET read_at = ?, updated_at = ?
WHERE user_id = ? AND read_at IS NULL;

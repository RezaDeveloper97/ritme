-- Challenges v2 (bloom B-N9-07, D-87): program settings, per-day tasks, enrollments, completions, reward grants.
-- Every user-side query is scoped by user_id; the admin queries are the editor and the stats.

-- ---------------------------------------------------------------------------
-- Programs (admin editor + user list)

-- name: GetProgram :one
SELECT * FROM `challenge_programs` WHERE challenge_id = ? LIMIT 1;

-- name: ListPrograms :many
SELECT * FROM `challenge_programs` WHERE challenge_id IN (sqlc.slice(ids));

-- name: UpsertProgram :exec
INSERT INTO `challenge_programs` (challenge_id, duration_days, start_rule, starts_on, reminder_time, eligibility,
                                  reward_badge_code, reward_badge_title, reward_plus_days, completion_min_days,
                                  created_at, updated_at)
VALUES (sqlc.arg(challenge_id), sqlc.arg(duration_days), sqlc.arg(start_rule), sqlc.narg(starts_on),
        sqlc.narg(reminder_time), sqlc.narg(eligibility), sqlc.narg(reward_badge_code),
        sqlc.narg(reward_badge_title), sqlc.arg(reward_plus_days), sqlc.narg(completion_min_days),
        sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE duration_days = VALUES(duration_days), start_rule = VALUES(start_rule),
  starts_on = VALUES(starts_on), reminder_time = VALUES(reminder_time), eligibility = VALUES(eligibility),
  reward_badge_code = VALUES(reward_badge_code), reward_badge_title = VALUES(reward_badge_title),
  reward_plus_days = VALUES(reward_plus_days), completion_min_days = VALUES(completion_min_days),
  updated_at = VALUES(updated_at);

-- name: DeleteProgram :exec
DELETE FROM `challenge_programs` WHERE challenge_id = ?;

-- ---------------------------------------------------------------------------
-- Tasks

-- name: ListTasks :many
SELECT * FROM `challenge_tasks` WHERE challenge_id = ? ORDER BY sort_order, id;

-- name: GetTask :one
SELECT * FROM `challenge_tasks` WHERE id = ? AND challenge_id = ? LIMIT 1;

-- name: InsertTask :exec
INSERT INTO `challenge_tasks` (challenge_id, day_from, day_to, title, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateTask :exec
UPDATE `challenge_tasks` SET day_from = ?, day_to = ?, title = ?, sort_order = ?, updated_at = ?
WHERE id = ? AND challenge_id = ?;

-- name: DeleteTasksExcept :exec
DELETE FROM `challenge_tasks` WHERE challenge_id = sqlc.arg(challenge_id) AND id NOT IN (sqlc.slice(keep_ids));

-- name: DeleteAllTasks :exec
DELETE FROM `challenge_tasks` WHERE challenge_id = ?;

-- ---------------------------------------------------------------------------
-- User side

-- name: ListActivePrograms :many
-- Active v2 challenges with their program settings, for the user list.
SELECT c.id, c.title, c.description, c.category, c.sort_order,
       p.duration_days, p.start_rule, p.starts_on, p.reminder_time, p.eligibility, p.reward_badge_code,
       p.reward_badge_title, p.reward_plus_days, p.completion_min_days
FROM `challenges` c
JOIN `challenge_programs` p ON p.challenge_id = c.id
WHERE c.is_active = 1
ORDER BY c.sort_order, c.id;

-- name: GetActiveProgram :one
SELECT c.id, c.title, c.description, c.category, c.sort_order,
       p.duration_days, p.start_rule, p.starts_on, p.reminder_time, p.eligibility, p.reward_badge_code,
       p.reward_badge_title, p.reward_plus_days, p.completion_min_days
FROM `challenges` c
JOIN `challenge_programs` p ON p.challenge_id = c.id
WHERE c.id = ? AND c.is_active = 1
LIMIT 1;

-- name: ListUserEnrollments :many
SELECT * FROM `challenge_enrollments` WHERE user_id = ?;

-- name: GetUserEnrollment :one
SELECT * FROM `challenge_enrollments` WHERE user_id = ? AND challenge_id = ? LIMIT 1;

-- name: LockUserEnrollment :one
-- Serialises the completion check of one enrollment (the reward is granted under this lock).
SELECT * FROM `challenge_enrollments` WHERE user_id = ? AND challenge_id = ? LIMIT 1 FOR UPDATE;

-- name: InsertEnrollment :execlastid
INSERT INTO `challenge_enrollments` (user_id, challenge_id, started_on, status, created_at, updated_at)
VALUES (?, ?, ?, 'active', ?, ?);

-- name: CompleteEnrollment :exec
UPDATE `challenge_enrollments` SET status = 'completed', completed_at = ?, updated_at = ?
WHERE id = ? AND status = 'active';

-- name: InsertTaskCompletion :execrows
INSERT IGNORE INTO `challenge_task_completions` (enrollment_id, task_id, day_number, completed_at)
VALUES (?, ?, ?, ?);

-- name: ListEnrollmentCompletions :many
SELECT task_id, day_number FROM `challenge_task_completions` WHERE enrollment_id = ?;

-- name: InsertRewardGrant :execlastid
INSERT INTO `challenge_reward_grants` (user_id, challenge_id, enrollment_id, badge_code, badge_title, plus_days,
                                       subscription_id, granted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetGrantSubscription :exec
UPDATE `challenge_reward_grants` SET subscription_id = ? WHERE id = ?;

-- name: ListUserRewards :many
SELECT g.challenge_id, g.badge_code, g.badge_title, g.plus_days, g.granted_at, c.title AS challenge_title
FROM `challenge_reward_grants` g
JOIN `challenges` c ON c.id = g.challenge_id
WHERE g.user_id = ?
ORDER BY g.granted_at DESC, g.id DESC;

-- ---------------------------------------------------------------------------
-- Admin stats

-- name: ProgramStats :many
-- Participants / completions per v2 challenge (the list columns).
SELECT e.challenge_id, COUNT(*) AS participants,
       CAST(COALESCE(SUM(e.status = 'completed'), 0) AS SIGNED) AS completed
FROM `challenge_enrollments` e
WHERE e.challenge_id IN (sqlc.slice(ids))
GROUP BY e.challenge_id;

-- name: ProgramRewardStats :one
SELECT COUNT(*) AS rewards, CAST(COALESCE(SUM(plus_days), 0) AS SIGNED) AS plus_days
FROM `challenge_reward_grants` WHERE challenge_id = ?;

-- name: ProgramDayStats :many
-- Distinct users who completed anything on each day of the run.
SELECT tc.day_number, COUNT(DISTINCT e.user_id) AS users
FROM `challenge_task_completions` tc
JOIN `challenge_enrollments` e ON e.id = tc.enrollment_id
WHERE e.challenge_id = ?
GROUP BY tc.day_number
ORDER BY tc.day_number;

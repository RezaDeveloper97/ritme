-- Per-user activity days (migration 00105, T-M10-25; SPEC v6 §5.6, DECISIONS.md A-06). One row per user and Tehran
-- day; writers mark a session or a data log, readers count DAU / MAU / logging users. The existing raw-SQL
-- aggregation in internal/metrics stays as it is.

-- name: MarkUserSessionDay :exec
INSERT INTO `user_activity_days` (`user_id`, `day`, `session`) VALUES (sqlc.arg(user_id), sqlc.arg(day), 1)
ON DUPLICATE KEY UPDATE `session` = 1;

-- name: MarkUserLoggedDay :exec
INSERT INTO `user_activity_days` (`user_id`, `day`, `logged`) VALUES (sqlc.arg(user_id), sqlc.arg(day), 1)
ON DUPLICATE KEY UPDATE `logged` = 1;

-- name: CountActiveUsersByDay :many
-- DAU and logging users per day in [from_day, to_day].
SELECT `day`,
       CAST(SUM(`session`) AS SIGNED) AS session_users,
       CAST(SUM(`logged`) AS SIGNED) AS logged_users,
       COUNT(*) AS active_users
FROM `user_activity_days`
WHERE `day` BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day)
GROUP BY `day`
ORDER BY `day`;

-- name: CountDistinctSessionUsers :one
-- MAU: distinct users with at least one session in [from_day, to_day].
SELECT COUNT(DISTINCT `user_id`) FROM `user_activity_days`
WHERE `day` BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day) AND `session` = 1;

-- name: CountDistinctActiveUsers :one
-- Distinct users with a session or a log in [from_day, to_day] (A-06 "active user", sessions and logs part).
SELECT COUNT(DISTINCT `user_id`) FROM `user_activity_days`
WHERE `day` BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day);

-- name: ListUserActivityDays :many
-- One user's days in [from_day, to_day] (the activity projection, T-M10-32).
SELECT `day`, `session`, `logged` FROM `user_activity_days`
WHERE `user_id` = sqlc.arg(user_id) AND `day` BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day)
ORDER BY `day`;

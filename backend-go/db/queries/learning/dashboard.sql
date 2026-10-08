-- Ins_Dashboard (bloom B-N8-06): owner-scoped aggregates of one instructor. Student counts follow the H1 rule of
-- B-N8-01b — a grant counts as an active student only once she opened one of the instructor's courses (seen_at).

-- name: DashboardStudentCounts :one
SELECT
  CAST(COUNT(DISTINCT CASE WHEN a.status = 'active' AND a.seen_at IS NOT NULL
    AND (a.expires_at IS NULL OR a.expires_at > sqlc.arg(now)) THEN a.phone END) AS SIGNED) AS active_students,
  CAST(COUNT(DISTINCT CASE WHEN a.created_at >= sqlc.arg(since) THEN a.phone END) AS SIGNED) AS new_students
FROM `learning_grants` a
WHERE a.instructor_id = sqlc.arg(instructor_id) AND a.status <> 'revoked';

-- name: DashboardWatchSeconds :one
-- Approximation: the playback position of every progress row touched since `since` (no per-session watch log yet).
SELECT CAST(COALESCE(SUM(p.position_seconds), 0) AS SIGNED) AS watch_seconds
FROM `learning_progress` p
JOIN `learning_courses` c ON c.id = p.course_id
WHERE c.instructor_id = sqlc.arg(instructor_id) AND p.last_seen_at >= sqlc.arg(since);

-- name: DashboardRecentLessons :many
SELECT l.id, l.title, l.published_at, c.id AS course_id, c.title AS course_title
FROM `learning_lessons` l
JOIN `learning_courses` c ON c.id = l.course_id
WHERE c.instructor_id = sqlc.arg(instructor_id) AND l.status = 'published' AND l.published_at >= sqlc.arg(since)
ORDER BY l.published_at DESC, l.id DESC
LIMIT 5;

-- name: DashboardRecentGrants :many
-- New grants per target and day (counts only: no phone, name or registration state).
SELECT a.group_id, a.course_id, g.name AS group_name, c.title AS course_title,
  CAST(COUNT(*) AS SIGNED) AS added, CAST(MAX(a.created_at) AS DATETIME) AS last_at
FROM `learning_grants` a
LEFT JOIN `learning_groups` g ON g.id = a.group_id
LEFT JOIN `learning_courses` c ON c.id = a.course_id
WHERE a.instructor_id = sqlc.arg(instructor_id) AND a.status <> 'revoked' AND a.created_at >= sqlc.arg(since)
GROUP BY a.group_id, a.course_id, g.name, c.title, DATE(a.created_at)
ORDER BY last_at DESC
LIMIT 5;

-- name: DashboardLessonKinds :many
-- The lesson kinds of each course (Ins_Content kind filter).
SELECT DISTINCT l.course_id, l.kind
FROM `learning_lessons` l
JOIN `learning_courses` c ON c.id = l.course_id
WHERE c.instructor_id = sqlc.arg(instructor_id)
ORDER BY l.course_id, l.kind;

-- Admin moderation of courses «مدرسین و دوره‌ها» (bloom B-N8-08; internal/admin/learning, admin-api.md §20).
-- Read-mostly over the B-N8-01 tables (00046) plus the review state and moderation log of 00054. Filters are LIKE
-- patterns ('%' = all) like the other admin lists. No health data: courses are instructor content, progress is only
-- aggregated (percent per student × course), phone numbers are never selected here.

-- ───────────── instructors ─────────────

-- name: CountLearningInstructors :one
SELECT COUNT(*) FROM `learning_instructors` i
JOIN `users` u ON u.id = i.user_id
WHERE i.status LIKE sqlc.arg(status)
  AND (sqlc.arg(q) = '' OR i.display_name LIKE CAST(sqlc.arg(q_like) AS CHAR) OR IFNULL(u.name, '') LIKE CAST(sqlc.arg(q_like) AS CHAR)
       OR IFNULL(u.mobile, '') LIKE CAST(sqlc.arg(q_like) AS CHAR));

-- name: CountLearningInstructorsByStatus :many
SELECT `status`, COUNT(*) AS total FROM `learning_instructors` GROUP BY `status`;

-- name: ListLearningInstructors :many
-- Pending applications first (what needs an answer), then newest.
SELECT i.id, i.user_id, i.display_name, i.title, i.bio, i.status, i.approved_at, i.approved_by, i.revoked_at,
  i.created_at, i.updated_at, u.name AS user_name, u.mobile AS user_mobile, ad.name AS approved_by_name,
  (SELECT COUNT(*) FROM `learning_courses` c WHERE c.instructor_id = i.id) AS courses_count,
  (SELECT COUNT(*) FROM `learning_courses` c WHERE c.instructor_id = i.id AND c.status = 'published') AS published_courses,
  (SELECT COUNT(DISTINCT a.user_id) FROM `learning_grants` a
    WHERE a.instructor_id = i.id AND a.status = 'active' AND a.user_id IS NOT NULL
      AND (a.expires_at IS NULL OR a.expires_at > sqlc.arg(now))) AS students_count
FROM `learning_instructors` i
JOIN `users` u ON u.id = i.user_id
LEFT JOIN `admins` ad ON ad.id = i.approved_by
WHERE i.status LIKE sqlc.arg(status)
  AND (sqlc.arg(q) = '' OR i.display_name LIKE CAST(sqlc.arg(q_like) AS CHAR) OR IFNULL(u.name, '') LIKE CAST(sqlc.arg(q_like) AS CHAR)
       OR IFNULL(u.mobile, '') LIKE CAST(sqlc.arg(q_like) AS CHAR))
ORDER BY (i.status = 'pending') DESC, i.created_at DESC, i.id DESC
LIMIT ? OFFSET ?;

-- name: GetLearningInstructorAdmin :one
SELECT i.id, i.user_id, i.display_name, i.title, i.bio, i.status, i.approved_at, i.approved_by, i.revoked_at,
  i.created_at, i.updated_at, u.name AS user_name, u.mobile AS user_mobile, ad.name AS approved_by_name,
  (SELECT COUNT(*) FROM `learning_courses` c WHERE c.instructor_id = i.id) AS courses_count,
  (SELECT COUNT(*) FROM `learning_courses` c WHERE c.instructor_id = i.id AND c.status = 'published') AS published_courses,
  (SELECT COUNT(DISTINCT a.user_id) FROM `learning_grants` a
    WHERE a.instructor_id = i.id AND a.status = 'active' AND a.user_id IS NOT NULL
      AND (a.expires_at IS NULL OR a.expires_at > sqlc.arg(now))) AS students_count
FROM `learning_instructors` i
JOIN `users` u ON u.id = i.user_id
LEFT JOIN `admins` ad ON ad.id = i.approved_by
WHERE i.id = sqlc.arg(id) LIMIT 1;

-- name: GetLearningInstructorForUpdate :one
SELECT id, status FROM `learning_instructors` WHERE id = sqlc.arg(id) LIMIT 1 FOR UPDATE;

-- name: ApproveLearningInstructor :exec
UPDATE `learning_instructors`
SET status = 'approved', approved_at = sqlc.arg(now), approved_by = sqlc.arg(admin_id), revoked_at = NULL,
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: RevokeLearningInstructor :exec
UPDATE `learning_instructors` SET status = 'revoked', revoked_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- ───────────── courses ─────────────

-- name: CountLearningCourses :one
SELECT COUNT(*) FROM `learning_courses` c
WHERE c.status LIKE sqlc.arg(status)
  AND (sqlc.arg(instructor_id) = 0 OR c.instructor_id = sqlc.arg(instructor_id))
  AND (sqlc.arg(q) = '' OR c.title LIKE CAST(sqlc.arg(q_like) AS CHAR));

-- name: CountLearningCoursesByStatus :many
SELECT `status`, COUNT(*) AS total FROM `learning_courses` GROUP BY `status`;

-- name: ListLearningCourses :many
SELECT c.id, c.instructor_id, c.kind, c.title, c.status, c.published_at, c.created_at, c.updated_at,
  i.display_name AS instructor_name, i.status AS instructor_status,
  (SELECT COUNT(*) FROM `learning_chapters` ch WHERE ch.course_id = c.id) AS chapters_count,
  (SELECT COUNT(*) FROM `learning_lessons` l WHERE l.course_id = c.id) AS lessons_count,
  (SELECT COUNT(*) FROM `learning_lessons` l WHERE l.course_id = c.id AND l.status = 'published') AS published_lessons,
  (SELECT COUNT(*) FROM `learning_lessons` l WHERE l.course_id = c.id AND l.status = 'published'
     AND (l.review_status IN ('pending', 'flagged') OR l.reviewed_at IS NULL OR l.updated_at > l.reviewed_at)) AS review_open,
  (SELECT COUNT(*) FROM `learning_lessons` l WHERE l.course_id = c.id AND l.review_status = 'flagged') AS flagged_count
FROM `learning_courses` c
JOIN `learning_instructors` i ON i.id = c.instructor_id
WHERE c.status LIKE sqlc.arg(status)
  AND (sqlc.arg(instructor_id) = 0 OR c.instructor_id = sqlc.arg(instructor_id))
  AND (sqlc.arg(q) = '' OR c.title LIKE CAST(sqlc.arg(q_like) AS CHAR))
ORDER BY c.created_at DESC, c.id DESC
LIMIT ? OFFSET ?;

-- name: GetLearningCourse :one
SELECT c.id, c.instructor_id, c.kind, c.title, c.description, c.status, c.published_at, c.created_at, c.updated_at,
  i.display_name AS instructor_name, i.title AS instructor_title, i.status AS instructor_status
FROM `learning_courses` c
JOIN `learning_instructors` i ON i.id = c.instructor_id
WHERE c.id = sqlc.arg(id) LIMIT 1;

-- name: ListLearningCourseChapters :many
SELECT id, title, sort_order, unlock_at FROM `learning_chapters` WHERE course_id = sqlc.arg(course_id)
ORDER BY sort_order, id;

-- name: ListLearningCourseLessons :many
SELECT l.id, l.chapter_id, l.kind, l.title, l.description, l.duration_seconds, l.page_count, l.size_bytes,
  l.media_status, l.status, l.published_at, l.review_status, l.reviewed_at, l.reviewed_by, l.review_note,
  l.sort_order, l.created_at, l.updated_at, ad.name AS reviewed_by_name
FROM `learning_lessons` l
LEFT JOIN `admins` ad ON ad.id = l.reviewed_by
WHERE l.course_id = sqlc.arg(course_id)
ORDER BY l.sort_order, l.id;

-- name: CountLearningCourseGrants :many
-- A course's grants by stored status (direct or through a group that opens it); expired = active past expires_at.
SELECT
  CASE WHEN a.status = 'active' AND a.expires_at IS NOT NULL AND a.expires_at <= sqlc.arg(now) THEN 'expired'
       ELSE a.status END AS state,
  COUNT(*) AS total
FROM `learning_grants` a
WHERE a.course_id = sqlc.narg(direct_course_id)
   OR a.group_id IN (SELECT gc.group_id FROM `learning_group_courses` gc WHERE gc.course_id = sqlc.arg(course_id))
GROUP BY state;

-- ───────────── usage aggregates (completion) ─────────────

-- name: ListLearningAccessPairs :many
-- (course, student) pairs with a running grant for the given courses: the denominator of «completion».
SELECT DISTINCT c.id AS course_id, a.user_id
FROM `learning_grants` a
JOIN `learning_courses` c ON c.instructor_id = a.instructor_id AND (
  c.id = a.course_id
  OR c.id IN (SELECT gc.course_id FROM `learning_group_courses` gc WHERE gc.group_id = a.group_id)
)
WHERE a.status = 'active' AND a.user_id IS NOT NULL AND (a.expires_at IS NULL OR a.expires_at > sqlc.arg(now))
  AND c.id IN (sqlc.slice(course_ids));

-- name: ListLearningAllAccessPairs :many
SELECT DISTINCT c.id AS course_id, a.user_id
FROM `learning_grants` a
JOIN `learning_courses` c ON c.instructor_id = a.instructor_id AND (
  c.id = a.course_id
  OR c.id IN (SELECT gc.course_id FROM `learning_group_courses` gc WHERE gc.group_id = a.group_id)
)
WHERE a.status = 'active' AND a.user_id IS NOT NULL AND (a.expires_at IS NULL OR a.expires_at > sqlc.arg(now));

-- name: SumLearningProgress :many
-- Per (course, student): the summed effective percent over published lessons (a completed lesson counts 100).
SELECT p.course_id, p.user_id,
  CAST(SUM(CASE WHEN p.completed_at IS NOT NULL THEN 100 ELSE LEAST(p.percent, 100) END) AS SIGNED) AS percent_sum
FROM `learning_progress` p
JOIN `learning_lessons` l ON l.id = p.lesson_id AND l.status = 'published'
WHERE p.course_id IN (sqlc.slice(course_ids))
GROUP BY p.course_id, p.user_id;

-- name: SumAllLearningProgress :many
SELECT p.course_id, p.user_id,
  CAST(SUM(CASE WHEN p.completed_at IS NOT NULL THEN 100 ELSE LEAST(p.percent, 100) END) AS SIGNED) AS percent_sum
FROM `learning_progress` p
JOIN `learning_lessons` l ON l.id = p.lesson_id AND l.status = 'published'
GROUP BY p.course_id, p.user_id;

-- name: CountLearningPublishedLessons :many
SELECT course_id, COUNT(*) AS total FROM `learning_lessons` WHERE status = 'published' GROUP BY course_id;

-- name: CountLearningGrantsByState :many
SELECT
  CASE WHEN status = 'active' AND expires_at IS NOT NULL AND expires_at <= sqlc.arg(now) THEN 'expired'
       ELSE status END AS state,
  COUNT(*) AS total
FROM `learning_grants`
GROUP BY state;

-- name: CountLearningActiveStudents :one
-- Distinct students who opened a lesson since `since` (activity, not access).
SELECT COUNT(DISTINCT user_id) FROM `learning_progress` WHERE last_seen_at >= sqlc.arg(since);

-- name: CountLearningCompletedLessons :one
SELECT COUNT(*) FROM `learning_progress` WHERE completed_at IS NOT NULL;

-- name: CountLearningGrantsSince :one
SELECT COUNT(*) FROM `learning_grants` WHERE created_at >= sqlc.arg(since);

-- name: CountLearningLessonsByState :many
-- Review queue counts over published lessons + flagged ones (an unpublished lesson stays flagged).
SELECT
  CASE WHEN review_status = 'flagged' THEN 'flagged'
       WHEN review_status = 'pending' THEN 'pending'
       WHEN reviewed_at IS NULL OR updated_at > reviewed_at THEN 'changed'
       ELSE 'approved' END AS state,
  COUNT(*) AS total
FROM `learning_lessons`
WHERE status = 'published' OR review_status = 'flagged'
GROUP BY state;

-- name: CountLearningLessons :one
SELECT COUNT(*) AS total, CAST(IFNULL(SUM(status = 'published'), 0) AS SIGNED) AS published FROM `learning_lessons`;

-- ───────────── review queue ─────────────

-- name: CountLearningReviewQueue :one
SELECT COUNT(*) FROM `learning_lessons` l
WHERE (l.status = 'published' OR l.review_status = 'flagged')
  AND (
    (sqlc.arg(want_flagged) AND l.review_status = 'flagged')
    OR (sqlc.arg(want_pending) AND l.review_status = 'pending')
    OR (sqlc.arg(want_changed) AND l.review_status = 'approved' AND (l.reviewed_at IS NULL OR l.updated_at > l.reviewed_at))
    OR (sqlc.arg(want_approved) AND l.review_status = 'approved' AND l.reviewed_at IS NOT NULL
        AND (l.updated_at IS NULL OR l.updated_at <= l.reviewed_at))
  );

-- name: ListLearningReviewQueue :many
-- Published lessons (and flagged ones, also after a takedown) in the given review states. Open states come oldest-change first (a queue), the rest newest first.
SELECT l.id, l.course_id, l.chapter_id, l.kind, l.title, l.description, l.duration_seconds, l.page_count,
  l.size_bytes, l.media_status, l.status, l.published_at, l.review_status, l.reviewed_at, l.reviewed_by,
  l.review_note, l.created_at, l.updated_at,
  c.title AS course_title, c.kind AS course_kind, c.status AS course_status,
  i.id AS instructor_id, i.display_name AS instructor_name, i.status AS instructor_status,
  ad.name AS reviewed_by_name
FROM `learning_lessons` l
JOIN `learning_courses` c ON c.id = l.course_id
JOIN `learning_instructors` i ON i.id = c.instructor_id
LEFT JOIN `admins` ad ON ad.id = l.reviewed_by
WHERE (l.status = 'published' OR l.review_status = 'flagged')
  AND (
    (sqlc.arg(want_flagged) AND l.review_status = 'flagged')
    OR (sqlc.arg(want_pending) AND l.review_status = 'pending')
    OR (sqlc.arg(want_changed) AND l.review_status = 'approved' AND (l.reviewed_at IS NULL OR l.updated_at > l.reviewed_at))
    OR (sqlc.arg(want_approved) AND l.review_status = 'approved' AND l.reviewed_at IS NOT NULL
        AND (l.updated_at IS NULL OR l.updated_at <= l.reviewed_at))
  )
ORDER BY
  CASE WHEN sqlc.arg(oldest_first) THEN l.updated_at END ASC,
  l.updated_at DESC,
  l.id
LIMIT ? OFFSET ?;

-- name: GetLearningReviewLesson :one
SELECT l.id, l.course_id, l.chapter_id, l.kind, l.title, l.description, l.duration_seconds, l.page_count,
  l.size_bytes, l.media_status, l.status, l.published_at, l.review_status, l.reviewed_at, l.reviewed_by,
  l.review_note, l.created_at, l.updated_at,
  c.title AS course_title, c.kind AS course_kind, c.status AS course_status,
  i.id AS instructor_id, i.display_name AS instructor_name, i.status AS instructor_status,
  ad.name AS reviewed_by_name
FROM `learning_lessons` l
JOIN `learning_courses` c ON c.id = l.course_id
JOIN `learning_instructors` i ON i.id = c.instructor_id
LEFT JOIN `admins` ad ON ad.id = l.reviewed_by
WHERE l.id = sqlc.arg(id) LIMIT 1;

-- name: GetLearningLessonForUpdate :one
SELECT l.id, l.course_id, l.status, c.instructor_id FROM `learning_lessons` l
JOIN `learning_courses` c ON c.id = l.course_id
WHERE l.id = sqlc.arg(id) LIMIT 1 FOR UPDATE;

-- name: ReviewLearningLesson :exec
-- approve / flag: the review fields only — updated_at stays, so «changed since review» keeps meaning the instructor.
UPDATE `learning_lessons`
SET review_status = sqlc.arg(review_status), reviewed_at = sqlc.arg(now), reviewed_by = sqlc.arg(admin_id),
    review_note = sqlc.narg(note)
WHERE id = sqlc.arg(id);

-- name: UnpublishLearningLesson :exec
-- A takedown: back to draft (students stop seeing it at once) and flagged with the admin's reason.
UPDATE `learning_lessons`
SET status = 'draft', review_status = 'flagged', reviewed_at = sqlc.arg(now), reviewed_by = sqlc.arg(admin_id),
    review_note = sqlc.narg(note), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- ───────────── moderation log ─────────────

-- name: InsertLearningModeration :exec
INSERT INTO `learning_moderation_log` (admin_id, action, target_type, target_id, instructor_id, note, created_at, updated_at)
VALUES (sqlc.arg(admin_id), sqlc.arg(action), sqlc.arg(target_type), sqlc.arg(target_id), sqlc.narg(instructor_id),
        sqlc.narg(note), sqlc.arg(now), sqlc.arg(now));

-- name: ListLearningModeration :many
SELECT m.id, m.admin_id, m.action, m.target_type, m.target_id, m.instructor_id, m.note, m.created_at,
  ad.name AS admin_name
FROM `learning_moderation_log` m
LEFT JOIN `admins` ad ON ad.id = m.admin_id
WHERE (m.target_type = 'instructor' AND m.target_id = sqlc.arg(target_id)) OR m.instructor_id = sqlc.narg(instructor_id)
ORDER BY m.created_at DESC, m.id DESC
LIMIT 20;

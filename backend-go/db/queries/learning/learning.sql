-- Courses / instructors / phone-based access (bloom B-N8-01; internal/learning).
-- Instructor queries are scoped by instructor_id, student queries by user_id, in the query itself (IDOR).

-- ───────────── instructors ─────────────

-- name: GetInstructorByUser :one
SELECT * FROM `learning_instructors` WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- name: GetInstructor :one
SELECT * FROM `learning_instructors` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: InsertInstructor :execlastid
INSERT INTO `learning_instructors` (user_id, display_name, title, bio, status, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(display_name), sqlc.narg(title), sqlc.narg(bio), 'pending', sqlc.arg(now), sqlc.arg(now));

-- name: UpdateInstructorProfile :exec
UPDATE `learning_instructors`
SET display_name = sqlc.arg(display_name), title = sqlc.narg(title), bio = sqlc.narg(bio), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetInstructorStatus :execrows
-- Admin approval / revocation (B-N8-08) and re-application after a revocation.
UPDATE `learning_instructors`
SET status = sqlc.arg(status), approved_at = sqlc.narg(approved_at), approved_by = sqlc.narg(approved_by),
    revoked_at = sqlc.narg(revoked_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- ───────────── courses (instructor) ─────────────

-- name: ListInstructorCourses :many
SELECT c.*,
  CAST((SELECT COUNT(*) FROM `learning_lessons` l WHERE l.course_id = c.id) AS SIGNED) AS lessons_count,
  CAST((SELECT COUNT(*) FROM `learning_lessons` l WHERE l.course_id = c.id AND l.status = 'published') AS SIGNED) AS published_lessons,
  CAST((SELECT COALESCE(SUM(l.duration_seconds), 0) FROM `learning_lessons` l WHERE l.course_id = c.id) AS SIGNED) AS duration_seconds
FROM `learning_courses` c
WHERE c.instructor_id = sqlc.arg(instructor_id)
ORDER BY c.sort_order, c.id;

-- name: GetInstructorCourse :one
SELECT * FROM `learning_courses` WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id) LIMIT 1;

-- name: CountInstructorCourses :one
SELECT COUNT(*) FROM `learning_courses` WHERE instructor_id = sqlc.arg(instructor_id);

-- name: CountInstructorCoursesIn :one
SELECT COUNT(*) FROM `learning_courses` WHERE instructor_id = sqlc.arg(instructor_id) AND id IN (sqlc.slice(ids));

-- name: InsertCourse :execlastid
INSERT INTO `learning_courses` (instructor_id, kind, title, description, status, published_at, sort_order, created_at, updated_at)
VALUES (sqlc.arg(instructor_id), sqlc.arg(kind), sqlc.arg(title), sqlc.narg(description), sqlc.arg(status),
        sqlc.narg(published_at), 0, sqlc.arg(now), sqlc.arg(now));

-- name: UpdateCourse :execrows
UPDATE `learning_courses`
SET title = sqlc.arg(title), description = sqlc.narg(description), status = sqlc.arg(status),
    published_at = sqlc.narg(published_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id);

-- name: DeleteCourse :execrows
DELETE FROM `learning_courses` WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id);

-- ───────────── chapters ─────────────

-- name: ListChapters :many
SELECT * FROM `learning_chapters` WHERE course_id = sqlc.arg(course_id) ORDER BY sort_order, id;

-- name: ListChaptersForCourses :many
SELECT * FROM `learning_chapters` WHERE course_id IN (sqlc.slice(course_ids)) ORDER BY course_id, sort_order, id;

-- name: GetChapter :one
SELECT * FROM `learning_chapters` WHERE id = sqlc.arg(id) AND course_id = sqlc.arg(course_id) LIMIT 1;

-- name: CountChapters :one
SELECT COUNT(*) FROM `learning_chapters` WHERE course_id = sqlc.arg(course_id);

-- name: MaxChapterSort :one
SELECT CAST(COALESCE(MAX(sort_order), 0) AS SIGNED) FROM `learning_chapters` WHERE course_id = sqlc.arg(course_id);

-- name: InsertChapter :execlastid
INSERT INTO `learning_chapters` (course_id, title, sort_order, unlock_at, created_at, updated_at)
VALUES (sqlc.arg(course_id), sqlc.arg(title), sqlc.arg(sort_order), sqlc.narg(unlock_at), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateChapter :execrows
UPDATE `learning_chapters`
SET title = sqlc.arg(title), sort_order = sqlc.arg(sort_order), unlock_at = sqlc.narg(unlock_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND course_id = sqlc.arg(course_id);

-- name: DeleteChapter :execrows
DELETE FROM `learning_chapters` WHERE id = sqlc.arg(id) AND course_id = sqlc.arg(course_id);

-- ───────────── lessons ─────────────

-- name: ListLessons :many
SELECT * FROM `learning_lessons` WHERE course_id = sqlc.arg(course_id) ORDER BY sort_order, id;

-- name: ListPublishedLessonsForCourses :many
SELECT * FROM `learning_lessons`
WHERE course_id IN (sqlc.slice(course_ids)) AND status = 'published'
ORDER BY course_id, sort_order, id;

-- name: GetLesson :one
SELECT * FROM `learning_lessons` WHERE id = sqlc.arg(id) AND course_id = sqlc.arg(course_id) LIMIT 1;

-- name: GetLessonByID :one
SELECT * FROM `learning_lessons` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: CountLessons :one
SELECT COUNT(*) FROM `learning_lessons` WHERE course_id = sqlc.arg(course_id);

-- name: MaxLessonSort :one
SELECT CAST(COALESCE(MAX(sort_order), 0) AS SIGNED) FROM `learning_lessons` WHERE course_id = sqlc.arg(course_id);

-- name: InsertLesson :execlastid
INSERT INTO `learning_lessons` (
  course_id, chapter_id, kind, title, description, duration_seconds, page_count, size_bytes, media_id, media_status,
  status, published_at, sort_order, created_at, updated_at
) VALUES (
  sqlc.arg(course_id), sqlc.narg(chapter_id), sqlc.arg(kind), sqlc.arg(title), sqlc.narg(description),
  sqlc.narg(duration_seconds), sqlc.narg(page_count), sqlc.narg(size_bytes), NULL, 'none', sqlc.arg(status),
  sqlc.narg(published_at), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now)
);

-- name: UpdateLesson :execrows
UPDATE `learning_lessons`
SET chapter_id = sqlc.narg(chapter_id), kind = sqlc.arg(kind), title = sqlc.arg(title), description = sqlc.narg(description),
    duration_seconds = sqlc.narg(duration_seconds), page_count = sqlc.narg(page_count), size_bytes = sqlc.narg(size_bytes),
    status = sqlc.arg(status), published_at = sqlc.narg(published_at), sort_order = sqlc.arg(sort_order),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND course_id = sqlc.arg(course_id);

-- name: DeleteLesson :execrows
DELETE FROM `learning_lessons` WHERE id = sqlc.arg(id) AND course_id = sqlc.arg(course_id);

-- ───────────── groups ─────────────

-- name: ListGroups :many
SELECT g.*,
  CAST((SELECT COUNT(*) FROM `learning_grants` a WHERE a.group_id = g.id AND a.status = 'active') AS SIGNED) AS active_members,
  CAST((SELECT COUNT(*) FROM `learning_grants` a WHERE a.group_id = g.id AND a.status = 'pending') AS SIGNED) AS pending_members
FROM `learning_groups` g
WHERE g.instructor_id = sqlc.arg(instructor_id)
ORDER BY g.id DESC;

-- name: GetGroup :one
SELECT * FROM `learning_groups` WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id) LIMIT 1;

-- name: CountGroups :one
SELECT COUNT(*) FROM `learning_groups` WHERE instructor_id = sqlc.arg(instructor_id);

-- name: InsertGroup :execlastid
INSERT INTO `learning_groups` (instructor_id, name, created_at, updated_at)
VALUES (sqlc.arg(instructor_id), sqlc.arg(name), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateGroup :execrows
UPDATE `learning_groups` SET name = sqlc.arg(name), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id);

-- name: DeleteGroup :execrows
DELETE FROM `learning_groups` WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id);

-- name: ListGroupCourses :many
SELECT gc.group_id, c.id AS course_id, c.kind, c.title, c.status
FROM `learning_group_courses` gc
JOIN `learning_courses` c ON c.id = gc.course_id
WHERE gc.group_id IN (sqlc.slice(group_ids))
ORDER BY gc.group_id, gc.id;

-- name: DeleteGroupCourses :exec
DELETE FROM `learning_group_courses` WHERE group_id = sqlc.arg(group_id);

-- name: InsertGroupCourse :exec
INSERT IGNORE INTO `learning_group_courses` (group_id, course_id, created_at, updated_at)
VALUES (sqlc.arg(group_id), sqlc.arg(course_id), sqlc.arg(now), sqlc.arg(now));

-- name: ListCourseGroups :many
-- The groups that open one course (Ins_Course «چه کسانی دسترسی دارند»), with their active member counts.
SELECT g.id, g.name,
  CAST((SELECT COUNT(*) FROM `learning_grants` a WHERE a.group_id = g.id AND a.status = 'active') AS SIGNED) AS active_members
FROM `learning_group_courses` gc
JOIN `learning_groups` g ON g.id = gc.group_id
WHERE gc.course_id = sqlc.arg(course_id) AND g.instructor_id = sqlc.arg(instructor_id)
ORDER BY g.id;

-- name: CountDirectCourseGrants :one
SELECT COUNT(*) FROM `learning_grants`
WHERE course_id = sqlc.arg(course_id) AND instructor_id = sqlc.arg(instructor_id) AND status = 'active';

-- ───────────── grants ─────────────

-- name: GetUserIDByMobile :one
SELECT id FROM `users` WHERE mobile = sqlc.arg(mobile) LIMIT 1;

-- name: GetUserNames :many
SELECT id, name FROM `users` WHERE id IN (sqlc.slice(ids));

-- name: FindGrantForTarget :one
-- The grant an instructor already gave this phone for the same group / course (re-adding renews it).
SELECT * FROM `learning_grants`
WHERE instructor_id = sqlc.arg(instructor_id) AND phone = sqlc.arg(phone)
  AND group_id <=> sqlc.narg(group_id) AND course_id <=> sqlc.narg(course_id)
ORDER BY id DESC
LIMIT 1;

-- name: InsertGrant :execlastid
INSERT INTO `learning_grants` (
  instructor_id, phone, user_id, group_id, course_id, duration, duration_days, until_date, status, activated_at,
  expires_at, revoked_at, created_at, updated_at
) VALUES (
  sqlc.arg(instructor_id), sqlc.arg(phone), sqlc.narg(user_id), sqlc.narg(group_id), sqlc.narg(course_id),
  sqlc.arg(duration), sqlc.narg(duration_days), sqlc.narg(until_date), sqlc.arg(status), sqlc.narg(activated_at),
  sqlc.narg(expires_at), NULL, sqlc.arg(now), sqlc.arg(now)
);

-- name: RenewGrant :exec
UPDATE `learning_grants`
SET user_id = sqlc.narg(user_id), duration = sqlc.arg(duration), duration_days = sqlc.narg(duration_days),
    until_date = sqlc.narg(until_date), status = sqlc.arg(status), activated_at = sqlc.narg(activated_at),
    expires_at = sqlc.narg(expires_at), revoked_at = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id);

-- name: ListPendingGrantsByPhone :many
SELECT * FROM `learning_grants` WHERE phone = sqlc.arg(phone) AND status = 'pending' ORDER BY id;

-- name: ActivateGrant :execrows
-- Signup claim: only a still-pending grant flips (two concurrent claims activate it once).
UPDATE `learning_grants`
SET user_id = sqlc.arg(user_id), status = 'active', activated_at = sqlc.arg(activated_at),
    expires_at = sqlc.narg(expires_at), updated_at = sqlc.arg(activated_at)
WHERE id = sqlc.arg(id) AND status = 'pending' AND phone = sqlc.arg(phone);

-- name: GetInstructorGrant :one
SELECT * FROM `learning_grants` WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id) LIMIT 1;

-- name: RevokeGrant :execrows
UPDATE `learning_grants` SET status = 'revoked', revoked_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id) AND status <> 'revoked';

-- name: ListInstructorGrants :many
-- Ins_Students / Ins_Group members: every non-revoked grant of the instructor with the student's name (NULL while
-- pending), the group / course it opens.
SELECT a.*, u.name AS user_name, g.name AS group_name, c.title AS course_title
FROM `learning_grants` a
LEFT JOIN `users` u ON u.id = a.user_id
LEFT JOIN `learning_groups` g ON g.id = a.group_id
LEFT JOIN `learning_courses` c ON c.id = a.course_id
WHERE a.instructor_id = sqlc.arg(instructor_id) AND a.status <> 'revoked'
  AND (sqlc.narg(group_id) IS NULL OR a.group_id = sqlc.narg(group_id))
ORDER BY a.id DESC
LIMIT ?;

-- name: GetUserGrant :one
SELECT a.*, i.display_name AS instructor_name, i.title AS instructor_title, g.name AS group_name
FROM `learning_grants` a
JOIN `learning_instructors` i ON i.id = a.instructor_id
LEFT JOIN `learning_groups` g ON g.id = a.group_id
WHERE a.id = sqlc.arg(id) AND a.user_id = sqlc.arg(user_id) AND a.status = 'active' AND i.status = 'approved'
LIMIT 1;

-- name: ListGrantCourses :many
-- The published courses one grant opens (its course, or every course of its group).
SELECT c.* FROM `learning_courses` c
WHERE c.status = 'published' AND (
  c.id = sqlc.narg(course_id)
  OR c.id IN (SELECT gc.course_id FROM `learning_group_courses` gc WHERE gc.group_id = sqlc.narg(group_id))
)
ORDER BY c.sort_order, c.id;

-- ───────────── student access ─────────────

-- name: ListUserCourseAccess :many
-- Every (active grant × published course) pair of the user, expired grants included (the list shows «دسترسی تمام شد»);
-- the service keeps the best grant per course.
SELECT a.id AS grant_id, a.group_id, a.expires_at, a.activated_at,
  g.name AS group_name,
  c.id AS course_id, c.kind, c.title, c.description, c.cover_media_id, c.published_at,
  i.id AS instructor_id, i.display_name AS instructor_name, i.title AS instructor_title
FROM `learning_grants` a
JOIN `learning_instructors` i ON i.id = a.instructor_id AND i.status = 'approved'
LEFT JOIN `learning_groups` g ON g.id = a.group_id
JOIN `learning_courses` c ON c.instructor_id = a.instructor_id AND c.status = 'published' AND (
  c.id = a.course_id
  OR c.id IN (SELECT gc.course_id FROM `learning_group_courses` gc WHERE gc.group_id = a.group_id)
)
WHERE a.user_id = sqlc.arg(user_id) AND a.status = 'active'
ORDER BY a.activated_at DESC, a.id DESC;

-- name: ListUserProgressForCourses :many
SELECT * FROM `learning_progress`
WHERE user_id = sqlc.arg(user_id) AND course_id IN (sqlc.slice(course_ids));

-- name: ListProgressForUsersCourses :many
-- Group member progress (Ins_Group): rows of the given students in the given courses.
SELECT user_id, course_id, lesson_id, percent, completed_at FROM `learning_progress`
WHERE user_id IN (sqlc.slice(user_ids)) AND course_id IN (sqlc.slice(course_ids));

-- name: GetProgress :one
SELECT * FROM `learning_progress` WHERE user_id = sqlc.arg(user_id) AND lesson_id = sqlc.arg(lesson_id) LIMIT 1;

-- name: UpsertProgress :exec
INSERT INTO `learning_progress` (
  user_id, lesson_id, course_id, position_seconds, percent, completed_at, last_seen_at, created_at, updated_at
) VALUES (
  sqlc.arg(user_id), sqlc.arg(lesson_id), sqlc.arg(course_id), sqlc.arg(position_seconds), sqlc.arg(percent),
  sqlc.narg(completed_at), sqlc.arg(now), sqlc.arg(now), sqlc.arg(now)
) ON DUPLICATE KEY UPDATE
  position_seconds = VALUES(position_seconds), percent = VALUES(percent), completed_at = VALUES(completed_at),
  last_seen_at = VALUES(last_seen_at), updated_at = VALUES(updated_at);

-- ───────────── notifications ─────────────

-- name: InsertUserNotification :exec
-- Student inbox row «دوره برایت باز شد»: title/body are {lang: text} JSON.
INSERT INTO `user_notifications` (user_id, type, title, body, action_url, data, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(type), sqlc.arg(title), sqlc.arg(body), sqlc.arg(action_url), sqlc.arg(data),
  sqlc.arg(now), sqlc.arg(now));

-- name: InsertSMSOutbox :exec
INSERT INTO `learning_sms_outbox` (grant_id, status, due_at, attempts, created_at, updated_at)
VALUES (sqlc.arg(grant_id), 'pending', sqlc.arg(due_at), 0, sqlc.arg(now), sqlc.arg(now));

-- name: ListDueSMS :many
SELECT id FROM `learning_sms_outbox`
WHERE status = 'pending' AND due_at <= sqlc.arg(due_before) AND (lease_until IS NULL OR lease_until < sqlc.arg(lease_before))
ORDER BY due_at, id
LIMIT ?;

-- name: ClaimSMS :execrows
UPDATE `learning_sms_outbox` SET lease_until = sqlc.arg(lease_until), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending' AND due_at <= sqlc.arg(due_before)
  AND (lease_until IS NULL OR lease_until < sqlc.arg(lease_before));

-- name: GetSMSJob :one
SELECT o.id, o.attempts, a.id AS grant_id, a.phone, a.user_id, a.instructor_id, a.status AS grant_status
FROM `learning_sms_outbox` o
JOIN `learning_grants` a ON a.id = o.grant_id
WHERE o.id = sqlc.arg(id) LIMIT 1;

-- name: FinishSMS :exec
UPDATE `learning_sms_outbox`
SET status = sqlc.arg(status), reason = sqlc.narg(reason), attempts = sqlc.arg(attempts), sent_at = sqlc.narg(sent_at),
    lease_until = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: DeferSMS :exec
UPDATE `learning_sms_outbox`
SET due_at = sqlc.arg(due_at), attempts = sqlc.arg(attempts), reason = sqlc.narg(reason), lease_until = NULL,
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: CountSMSSentToPhone :one
SELECT COUNT(*) FROM `learning_sms_outbox` o
JOIN `learning_grants` a ON a.id = o.grant_id
WHERE a.phone = sqlc.arg(phone) AND o.status = 'sent' AND o.sent_at >= sqlc.arg(since);

-- name: CountSMSSentByInstructor :one
SELECT COUNT(*) FROM `learning_sms_outbox` o
JOIN `learning_grants` a ON a.id = o.grant_id
WHERE a.instructor_id = sqlc.arg(instructor_id) AND o.status = 'sent' AND o.sent_at >= sqlc.arg(since);

-- name: CountPendingSMSForGrant :one
SELECT COUNT(*) FROM `learning_sms_outbox` WHERE grant_id = sqlc.arg(grant_id) AND status = 'pending';

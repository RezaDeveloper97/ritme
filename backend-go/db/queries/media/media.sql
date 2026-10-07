-- Media pipeline of the courses domain (bloom B-N8-02, internal/media). Every instructor query is scoped by the
-- instructor id; a foreign id reads as missing.

-- name: GetOwnedLesson :one
SELECT l.id, l.course_id, l.kind, l.media_id, l.media_status
FROM `learning_lessons` l
JOIN `learning_courses` c ON c.id = l.course_id
WHERE l.id = sqlc.arg(id) AND c.instructor_id = sqlc.arg(instructor_id)
LIMIT 1;

-- name: GetLessonMediaRef :one
SELECT id, media_id, media_status FROM `learning_lessons` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: InsertMedia :execlastid
INSERT INTO `learning_media` (
  instructor_id, lesson_id, kind, mime, size_bytes, offset_bytes, path, status, upload_expires_at, created_at, updated_at
) VALUES (
  sqlc.arg(instructor_id), sqlc.arg(lesson_id), sqlc.arg(kind), NULL, sqlc.arg(size_bytes), 0, sqlc.arg(path),
  'uploading', sqlc.arg(upload_expires_at), sqlc.arg(now), sqlc.arg(now)
);

-- name: GetMedia :one
SELECT * FROM `learning_media` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: GetOwnedMedia :one
SELECT * FROM `learning_media` WHERE id = sqlc.arg(id) AND instructor_id = sqlc.arg(instructor_id) LIMIT 1;

-- name: AdvanceOffset :execrows
UPDATE `learning_media`
SET offset_bytes = sqlc.arg(new_offset), mime = COALESCE(mime, sqlc.narg(mime)), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND offset_bytes = sqlc.arg(old_offset) AND status = 'uploading';

-- name: SetMediaStatus :execrows
UPDATE `learning_media` SET status = sqlc.arg(status), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = sqlc.arg(from_status);

-- name: MarkMediaReady :execrows
UPDATE `learning_media`
SET status = 'ready', path = sqlc.arg(path), completed_at = sqlc.arg(now), upload_expires_at = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'processing';

-- name: DeleteMedia :execrows
DELETE FROM `learning_media` WHERE id = sqlc.arg(id);

-- name: ListLessonMedia :many
SELECT * FROM `learning_media` WHERE lesson_id = sqlc.arg(lesson_id) ORDER BY id;

-- name: SetLessonMedia :exec
UPDATE `learning_lessons`
SET media_id = sqlc.narg(media_id), media_status = sqlc.arg(media_status), size_bytes = sqlc.narg(size_bytes),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetLessonMediaStatus :exec
UPDATE `learning_lessons` SET media_status = sqlc.arg(media_status), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: CountActiveUploads :one
SELECT COUNT(*) FROM `learning_media` WHERE instructor_id = sqlc.arg(instructor_id) AND status IN ('uploading', 'processing');

-- name: SumInstructorBytes :one
SELECT CAST(COALESCE(SUM(size_bytes), 0) AS SIGNED) FROM `learning_media`
WHERE instructor_id = sqlc.arg(instructor_id) AND status IN ('uploading', 'processing', 'ready');

-- name: ListExpiredUploads :many
SELECT * FROM `learning_media`
WHERE status IN ('uploading', 'failed') AND upload_expires_at IS NOT NULL AND upload_expires_at < sqlc.arg(now)
ORDER BY id LIMIT 200;

-- name: ListOrphanMedia :many
SELECT * FROM `learning_media` WHERE lesson_id IS NULL ORDER BY id LIMIT 200;

-- name: ListInstructorMediaPaths :many
SELECT path FROM `learning_media` WHERE instructor_id = sqlc.arg(instructor_id);

-- name: ExistingInstructors :many
SELECT id FROM `learning_instructors` WHERE id IN (sqlc.slice(ids));

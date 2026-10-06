-- Lab analysis (bloom B-N6-06; internal/labs): lab sheets, their encrypted page files, markers and the DB-backed job
-- queue. Every user-facing query is scoped by user_id in the query itself (IDOR); the worker reads its job's lab by
-- id only (it has no request user) and writes nothing a user chose.

-- name: CreateLabReport :execlastid
INSERT INTO `lab_reports` (
  user_id, source, category, title, taken_on, fasting, status, progress, quota_at, verified_at, created_at, updated_at
) VALUES (
  sqlc.arg(user_id), sqlc.arg(source), sqlc.arg(category), sqlc.narg(title), sqlc.narg(taken_on), sqlc.narg(fasting),
  sqlc.arg(status), sqlc.arg(progress), sqlc.narg(quota_at), sqlc.narg(verified_at), sqlc.arg(now), sqlc.arg(now)
);

-- name: GetLabReport :one
SELECT * FROM `lab_reports` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: GetLabReportByID :one
-- The worker's read (the job names the lab; the job row carries the owner).
SELECT * FROM `lab_reports` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: ListLabReports :many
-- The user's labs, newest sheet first (sheet date, else upload day), capped by the caller.
SELECT * FROM `lab_reports`
WHERE user_id = sqlc.arg(user_id)
ORDER BY COALESCE(taken_on, DATE(created_at)) DESC, id DESC
LIMIT ?;

-- name: UpdateLabMeta :execrows
UPDATE `lab_reports`
SET category = sqlc.arg(category), title = sqlc.narg(title), taken_on = sqlc.narg(taken_on), fasting = sqlc.narg(fasting),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: SetLabStatus :exec
UPDATE `lab_reports`
SET status = sqlc.arg(status), progress = sqlc.arg(progress), error_code = sqlc.narg(error_code), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetLabExtracted :exec
-- Extraction done: needs_review; the sheet's date and lab fill only what the user left empty.
UPDATE `lab_reports`
SET status = 'needs_review', progress = 100, error_code = NULL,
    taken_on = COALESCE(taken_on, sqlc.narg(taken_on)), lab_name = COALESCE(lab_name, sqlc.narg(lab_name)),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: MarkLabVerified :execrows
-- The user confirmed the values: one more interpretation, while fewer than max_count were made.
UPDATE `lab_reports`
SET verified_at = sqlc.arg(now), status = sqlc.arg(status), progress = 0, error_code = NULL,
    interpret_count = interpret_count + 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND interpret_count < sqlc.arg(max_count);

-- name: SetLabInterpretation :exec
UPDATE `lab_reports`
SET interpretation = sqlc.arg(interpretation), interpreted_at = sqlc.arg(now), status = 'ready', progress = 100,
    error_code = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ClearLabQuota :execrows
-- Claims the refund of the reserved Plus use: only the first caller sees one affected row.
UPDATE `lab_reports` SET quota_at = NULL WHERE id = sqlc.arg(id) AND quota_at IS NOT NULL;

-- name: SetLabFeedback :execrows
UPDATE `lab_reports`
SET feedback = sqlc.arg(feedback), feedback_note = sqlc.narg(feedback_note), feedback_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: TouchLab :exec
UPDATE `lab_reports` SET updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteLabReport :execrows
DELETE FROM `lab_reports` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: CountLabReportsSince :one
-- Uploads of the user since a time (per-user daily cap on top of the Plus quota).
SELECT COUNT(*) FROM `lab_reports`
WHERE user_id = sqlc.arg(user_id) AND source = 'upload' AND created_at >= sqlc.arg(since);

-- name: CreateLabFile :execlastid
INSERT INTO `lab_files` (lab_id, user_id, page, mime, size_bytes, path, created_at, updated_at)
VALUES (sqlc.arg(lab_id), sqlc.arg(user_id), sqlc.arg(page), sqlc.arg(mime), sqlc.arg(size_bytes), sqlc.arg(path), sqlc.arg(now), sqlc.arg(now));

-- name: ListLabFiles :many
SELECT * FROM `lab_files` WHERE lab_id = sqlc.arg(lab_id) AND user_id = sqlc.arg(user_id) ORDER BY page, id;

-- name: GetLabFile :one
SELECT * FROM `lab_files` WHERE id = sqlc.arg(id) AND lab_id = sqlc.arg(lab_id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: DeleteLabFile :execrows
DELETE FROM `lab_files` WHERE id = sqlc.arg(id) AND lab_id = sqlc.arg(lab_id) AND user_id = sqlc.arg(user_id);

-- name: CreateLabMarker :execlastid
INSERT INTO `lab_markers` (
  lab_id, user_id, code, name, value, value_text, unit, ref_low, ref_high, ref_text, confidence, source, sort_order,
  created_at, updated_at
) VALUES (
  sqlc.arg(lab_id), sqlc.arg(user_id), sqlc.narg(code), sqlc.arg(name), sqlc.narg(value), sqlc.narg(value_text),
  sqlc.narg(unit), sqlc.narg(ref_low), sqlc.narg(ref_high), sqlc.narg(ref_text), sqlc.narg(confidence), sqlc.arg(source),
  sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now)
);

-- name: ListLabMarkers :many
SELECT * FROM `lab_markers` WHERE lab_id = sqlc.arg(lab_id) AND user_id = sqlc.arg(user_id) ORDER BY sort_order, id;

-- name: GetLabMarker :one
SELECT * FROM `lab_markers` WHERE id = sqlc.arg(id) AND lab_id = sqlc.arg(lab_id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: UpdateLabMarker :execrows
UPDATE `lab_markers`
SET code = sqlc.narg(code), name = sqlc.arg(name), value = sqlc.narg(value), value_text = sqlc.narg(value_text),
    unit = sqlc.narg(unit), ref_low = sqlc.narg(ref_low), ref_high = sqlc.narg(ref_high), ref_text = sqlc.narg(ref_text),
    source = sqlc.arg(source), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND lab_id = sqlc.arg(lab_id) AND user_id = sqlc.arg(user_id);

-- name: DeleteLabMarker :execrows
DELETE FROM `lab_markers` WHERE id = sqlc.arg(id) AND lab_id = sqlc.arg(lab_id) AND user_id = sqlc.arg(user_id);

-- name: DeleteExtractedLabMarkers :exec
-- A retried extraction replaces what an earlier try wrote.
DELETE FROM `lab_markers` WHERE lab_id = sqlc.arg(lab_id) AND source = 'extracted';

-- name: NextLabMarkerSort :one
SELECT CAST(COALESCE(MAX(sort_order), 0) + 1 AS UNSIGNED) FROM `lab_markers` WHERE lab_id = sqlc.arg(lab_id);

-- name: CountLabMarkers :one
SELECT COUNT(*) FROM `lab_markers` WHERE lab_id = sqlc.arg(lab_id) AND user_id = sqlc.arg(user_id);

-- name: ListUserLabMarkers :many
-- Every marker of the user's labs with its lab's date and state (history counts, trends, the analysis hub).
SELECT m.*, r.taken_on AS lab_taken_on, r.created_at AS lab_created_at, r.verified_at AS lab_verified_at,
       r.status AS lab_status, r.source AS lab_source
FROM `lab_markers` m
JOIN `lab_reports` r ON r.id = m.lab_id AND r.user_id = m.user_id
WHERE m.user_id = sqlc.arg(user_id)
ORDER BY COALESCE(r.taken_on, DATE(r.created_at)), r.id, m.sort_order, m.id;

-- name: CreateLabJob :execlastid
INSERT INTO `lab_jobs` (lab_id, user_id, kind, locale, status, attempts, available_at, created_at, updated_at)
VALUES (sqlc.arg(lab_id), sqlc.arg(user_id), sqlc.arg(kind), sqlc.narg(locale), 'pending', 0, sqlc.arg(available_at), sqlc.arg(now), sqlc.arg(now));

-- name: NextLabJob :one
-- The oldest job a worker may take: pending and due, or running with an expired lease (its worker died).
SELECT id FROM `lab_jobs`
WHERE (status = 'pending' AND available_at <= sqlc.arg(due)) OR (status = 'running' AND locked_until < sqlc.arg(expired))
ORDER BY id LIMIT 1;

-- name: ClaimLabJob :execrows
-- Atomic claim: only one worker turns the job running (the same condition as NextLabJob).
UPDATE `lab_jobs`
SET status = 'running', locked_until = sqlc.arg(locked_until), attempts = attempts + 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
  AND ((status = 'pending' AND available_at <= sqlc.arg(due)) OR (status = 'running' AND locked_until < sqlc.arg(expired)));

-- name: GetLabJob :one
SELECT * FROM `lab_jobs` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: FinishLabJob :exec
UPDATE `lab_jobs`
SET status = sqlc.arg(status), last_error = sqlc.narg(last_error), locked_until = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: RetryLabJob :exec
UPDATE `lab_jobs`
SET status = 'pending', available_at = sqlc.arg(available_at), locked_until = NULL, last_error = sqlc.narg(last_error),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: GetLabUserContext :one
-- What the interpretation may know: name (PII redaction only), age, the cycle inputs and the life mode inputs
-- (enums.ResolveLifeMode), in one read.
SELECT
  u.name,
  p.birthday,
  p.last_period_start,
  p.cycle_duration,
  p.period_duration,
  p.user_goal,
  CAST(EXISTS (
    SELECT 1 FROM `pregnancy_profiles` pp WHERE pp.user_id = u.id AND pp.pregnancy_mode = 1
  ) AS SIGNED) AS pregnant,
  lp.life_mode
FROM `users` u
LEFT JOIN `user_profiles` p ON p.user_id = u.id
LEFT JOIN `user_life_profiles` lp ON lp.user_id = u.id
WHERE u.id = sqlc.arg(user_id)
LIMIT 1;

-- name: ListLabMedicationNames :many
-- The user's own active medications (reminders type medication), names only.
SELECT title FROM `reminders`
WHERE user_id = sqlc.arg(user_id) AND `type` = 'medication' AND is_active = 1
ORDER BY id LIMIT 10;

-- name: LabUserExists :one
SELECT COUNT(*) FROM `users` WHERE id = sqlc.arg(user_id);

-- name: DeleteFinishedLabJobs :exec
-- Housekeeping: finished jobs older than a cut-off (the lab keeps its own status).
DELETE FROM `lab_jobs` WHERE status IN ('done', 'failed') AND updated_at < sqlc.arg(before);

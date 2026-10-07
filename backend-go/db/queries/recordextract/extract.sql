-- Record document extraction (canvas-build CB-REC-02; internal/healthrecord/extract). No table of its own: the job
-- state lives in record_documents.extracted (key "job", never shown to clients) while review_state = 'pending'.
-- Every user-facing query is scoped by user_id in the query itself (IDOR); the worker's queries take the id and the
-- user id it read from the claimed row.

-- name: GetExtractDocument :one
SELECT * FROM `record_documents` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: LockExtractDocument :one
SELECT * FROM `record_documents` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1 FOR UPDATE;

-- name: ListPendingExtractions :many
-- Candidate documents for a worker (inside the claim transaction). SKIP LOCKED: concurrent workers never wait on,
-- or both take, the same row; due / lease / attempts are decided in Go from the job JSON.
SELECT id, user_id FROM `record_documents`
WHERE review_state = 'pending'
ORDER BY updated_at, id
LIMIT 20
FOR UPDATE SKIP LOCKED;

-- name: SetExtractState :execrows
UPDATE `record_documents`
SET review_state = sqlc.arg(review_state), extracted = sqlc.narg(extracted), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: SetExtractJob :execrows
-- A job bookkeeping write (claim / retry). updated_at moves on, so a claimed or backed-off job goes to the back of
-- ListPendingExtractions' order and can never starve the jobs behind it (security audit L7).
UPDATE `record_documents`
SET extracted = sqlc.narg(extracted), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND review_state = 'pending';

-- name: ApplyExtractReview :execrows
UPDATE `record_documents`
SET document_date = sqlc.narg(document_date), ended_on = sqlc.narg(ended_on), centre = sqlc.narg(centre),
    doctor = sqlc.narg(doctor), review_state = sqlc.arg(review_state), extracted = sqlc.narg(extracted),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListExtractFileIDs :many
SELECT file_id FROM `record_document_files`
WHERE document_id = sqlc.arg(document_id) AND user_id = sqlc.arg(user_id)
ORDER BY position, id;

-- name: CountDocExtractCallsSince :one
-- Extraction calls of the user since a time, from the append-only AI usage log: deleting documents never resets the
-- refund allowance of paid-but-useless extractions.
SELECT COUNT(*) FROM `ai_usage_logs`
WHERE user_id = sqlc.arg(user_id) AND feature = 'doc_extract' AND op = 'extract' AND created_at >= sqlc.arg(since);

-- name: GetExtractUserName :one
SELECT name FROM `users` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: LockExtractPregnancyProfile :one
-- The user's pregnancy profile row, locked for the dating update (read through internal/pregnancy afterwards).
SELECT id FROM `pregnancy_profiles` WHERE user_id = sqlc.arg(user_id) LIMIT 1 FOR UPDATE;

-- name: UpsertExtractDocumentLink :exec
-- The «where used» link of the dating update, inside the same transaction (same statement as healthrecord's
-- UpsertRecordDocumentLink).
INSERT INTO `record_document_links` (user_id, document_id, target_type, target_id, state, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(document_id), sqlc.arg(target_type), sqlc.arg(target_id), sqlc.arg(state),
        sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE state = VALUES(state), updated_at = VALUES(updated_at);

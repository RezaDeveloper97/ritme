-- Record documents, extras and timeline (canvas-build CB-REC-01; internal/healthrecord documents*.go, extras.go,
-- timeline.go). Every query is scoped by user_id in the query itself (IDOR): a document, its files and its links are
-- only ever read or written together with the owner's id.

-- name: GetRecordExtras :one
SELECT allergies, allergies_on_emergency_card, surgeries, family_history, updated_at FROM `health_records`
WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- name: UpsertRecordExtras :exec
-- Only the CB-REC-01 columns: blood type and allergies stay with UpsertHealthRecord (B-N6-03).
INSERT INTO `health_records` (user_id, allergies_on_emergency_card, surgeries, family_history, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(allergies_on_emergency_card), sqlc.narg(surgeries), sqlc.narg(family_history),
        sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE allergies_on_emergency_card = VALUES(allergies_on_emergency_card),
    surgeries = VALUES(surgeries), family_history = VALUES(family_history), updated_at = VALUES(updated_at);

-- name: CountRecordDocuments :one
SELECT COUNT(*) FROM `record_documents` WHERE user_id = sqlc.arg(user_id);

-- name: CountRecordDocumentsByKind :many
SELECT kind, COUNT(*) AS documents FROM `record_documents` WHERE user_id = sqlc.arg(user_id) GROUP BY kind;

-- name: CountRecordDocumentsInReview :one
SELECT COUNT(*) FROM `record_documents`
WHERE user_id = sqlc.arg(user_id) AND review_state = 'needs_review';

-- name: ListRecordDocuments :many
-- Newest first by the document's date (the upload day when the date is not known yet).
SELECT d.*, CAST(COALESCE(d.document_date, DATE(d.created_at)) AS DATE) AS sort_date,
       (SELECT COUNT(*) FROM `record_document_files` f WHERE f.document_id = d.id AND f.user_id = d.user_id) AS file_count
FROM `record_documents` d
WHERE d.user_id = sqlc.arg(user_id)
ORDER BY sort_date DESC, d.id DESC;

-- name: GetRecordDocument :one
SELECT * FROM `record_documents` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: InsertRecordDocument :execlastid
INSERT INTO `record_documents` (user_id, kind, title, document_date, ended_on, centre, doctor, note, review_state,
    created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(kind), sqlc.narg(title), sqlc.narg(document_date), sqlc.narg(ended_on),
        sqlc.narg(centre), sqlc.narg(doctor), sqlc.narg(note), sqlc.arg(review_state), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateRecordDocument :execrows
UPDATE `record_documents`
SET kind = sqlc.arg(kind), title = sqlc.narg(title), document_date = sqlc.narg(document_date),
    ended_on = sqlc.narg(ended_on), centre = sqlc.narg(centre), doctor = sqlc.narg(doctor), note = sqlc.narg(note),
    review_state = sqlc.arg(review_state), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteRecordDocument :execrows
DELETE FROM `record_documents` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListRecordDocumentFiles :many
SELECT file_id FROM `record_document_files`
WHERE document_id = sqlc.arg(document_id) AND user_id = sqlc.arg(user_id)
ORDER BY position, id;

-- name: GetRecordFileDocument :one
-- The document a file is attached to (a file belongs to at most one document).
SELECT document_id FROM `record_document_files`
WHERE file_id = sqlc.arg(file_id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: InsertRecordDocumentFile :exec
INSERT INTO `record_document_files` (user_id, document_id, file_id, position, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(document_id), sqlc.arg(file_id), sqlc.arg(position), sqlc.arg(now), sqlc.arg(now));

-- name: DeleteRecordDocumentFiles :exec
DELETE FROM `record_document_files` WHERE document_id = sqlc.arg(document_id) AND user_id = sqlc.arg(user_id);

-- name: ListRecordDocumentLinks :many
SELECT * FROM `record_document_links`
WHERE document_id = sqlc.arg(document_id) AND user_id = sqlc.arg(user_id)
ORDER BY created_at, id;

-- name: ListUserRecordDocumentLinks :many
-- Every link of the user's documents (the timeline's badges).
SELECT document_id, target_type, target_id, state FROM `record_document_links`
WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at, id;

-- name: UpsertRecordDocumentLink :exec
INSERT INTO `record_document_links` (user_id, document_id, target_type, target_id, state, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(document_id), sqlc.arg(target_type), sqlc.arg(target_id), sqlc.arg(state),
        sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE state = VALUES(state), updated_at = VALUES(updated_at);

-- name: DeleteRecordDocumentLink :execrows
DELETE FROM `record_document_links`
WHERE document_id = sqlc.arg(document_id) AND user_id = sqlc.arg(user_id) AND target_type = sqlc.arg(target_type)
  AND target_id = sqlc.arg(target_id);

-- name: LockRecordOwner :one
-- First statement of a document create: the user row lock serialises one user's creates, so the MaxDocuments count
-- that follows cannot race (same as internal/files LockOwner).
SELECT id FROM `users` WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: LockRecordDocument :one
SELECT id FROM `record_documents` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1 FOR UPDATE;

-- name: LockRecordDocumentFiles :many
-- The document's files with their storage rows, locked, inside the update / delete transaction.
SELECT f.id, f.purpose, f.path FROM `record_document_files` r
JOIN `files` f ON f.id = r.file_id AND f.user_id = r.user_id
WHERE r.document_id = sqlc.arg(document_id) AND r.user_id = sqlc.arg(user_id)
ORDER BY r.position, r.id
FOR UPDATE;

-- name: DeleteDetachedRecordFile :execrows
-- Deletes a record_document file of the user that no document uses any more (inside the transaction that detached
-- it; the blob is removed after commit).
DELETE FROM `files`
WHERE `files`.id = sqlc.arg(id) AND `files`.user_id = sqlc.arg(user_id) AND `files`.purpose = 'record_document'
  AND NOT EXISTS (SELECT 1 FROM `record_document_files` r WHERE r.file_id = `files`.id);

-- name: GetRecordPregnancyProfileID :one
-- A pregnancy link target must be the user's own pregnancy profile.
SELECT id FROM `pregnancy_profiles` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

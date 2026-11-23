-- AI assistant clinic (bloom B-N7-06, goose 00052). Every user-facing read and write is scoped by user_id in the query.

-- name: ListActiveDepartments :many
SELECT * FROM `assistant_departments` WHERE is_active = 1 ORDER BY sort_order, id;

-- name: GetActiveDepartmentByCode :one
SELECT * FROM `assistant_departments` WHERE code = ? AND is_active = 1 LIMIT 1;

-- name: GetDepartment :one
SELECT * FROM `assistant_departments` WHERE id = ? LIMIT 1;

-- name: ListSafetyTerms :many
SELECT kind, locale, term FROM `assistant_safety_terms` WHERE is_active = 1 ORDER BY id;

-- name: GetProfileSettings :one
SELECT shared FROM `assistant_profile_settings` WHERE user_id = ? LIMIT 1;

-- name: UpsertProfileSettings :exec
INSERT INTO `assistant_profile_settings` (user_id, shared, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(shared), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE shared = VALUES(shared), updated_at = VALUES(updated_at);

-- name: CountDepartmentConversations :many
SELECT department_id, COUNT(*) AS conversations FROM `assistant_conversations` WHERE user_id = ? GROUP BY department_id;

-- name: InsertConversation :execlastid
INSERT INTO `assistant_conversations` (user_id, department_id, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(department_id), sqlc.arg(now), sqlc.arg(now));

-- name: GetConversation :one
SELECT * FROM `assistant_conversations` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: ListConversations :many
SELECT * FROM `assistant_conversations` WHERE user_id = sqlc.arg(user_id)
ORDER BY COALESCE(last_message_at, created_at) DESC, id DESC LIMIT 100;

-- name: ListDepartmentConversations :many
SELECT * FROM `assistant_conversations` WHERE user_id = sqlc.arg(user_id) AND department_id = sqlc.arg(department_id)
ORDER BY COALESCE(last_message_at, created_at) DESC, id DESC LIMIT 100;

-- name: TouchConversation :exec
UPDATE `assistant_conversations`
SET title = NULL, -- B-N7-06b: the title is derived from the sealed first message; a legacy plaintext title is cleared
    triage_level = COALESCE(sqlc.narg(triage_level), triage_level),
    emergency = emergency OR sqlc.arg(emergency),
    message_count = message_count + sqlc.arg(added),
    last_message_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: SaveSummary :exec
UPDATE `assistant_conversations`
SET summary = sqlc.arg(summary), summary_at = sqlc.arg(now), summary_message_count = message_count, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteConversation :execrows
DELETE FROM `assistant_conversations` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteAllConversations :execrows
DELETE FROM `assistant_conversations` WHERE user_id = ?;

-- name: InsertMessage :execlastid
INSERT INTO `assistant_messages` (conversation_id, role, body, photo_file_id, card, triage_level, guardrail, chips, status, created_at, updated_at)
VALUES (sqlc.arg(conversation_id), sqlc.arg(role), sqlc.arg(body), sqlc.narg(photo_file_id), sqlc.narg(card), sqlc.narg(triage_level),
        sqlc.narg(guardrail), sqlc.narg(chips), sqlc.arg(status), sqlc.arg(now), sqlc.arg(now));

-- name: SetMessageBody :exec
-- B-N7-06b: the body is sealed with the row id in its additional data, so it is written right after the insert (same tx).
UPDATE `assistant_messages` SET body = sqlc.arg(body) WHERE id = sqlc.arg(id);

-- name: FirstUserMessages :many
-- The first user turn of each of the user's conversations (the sealed source of the title).
SELECT m.id, m.conversation_id, m.body FROM `assistant_messages` m
JOIN `assistant_conversations` c ON c.id = m.conversation_id
WHERE c.user_id = sqlc.arg(user_id) AND (sqlc.arg(conversation_id) = 0 OR c.id = sqlc.arg(conversation_id)) AND m.role = 'user'
  AND m.id = (SELECT MIN(m2.id) FROM `assistant_messages` m2 WHERE m2.conversation_id = m.conversation_id AND m2.role = 'user');

-- name: ListMessages :many
SELECT * FROM `assistant_messages` WHERE conversation_id = ? ORDER BY id;

-- name: ListRecentChatMessages :many
-- The newest user / assistant turns of a conversation that go back to the model (newest first; the caller reverses):
-- cards and empty assistant replies (a failed stream) never do, so they do not take a slot (T-M12-32).
SELECT * FROM `assistant_messages`
WHERE conversation_id = ?
  AND (role = 'user' OR (role = 'assistant' AND body <> ''))
ORDER BY id DESC
LIMIT ?;

-- name: ListUserPhotoIDs :many
-- Photos of the user's conversations (one, or all when conversation_id = 0), removed with them (hard delete).
SELECT m.photo_file_id FROM `assistant_messages` m
JOIN `assistant_conversations` c ON c.id = m.conversation_id
WHERE c.user_id = sqlc.arg(user_id) AND (sqlc.arg(conversation_id) = 0 OR c.id = sqlc.arg(conversation_id))
  AND m.photo_file_id IS NOT NULL;

-- name: ListUserAssistantPhotos :many
-- Every assistant photo file of the user, attached or not (delete-all, B-N7-06b M4).
SELECT id FROM `files` WHERE user_id = sqlc.arg(user_id) AND purpose = 'assistant_photo';

-- name: ListStaleAssistantPhotos :many
-- Uploaded photos never attached to a message and older than the cut-off (swept, B-N7-06b M4).
SELECT f.id FROM `files` f
WHERE f.user_id = sqlc.arg(user_id) AND f.purpose = 'assistant_photo' AND f.created_at < sqlc.arg(before)
  AND NOT EXISTS (SELECT 1 FROM `assistant_messages` m WHERE m.photo_file_id = f.id)
LIMIT 50;

-- name: CountPhotoUses :one
SELECT COUNT(*) FROM `assistant_messages` WHERE photo_file_id = ?;

-- name: UpsertHandoff :exec
INSERT INTO `assistant_handoffs` (conversation_id, user_id, booking_id, summary, triage_level, created_at, updated_at)
VALUES (sqlc.arg(conversation_id), sqlc.arg(user_id), sqlc.arg(booking_id), sqlc.arg(summary), sqlc.narg(triage_level), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE summary = VALUES(summary), triage_level = VALUES(triage_level), updated_at = VALUES(updated_at);

-- name: ListConversationHandoffs :many
SELECT * FROM `assistant_handoffs` WHERE conversation_id = sqlc.arg(conversation_id) AND user_id = sqlc.arg(user_id) ORDER BY id;

-- name: ListBookingHandoffs :many
-- Doctor side: only after telemed AuthorizeShare(doctor, booking, assistant_summaries) passed.
SELECT h.*, d.code AS department_code, d.name AS department_name FROM `assistant_handoffs` h
JOIN `assistant_conversations` c ON c.id = h.conversation_id
JOIN `assistant_departments` d ON d.id = c.department_id
WHERE h.booking_id = ? ORDER BY h.id DESC LIMIT 5;

-- Emergency card «کارت اضطراری» (canvas-build CB-REC-03, D-72; internal/emergency). Owner queries are scoped by
-- user_id in the query itself; the public card goes by the SHA-256 of its token only.

-- name: GetEmergencyCard :one
SELECT * FROM `emergency_cards` WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- name: EnsureEmergencyCard :exec
INSERT INTO `emergency_cards` (user_id, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE id = id;

-- name: UpdateEmergencyCard :exec
UPDATE `emergency_cards`
SET show_on_lock_screen = sqlc.arg(show_on_lock_screen), show_pregnancy = sqlc.arg(show_pregnancy),
    contact_name = sqlc.arg(contact_name), contact_relation = sqlc.arg(contact_relation),
    contact_phone = sqlc.arg(contact_phone), insurance_label = sqlc.arg(insurance_label),
    insurance_last4 = sqlc.arg(insurance_last4), updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id);

-- name: SetEmergencyPublicToken :exec
-- Turns the public card on with a new token (the old one stops working) or off (NULL).
UPDATE `emergency_cards`
SET public_token_hash = sqlc.arg(token_hash), public_enabled_at = sqlc.arg(enabled_at), public_view_count = 0,
    public_last_viewed_at = NULL, updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id);

-- name: GetEmergencyCardByToken :one
SELECT * FROM `emergency_cards` WHERE public_token_hash = sqlc.arg(token_hash) LIMIT 1;

-- name: CountEmergencyCardView :exec
UPDATE `emergency_cards`
SET public_view_count = public_view_count + 1, public_last_viewed_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: GetEmergencyAllergiesFlag :one
-- The record's allergies flag (CB-REC-01: health_records.allergies_on_emergency_card, default on).
SELECT allergies_on_emergency_card FROM `health_records` WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- Admin companion «همدم» module (B-N4-07): the companion tips copy (message_contents group companion_tip, fixed slots
-- of internal/admin/messages/registry) and a read-only overview of companion links. The overview selects link
-- metadata and the people's name / mobile only (masked in Go): never an invite code hash, never grants' data, never
-- anything of the owner's health records. Status / type filters are LIKE patterns ('%' = all).

-- name: ListCompanionTipRows :many
SELECT * FROM `message_contents` WHERE `group` = 'companion_tip' ORDER BY item_key, locale;

-- name: InsertCompanionTipRow :exec
INSERT INTO `message_contents` (`group`, item_key, locale, label, payload, is_active, is_approved, sort_order,
                                created_at, updated_at)
VALUES ('companion_tip', sqlc.arg(item_key), sqlc.arg(locale), sqlc.narg(label), sqlc.arg(payload), 1, 1,
        sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: SetCompanionTipRow :exec
-- An editor save makes the row live again (active + approved): the tips page is the copy the panel shows.
UPDATE `message_contents` SET payload = sqlc.arg(payload), is_active = 1, is_approved = 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND `group` = 'companion_tip';

-- name: DeleteCompanionTipRows :execresult
DELETE FROM `message_contents` WHERE `group` = 'companion_tip' AND locale = sqlc.arg(locale)
  AND item_key IN (sqlc.slice(item_keys));

-- name: CountCompanionLinksByTypeStatus :many
SELECT `type`, `status`, COUNT(*) AS total FROM `companions` GROUP BY `type`, `status`;

-- name: CountAdminCompanionLinks :one
SELECT COUNT(*) FROM `companions` WHERE `status` LIKE sqlc.arg(status) AND `type` LIKE sqlc.arg(type);

-- name: ListAdminCompanionLinks :many
SELECT c.id, c.type, c.status, c.display_name, c.invited_at, c.accepted_at, c.revoked_at, c.revoked_by, c.created_at,
       c.owner_id, o.name AS owner_name, o.mobile AS owner_mobile,
       c.companion_user_id, cu.name AS companion_name, cu.mobile AS companion_mobile,
       (SELECT COUNT(*) FROM `companion_grants` g WHERE g.companion_id = c.id) AS grants_count,
       (SELECT ci.phone FROM `companion_invites` ci
        WHERE ci.companion_id = c.id ORDER BY ci.id DESC LIMIT 1) AS invite_phone,
       (SELECT ci.expires_at FROM `companion_invites` ci
        WHERE ci.companion_id = c.id AND ci.used_at IS NULL AND ci.revoked_at IS NULL
        ORDER BY ci.id DESC LIMIT 1) AS invite_expires_at
FROM `companions` c
JOIN `users` o ON o.id = c.owner_id
LEFT JOIN `users` cu ON cu.id = c.companion_user_id
WHERE c.status LIKE sqlc.arg(status) AND c.type LIKE sqlc.arg(type)
ORDER BY c.created_at DESC, c.id DESC
LIMIT ? OFFSET ?;

-- Pregnancy v2 admin API (T-M7-06, docs/go-migration/admin-api.md §13): care-item delete guard and the
-- message_contents writes of the alert-rule editor and POST /messages (create in a registered group).

-- name: CountCareItemAppointments :many
-- Appointments (reminders type = 'appointment') linked to each care item through meta.care_item_key.
SELECT c.id, COUNT(*) AS appointments
FROM `pregnancy_care_items` c
JOIN `reminders` r ON r.`type` = 'appointment' AND JSON_UNQUOTE(JSON_EXTRACT(r.meta, '$.care_item_key')) = c.`key`
GROUP BY c.id;

-- name: CountAppointmentsOfCareItem :one
SELECT COUNT(*) FROM `pregnancy_care_items` c
JOIN `reminders` r ON r.`type` = 'appointment' AND JSON_UNQUOTE(JSON_EXTRACT(r.meta, '$.care_item_key')) = c.`key`
WHERE c.id = sqlc.arg(id);

-- name: DeleteUnlinkedCareItem :execrows
-- Deletes the item only while no appointment references its key (one statement: no race with a new booking).
DELETE FROM `pregnancy_care_items`
WHERE `pregnancy_care_items`.id = sqlc.arg(id) AND NOT EXISTS (
  SELECT 1 FROM `reminders` r
  WHERE r.`type` = 'appointment'
    AND JSON_UNQUOTE(JSON_EXTRACT(r.meta, '$.care_item_key')) = `pregnancy_care_items`.`key`
);

-- name: NextCareItemSortOrder :one
SELECT CAST(COALESCE(MAX(sort_order), 0) + 1 AS SIGNED) FROM `pregnancy_care_items`;

-- name: ListMessageContentsOfGroup :many
SELECT * FROM `message_contents` WHERE `group` = ? ORDER BY item_key, locale;

-- name: ListMessageContentKeys :many
-- Every (group, item_key, locale) triple — the "missing rows of registered groups" view.
SELECT `group`, item_key, locale FROM `message_contents` ORDER BY `group`, item_key, locale;

-- name: MessageContentExists :one
SELECT EXISTS(
  SELECT 1 FROM `message_contents`
  WHERE `group` = sqlc.arg(msg_group) AND item_key = sqlc.arg(item_key) AND locale = sqlc.arg(locale)
) AS found;

-- name: InsertMessageContent :execlastid
INSERT INTO `message_contents` (`group`, item_key, locale, label, payload, is_active, is_approved, sort_order,
                                created_at, updated_at)
VALUES (sqlc.arg(msg_group), sqlc.arg(item_key), sqlc.arg(locale), sqlc.narg(label), sqlc.arg(payload),
        sqlc.arg(is_active), sqlc.arg(is_approved), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: SetMessageContentPayload :exec
UPDATE `message_contents` SET payload = sqlc.arg(payload), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

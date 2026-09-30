-- name: ListActiveInfoPageSections :many
-- GET /info-pages/{group} (B-N1-12, Go only): the info boxes of one screen with their stable key and last edit.
SELECT id, `key`, heading, body, link_label, link_url, updated_at
FROM `info_sections`
WHERE `group` = ? AND is_active = 1
ORDER BY sort_order, id;

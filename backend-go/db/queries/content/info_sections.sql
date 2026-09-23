-- info_sections (App\Models\InfoSection), read side of InfoController.

-- name: ListActiveInfoSections :many
-- InfoSection::inGroup($group)->active()->ordered()->get().
SELECT id, heading, body, link_label, link_url
FROM `info_sections`
WHERE `group` = ? AND is_active = 1
ORDER BY sort_order, id;

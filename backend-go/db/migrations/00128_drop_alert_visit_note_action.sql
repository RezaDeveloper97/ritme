-- 00128_drop_alert_visit_note_action.sql — data only: alerts and notifications lose «افزودن به یادداشت ویزیت».
--
-- Product decision (2026-11-17): an alert / notification is a message and nothing more, so the
-- `add_to_visit_note` action is gone from the app, the API (POST /alerts/{id}/visit-note and the pregnancy v2
-- action of that name) and the admin action list (registry.AlertActions). The code-held pregnancy rules no longer
-- offer it; this removes it from the admin-editable alert copies in `message_contents` (group `pregnancy_alert`,
-- seeded by 00005), whose `actions` would otherwise fail the admin form's enum check on the next save.
--
-- Every other action is kept in its order with its (possibly admin-edited) label. Rows without the action are not
-- touched; `updated_at` is not touched (a data fix, not an edit). Re-running is a no-op. Rows already in the
-- `added_to_visit_note` inbox state stay as they are (they read as acknowledged).
-- Down is a no-op: the removed labels were per-row copy and the action no longer exists in code.
-- No Laravel twin: a guarded update with no row-count effect (`make schema-diff`); prod gets it at cutover
-- (T-M2-27), as 00008 did.
-- The UPDATE goes through PREPARE (T-M12-19c) so sqlc, which parses every Up section, never sees the JSON_TABLE /
-- JSON_ARRAYAGG(… ORDER BY …) syntax its MySQL parser rejects. The statement text is unchanged (only its quotes are
-- doubled inside the literal), so MariaDB runs exactly what it ran when 00128 was first applied; goose tracks
-- versions, not file checksums, so stage/beta see nothing new.
-- Test: db/migrations/drop_alert_visit_note_action_int_test.go (`make test-int PKG=./db/migrations/...`).

-- +goose Up
PREPARE `drop_visit_note` FROM 'UPDATE `message_contents` mc
   SET mc.`payload` = JSON_SET(mc.`payload`, ''$.actions'', (
         SELECT COALESCE(JSON_ARRAYAGG(JSON_OBJECT(''key'', jt.k, ''label'', jt.l) ORDER BY jt.n), JSON_ARRAY())
           FROM JSON_TABLE(mc.`payload`, ''$.actions[*]'' COLUMNS (
                  n FOR ORDINALITY,
                  k VARCHAR(64) PATH ''$.key'',
                  l VARCHAR(255) PATH ''$.label''
                )) AS jt
          WHERE jt.k <> ''add_to_visit_note''))
 WHERE mc.`group` = ''pregnancy_alert''
   AND JSON_SEARCH(mc.`payload`, ''one'', ''add_to_visit_note'', NULL, ''$.actions[*].key'') IS NOT NULL';
EXECUTE `drop_visit_note`;
DEALLOCATE PREPARE `drop_visit_note`;

-- +goose Down
SELECT 1;

-- 00110_family_grants_v6.sql — legacy companion grants → v6 codes and levels (T-M11-05; docs/engine-v6/family.md §3;
-- DECISIONS.md F-03, F-23; deviations D-193, D-204). Data only — no schema change, so there is no Laravel twin
-- (production Laravel keeps its own rows; Go reads both spellings through the companion compatibility mapping).
--
--   section symptoms            deleted (F-03), with one owner inbox notice per owner «اشتراک «علائم» تمام شد»
--   level none                  deleted (no row = off)
--   cycle (view | edit)         → cycle_phase view (the domain max is view)
--   pregnancy edit              → pregnancy view
--   meds / appointments edit    → write (a member write on the owner's record becomes a proposal, F-05)
--   mode_epoch                  owner-data grants (cycle_phase, pregnancy, meds, appointments) get the owner's current
--                               epoch (1 when family_owner_state has no row yet — SyncMode creates it at 1), except the
--                               pregnancy grants of owners not pregnant now: NULL, i.e. inactive until she re-approves.
--                               Teen card sections are not epoch-bound and stay NULL.
--
-- Idempotent: every statement only matches legacy rows. Down maps back (cycle_phase → cycle, write → edit, nudge →
-- view), drops the v6-only domains, clears mode_epoch and the notices; deleted symptoms grants are not restored.

-- +goose Up
INSERT INTO `user_notifications` (user_id, type, title, body, action_url, data, created_at, updated_at)
SELECT o.owner_id, 'companion',
  COALESCE((SELECT JSON_OBJECTAGG(l.code, IF(l.code = 'fa', 'اشتراک «علائم» تمام شد', 'Symptoms sharing has ended'))
            FROM `languages` l WHERE l.is_active = 1),
           JSON_OBJECT('fa', 'اشتراک «علائم» تمام شد', 'en', 'Symptoms sharing has ended')),
  COALESCE((SELECT JSON_OBJECTAGG(l.code, IF(l.code = 'fa', 'همراهت دیگر علائمت را نمی‌بیند.',
              'Your companion no longer sees your symptoms.'))
            FROM `languages` l WHERE l.is_active = 1),
           JSON_OBJECT('fa', 'همراهت دیگر علائمت را نمی‌بیند.', 'en', 'Your companion no longer sees your symptoms.')),
  '/companions', JSON_OBJECT('event', 'symptoms_sharing_ended'), NOW(), NOW()
FROM (SELECT DISTINCT c.owner_id FROM `companion_grants` g JOIN `companions` c ON c.id = g.companion_id
      WHERE g.section = 'symptoms' AND g.level IN ('view', 'edit') AND c.status <> 'revoked') o
WHERE NOT EXISTS (SELECT 1 FROM `user_notifications` n WHERE n.user_id = o.owner_id AND n.type = 'companion'
                  AND JSON_VALUE(n.data, '$.event') = 'symptoms_sharing_ended');

DELETE FROM `companion_grants` WHERE section = 'symptoms';
DELETE FROM `companion_grants` WHERE level = 'none';

UPDATE `companion_grants` SET section = 'cycle_phase', level = 'view' WHERE section = 'cycle';
UPDATE `companion_grants` SET level = 'view' WHERE section = 'pregnancy' AND level = 'edit';
UPDATE `companion_grants` SET level = 'write' WHERE section IN ('meds', 'appointments') AND level = 'edit';

UPDATE `companion_grants` g JOIN `companions` c ON c.id = g.companion_id
LEFT JOIN `family_owner_state` s ON s.owner_id = c.owner_id
SET g.mode_epoch = COALESCE(s.mode_epoch, 1)
WHERE g.mode_epoch IS NULL AND g.section IN ('cycle_phase', 'meds', 'appointments');

UPDATE `companion_grants` g JOIN `companions` c ON c.id = g.companion_id
LEFT JOIN `family_owner_state` s ON s.owner_id = c.owner_id
SET g.mode_epoch = COALESCE(s.mode_epoch, 1)
WHERE g.mode_epoch IS NULL AND g.section = 'pregnancy'
  AND EXISTS (SELECT 1 FROM `pregnancy_profiles` p WHERE p.user_id = c.owner_id AND p.pregnancy_mode = 1);

-- +goose Down
DELETE FROM `user_notifications` WHERE type = 'companion' AND JSON_VALUE(data, '$.event') = 'symptoms_sharing_ended';
DELETE FROM `companion_grants`
WHERE section NOT IN ('cycle_phase', 'pregnancy', 'meds', 'appointments', 'teen_period_week', 'teen_kit', 'teen_notes')
   OR level = 'off';
UPDATE `companion_grants` SET section = 'cycle' WHERE section = 'cycle_phase';
UPDATE `companion_grants` SET level = 'edit' WHERE level = 'write';
UPDATE `companion_grants` SET level = 'view' WHERE level = 'nudge';
UPDATE `companion_grants` SET mode_epoch = NULL;

-- 00095_reset_stale_pregnancy_grants.sql — family privacy hotfix (T-M9-02, gap report 05 FAM-H3, SPEC v6 §1.7).
--
-- A `pregnancy` grant used to outlive the pregnancy it was given for: after a loss or a birth a member saw the owner's
-- next pregnancy from its first day. From now on companion.ResetModeScopedGrants sets the grant to none at every
-- start and end of a pregnancy; this migration applies the same rule to the grants that already exist.
--
-- A pregnancy grant (companion_grants.section = 'pregnancy'; "none" is no row) is removed when
--   * its owner is not pregnant now (no pregnancy_profiles row with pregnancy_mode = 1), or
--   * it was given (updated_at) before the start of the owner's current pregnancy episode — the latest of: the
--     onboarding of the current profile (onboarding_completed_at, else the profile's created_at), the newest recorded
--     loss (pregnancy_losses.created_at) and the newest birth that closed a pregnancy
--     (postpartum_profiles.pregnancy_closed_at). A grant without updated_at counts as stale.
-- A grant given during the current pregnancy is kept. For every removed grant one companion_audit_logs row
-- (action grants_changed, section pregnancy, actor NULL = the system) is written first, in the same transaction.
-- No owner notice: the inbox would fill with one line per link for a change the owner did not make today.
--
-- Data only: the schema is unchanged, so there is no Laravel twin (make schema-diff stays green). Timestamps are
-- Tehran wall-clock (the session time_zone is never set).

-- +goose Up
INSERT INTO `companion_audit_logs` (`owner_id`, `actor_id`, `companion_id`, `section`, `action`, `created_at`)
SELECT c.`owner_id`, NULL, g.`companion_id`, 'pregnancy', 'grants_changed', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
FROM `companion_grants` g
JOIN `companions` c ON c.`id` = g.`companion_id`
LEFT JOIN (
  SELECT p.`user_id`,
    GREATEST(
      COALESCE(p.`onboarding_completed_at`, p.`created_at`, '1970-01-02 00:00:00'),
      COALESCE((SELECT MAX(l.`created_at`) FROM `pregnancy_losses` l WHERE l.`user_id` = p.`user_id`), '1970-01-02 00:00:00'),
      COALESCE((SELECT MAX(pp.`pregnancy_closed_at`) FROM `postpartum_profiles` pp WHERE pp.`user_id` = p.`user_id`), '1970-01-02 00:00:00')
    ) AS `started_at`
  FROM `pregnancy_profiles` p
  WHERE p.`pregnancy_mode` = 1
) ep ON ep.`user_id` = c.`owner_id`
WHERE g.`section` = 'pregnancy'
  AND (ep.`user_id` IS NULL OR g.`updated_at` IS NULL OR g.`updated_at` < ep.`started_at`)
ORDER BY g.`id`;

DELETE g FROM `companion_grants` g
JOIN `companions` c ON c.`id` = g.`companion_id`
LEFT JOIN (
  SELECT p.`user_id`,
    GREATEST(
      COALESCE(p.`onboarding_completed_at`, p.`created_at`, '1970-01-02 00:00:00'),
      COALESCE((SELECT MAX(l.`created_at`) FROM `pregnancy_losses` l WHERE l.`user_id` = p.`user_id`), '1970-01-02 00:00:00'),
      COALESCE((SELECT MAX(pp.`pregnancy_closed_at`) FROM `postpartum_profiles` pp WHERE pp.`user_id` = p.`user_id`), '1970-01-02 00:00:00')
    ) AS `started_at`
  FROM `pregnancy_profiles` p
  WHERE p.`pregnancy_mode` = 1
) ep ON ep.`user_id` = c.`owner_id`
WHERE g.`section` = 'pregnancy'
  AND (ep.`user_id` IS NULL OR g.`updated_at` IS NULL OR g.`updated_at` < ep.`started_at`);

-- +goose Down
-- No-op on purpose: the removed grants were a privacy leak, and restoring them would re-open a pregnancy to members
-- the owner never chose for it. The audit rows stay (they record what happened).
SELECT 1;

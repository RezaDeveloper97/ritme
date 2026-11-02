-- 00114_pregnancy_v1_alerts_inbox.sql — The legacy v1 pregnancy_alerts rows move into the alerts inbox (T-M10-08c,
-- D-165; DECISIONS.md M-10, M-16; SPEC §5.4.1). Data only (no schema change), so no Laravel twin: Laravel never serves
-- the inbox. 00113 moved the v2 rows (alert_type 'v2:<rule>'); this moves every other row — the ones the retired v1
-- rules wrote before T-M10-08 (Laravel's PregnancyAlertService on prod, Go on beta/stage), so no code path reads
-- pregnancy_alerts any more. The old rows stay in place (legacy_alert_id = the old id).
--
-- v1 alert → preg.* rule (alert_type + trigger_symptoms; every v1 writer is listed in the int test):
--   symptom_based {bleeding|spotting|fluid_leakage|severe_sudden_pain: severity} (incl. the day-log flow alert)
--                                                → preg.critical_symptom (an `info` one — light spotting after week 12,
--                                                  a note today's rules no longer raise — is history: archived)
--   symptom_based {multiple_severe}             → preg.severe_symptoms
--   symptom_based {systolic, diastolic[, severe]} → preg.bp_severe when the row was an emergency, flagged severe or
--                                                  at/above 160/110; else preg.bp_high
--   symptom_based {fasting, post_meal}          → preg.gdm
--   symptom_based {fetal_movement: none|reduced} → preg.fetal_movement
--   symptom_based, no trigger, «Note: Mental Health» → preg.depression
--   missing_data / anything else                → archived under preg.weight_missing — except an emergency, which is
--                                                  never archived: preg.critical_symptom with the state rules below.
-- Level and kind are the rule's (as the engine writes them): every v1 emergency lands on a follow_soon rule, so it stays
-- an emergency in the v1 list and the summary; no row is lowered.
--
-- States (the v1 read = acknowledged, dismiss = archived mapping of T-M10-08b; 00113's 30-day window):
--   dismissed, or created more than 30 days ago, or history-only (above) → archived;
--   else per (user, rule) one row is the open row — the unread one first, then the highest v1 level, then the newest —
--   acknowledged when it was read, active when not; the others are resolved history. A rule that already has an
--   open row keeps it and the v1 rows are history, except that an unread v1 emergency is never hidden behind an
--   acknowledged / noted row: that row is resolved (its state kept in evidence.v1_superseded for Down) and the v1
--   emergency is the active open row.
-- A locked rule whose message_rules row was deleted gets its code-held id (inboxalerts.CodeRuleID, pinned by the int
-- test). Idempotent: a row already moved is skipped. Down removes the moved rows and restores a superseded open row.
-- The rows go in staged (state 'v1_staged', not open) and are settled in place, all in the migration's transaction.

-- +goose Up
INSERT INTO `message_inbox` (user_id, rule_id, version, rule_key, slot, kind, level, state, evidence_key, hit_count,
  evidence, first_at, last_at, acked_at, noted_at, resolved_at, archived_at, legacy_alert_id, created_at, updated_at)
SELECT m.user_id,
  COALESCE(r.id, CASE m.rule_key WHEN 'preg.critical_symptom' THEN 9000000000003 WHEN 'preg.fetal_movement' THEN 9000000000007
    WHEN 'preg.bp_severe' THEN 9000000000011 END),
  1, m.rule_key, 'alerts_inbox', COALESCE(r.kind, 'safety'), COALESCE(r.level, 'follow_soon'),
  IF(m.is_dismissed = 1 OR m.history = 1 OR m.created_at < NOW() - INTERVAL 30 DAY, 'archived', 'v1_staged'),
  NULL, 1, m.evidence, m.created_at, m.created_at,
  CASE WHEN m.is_dismissed = 1 THEN COALESCE(m.read_at, m.dismissed_at, m.created_at)
    WHEN m.is_read = 1 THEN COALESCE(m.read_at, m.updated_at, m.created_at) END,
  NULL, NULL,
  IF(m.is_dismissed = 1 OR m.history = 1 OR m.created_at < NOW() - INTERVAL 30 DAY, COALESCE(m.dismissed_at, NOW()), NULL),
  m.id, NOW(), NOW()
FROM (
  SELECT x.id, x.user_id, x.is_read, x.is_dismissed, x.read_at, x.dismissed_at, x.created_at, x.updated_at,
    CASE x.v1
      WHEN 'critical' THEN 'preg.critical_symptom'
      WHEN 'multiple' THEN 'preg.severe_symptoms'
      WHEN 'bp' THEN IF(x.alert_level = 'emergency' OR JSON_VALUE(x.t, '$.severe') IN ('1', 'true')
          OR CAST(JSON_VALUE(x.t, '$.systolic') AS DECIMAL(8,2)) >= 160 OR CAST(JSON_VALUE(x.t, '$.diastolic') AS DECIMAL(8,2)) >= 110,
        'preg.bp_severe', 'preg.bp_high')
      WHEN 'sugar' THEN 'preg.gdm'
      WHEN 'fetal' THEN 'preg.fetal_movement'
      WHEN 'depression' THEN 'preg.depression'
      ELSE IF(x.alert_level = 'emergency', 'preg.critical_symptom', 'preg.weight_missing')
    END AS rule_key,
    CASE
      WHEN x.v1 = 'critical' AND x.alert_level = 'info' THEN 1
      WHEN x.v1 = 'other' AND x.alert_level <> 'emergency' THEN 1
      ELSE 0
    END AS history,
    JSON_OBJECT('vars', JSON_MERGE_PATCH(
      JSON_OBJECT('preg_week', x.pregnancy_week),
      CASE x.v1
        WHEN 'critical' THEN JSON_OBJECT('preg_symptoms', JSON_ARRAY(x.sym), 'preg_symptom_on', DATE_FORMAT(x.created_at, '%Y-%m-%d'))
        WHEN 'multiple' THEN JSON_OBJECT('preg_severe_count', GREATEST(3, COALESCE((
          SELECT (s.nausea_severity <=> 'severe') + (s.vomiting_severity <=> 'severe') + (s.fatigue_severity <=> 'severe')
            + (s.headache_severity <=> 'severe') + (s.dizziness_severity <=> 'severe') + (s.breast_pain_severity <=> 'severe')
            + (s.lower_abdominal_pain_severity <=> 'severe') + (s.cramping_severity <=> 'severe')
            + (s.back_pain_severity <=> 'severe') + (s.pelvic_pressure_severity <=> 'severe')
          FROM `pregnancy_symptom_logs` s WHERE s.user_id = x.user_id AND s.log_date = DATE(x.created_at) LIMIT 1), 0)))
        WHEN 'bp' THEN JSON_OBJECT('preg_bp_sys', COALESCE(JSON_VALUE(x.t, '$.systolic'), '—'),
          'preg_bp_dia', COALESCE(JSON_VALUE(x.t, '$.diastolic'), '—'), 'preg_bp_on', DATE_FORMAT(x.created_at, '%Y-%m-%d'))
        WHEN 'sugar' THEN JSON_OBJECT('preg_fasting', COALESCE(JSON_VALUE(x.t, '$.fasting'), '—'),
          'preg_post_meal', COALESCE(JSON_VALUE(x.t, '$.post_meal'), '—'), 'preg_gdm_on', DATE_FORMAT(x.created_at, '%Y-%m-%d'))
        WHEN 'fetal' THEN JSON_OBJECT('preg_fetal_status', JSON_VALUE(x.t, '$.fetal_movement'),
          'preg_fetal_on', DATE_FORMAT(x.created_at, '%Y-%m-%d'))
        WHEN 'depression' THEN JSON_OBJECT('preg_mood_on', DATE_FORMAT(x.created_at, '%Y-%m-%d'))
        ELSE IF(x.alert_level = 'emergency',
          JSON_OBJECT('preg_symptoms', JSON_ARRAY(), 'preg_symptom_on', DATE_FORMAT(x.created_at, '%Y-%m-%d')), JSON_OBJECT())
      END)) AS evidence
  FROM (
    SELECT a.id, a.user_id, a.alert_level, a.pregnancy_week, a.is_read, a.is_dismissed, a.read_at, a.dismissed_at,
      a.created_at, a.updated_at, COALESCE(a.trigger_symptoms, '{}') AS t,
      CASE
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.bleeding') THEN 'bleeding'
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.fluid_leakage') THEN 'fluid_leakage'
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.severe_sudden_pain') THEN 'severe_sudden_pain'
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.spotting') THEN 'spotting'
      END AS sym,
      CASE
        WHEN a.alert_type <> 'symptom_based' THEN 'other'
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.bleeding', '$.spotting', '$.fluid_leakage', '$.severe_sudden_pain') THEN 'critical'
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.multiple_severe') THEN 'multiple'
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.systolic', '$.diastolic') THEN 'bp'
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.fasting', '$.post_meal') THEN 'sugar'
        WHEN JSON_CONTAINS_PATH(COALESCE(a.trigger_symptoms, '{}'), 'one', '$.fetal_movement') THEN 'fetal'
        WHEN COALESCE(JSON_LENGTH(a.trigger_symptoms), 0) = 0 AND a.title IN ('توجه: وضعیت روحی', 'Note: Mental Health') THEN 'depression'
        ELSE 'other'
      END AS v1
    FROM `pregnancy_alerts` a
    WHERE a.alert_type NOT LIKE 'v2:%'
      AND NOT EXISTS (SELECT 1 FROM `message_inbox` d WHERE d.legacy_alert_id = a.id)
  ) x
) m
LEFT JOIN `message_rules` r ON r.rule_key = m.rule_key
WHERE r.id IS NOT NULL OR m.rule_key IN ('preg.critical_symptom', 'preg.fetal_movement', 'preg.bp_severe');

-- An unread v1 emergency is never hidden behind an acknowledged / noted open row of its rule: that row becomes history.
UPDATE `message_inbox` o
JOIN (
  SELECT DISTINCT s.user_id, s.rule_id
  FROM `message_inbox` s JOIN `pregnancy_alerts` a ON a.id = s.legacy_alert_id
  WHERE s.state = 'v1_staged' AND a.is_read = 0 AND a.alert_level = 'emergency'
) e ON e.user_id = o.user_id AND e.rule_id = o.rule_id
SET o.evidence = JSON_SET(COALESCE(o.evidence, JSON_OBJECT()), '$.v1_superseded', o.state),
  o.state = 'resolved', o.resolved_at = NOW(), o.updated_at = NOW()
WHERE o.open_flag = 1 AND o.state <> 'active';

-- Settle the staged rows: per (user, rule) the unread, most severe, newest row is the open row when the rule has none.
UPDATE `message_inbox` m
JOIN (
  SELECT s.id, a.is_read,
    ROW_NUMBER() OVER (PARTITION BY s.user_id, s.rule_id
      ORDER BY a.is_read ASC, FIELD(a.alert_level, 'emergency', 'warning', 'info') ASC, a.created_at DESC, a.id DESC) AS rn,
    EXISTS (SELECT 1 FROM `message_inbox` o WHERE o.user_id = s.user_id AND o.rule_id = s.rule_id AND o.open_flag = 1) AS taken
  FROM `message_inbox` s JOIN `pregnancy_alerts` a ON a.id = s.legacy_alert_id
  WHERE s.state = 'v1_staged'
) t ON t.id = m.id
SET m.state = IF(t.rn = 1 AND t.taken = 0, IF(t.is_read = 1, 'acknowledged', 'active'), 'resolved'),
  m.resolved_at = IF(t.rn = 1 AND t.taken = 0, NULL, NOW());

-- +goose Down
DELETE m FROM `message_inbox` m JOIN `pregnancy_alerts` a ON a.id = m.legacy_alert_id WHERE a.alert_type NOT LIKE 'v2:%';

UPDATE `message_inbox` o
LEFT JOIN (SELECT DISTINCT user_id, rule_id FROM `message_inbox` WHERE open_flag = 1) t
  ON t.user_id = o.user_id AND t.rule_id = o.rule_id
SET o.state = IF(t.user_id IS NULL, JSON_VALUE(o.evidence, '$.v1_superseded'), o.state),
  o.resolved_at = IF(t.user_id IS NULL, NULL, o.resolved_at),
  o.evidence = JSON_REMOVE(o.evidence, '$.v1_superseded')
WHERE JSON_EXISTS(o.evidence, '$.v1_superseded');

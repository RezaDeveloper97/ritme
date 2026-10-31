-- Message events v6 (T-M10-06, D-163): the buffered writer's multi-row insert, the server-matched goals, the
-- inbox counts of the feed, the daily rollup (M-13), the retention sweep and the admin CTR reads.

-- name: InsertMessageEventsBatch :exec
-- One multi-row INSERT of a JSON array of rows [rule_id, user_id, version, variant, slot, event, created_at] (the
-- events writer flushes in chunks of at most 100 rows; the index below reaches 999). sqlc's parser has neither
-- JSON_TABLE nor recursive CTEs, so the array is walked by an index 0…999 built from three digit tables.
INSERT INTO `message_events` (rule_id, user_id, version, variant, slot, event, created_at)
SELECT CAST(JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][0]')) AS UNSIGNED),
  CAST(JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][1]')) AS UNSIGNED),
  NULLIF(CAST(JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][2]')) AS UNSIGNED), 0),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][3]')),
  NULLIF(JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][4]')), ''),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][5]')),
  CAST(JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][6]')) AS DATETIME)
FROM (
  SELECT d0.d + 10 * d1.d + 100 * d2.d AS i
  FROM (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
    UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) AS d0
  CROSS JOIN (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
    UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) AS d1
  CROSS JOIN (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
    UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) AS d2
) AS n
WHERE n.i < JSON_LENGTH(sqlc.arg(batch))
ORDER BY n.i;

-- name: ListUserGoalCandidates :many
-- The goal match of M-17: the user's shown (impression) and goal rows since the 7-day window on the rules whose
-- goal_event is the domain event, newest first. The caller keeps, per rule, the latest shown row and counts a goal
-- when no goal row followed it.
SELECT e.rule_id, e.event, COALESCE(e.version, r.version) AS version, e.variant,
  COALESCE(e.slot, r.placement) AS slot, e.created_at
FROM `message_events` e
JOIN `message_rules` r ON r.id = e.rule_id
WHERE e.user_id = sqlc.arg(user_id) AND e.created_at >= sqlc.arg(since) AND e.event IN ('impression', 'goal')
  AND r.goal_event = sqlc.arg(goal_event)
ORDER BY e.rule_id, e.created_at DESC, e.id DESC;

-- name: CountUserOpenInbox :one
-- The feed's alerts badge: the user's open inbox rows and the urgent ones among them (SPEC §5.4.1).
SELECT CAST(COUNT(*) AS SIGNED) AS open_count,
  CAST(COALESCE(SUM(level IN ('follow_soon', 'urgent')), 0) AS SIGNED) AS urgent_count
FROM `message_inbox`
WHERE user_id = sqlc.arg(user_id) AND open_flag = 1;

-- name: RollupMessageEvents :exec
-- messages.rollup (M-13): the raw rows of [from, to) counted per (Tehran day, rule, version, variant, slot, event)
-- into message_event_daily. The upsert overwrites, so a re-run gives the same counts. A legacy row without a version
-- counts as version 0; without a slot, under the rule's placement.
INSERT INTO `message_event_daily` (`day`, rule_id, version, variant, slot, event, `count`)
SELECT DATE(e.created_at), e.rule_id, COALESCE(e.version, 0), e.variant, COALESCE(e.slot, r.placement, ''), e.event,
  COUNT(*)
FROM `message_events` e
LEFT JOIN `message_rules` r ON r.id = e.rule_id
WHERE e.created_at >= sqlc.arg(from_at) AND e.created_at < sqlc.arg(to_at)
GROUP BY DATE(e.created_at), e.rule_id, COALESCE(e.version, 0), e.variant, COALESCE(e.slot, r.placement, ''), e.event
ON DUPLICATE KEY UPDATE `count` = VALUES(`count`);

-- name: BackfillMessageEventDaily :exec
-- messages.prune: before raw rows older than the cutoff go, every (day, …) key they make that the rollup does not
-- hold yet is added; existing rollup rows are kept as they are (a day already pruned in part must not shrink).
INSERT INTO `message_event_daily` (`day`, rule_id, version, variant, slot, event, `count`)
SELECT DATE(e.created_at), e.rule_id, COALESCE(e.version, 0), e.variant, COALESCE(e.slot, r.placement, ''), e.event,
  COUNT(*)
FROM `message_events` e
LEFT JOIN `message_rules` r ON r.id = e.rule_id
WHERE e.created_at < sqlc.arg(before_at)
GROUP BY DATE(e.created_at), e.rule_id, COALESCE(e.version, 0), e.variant, COALESCE(e.slot, r.placement, ''), e.event
ON DUPLICATE KEY UPDATE `count` = `count`;

-- name: DeleteMessageEventsBefore :execrows
-- messages.prune (M-13): one batch of raw rows older than the cutoff, oldest first.
DELETE FROM `message_events` WHERE created_at < sqlc.arg(before_at) ORDER BY created_at LIMIT ?;

-- name: MessageRuleDailyStats :many
-- Admin rules list: rollup counts per rule, version, variant and event of the Tehran days [since_day, until_day).
SELECT rule_id, version, variant, event, CAST(SUM(`count`) AS SIGNED) AS total
FROM `message_event_daily`
WHERE `day` >= sqlc.arg(since_day) AND `day` < sqlc.arg(until_day)
GROUP BY rule_id, version, variant, event;

-- name: MessageRuleRawStats :many
-- Admin rules list: today's raw rows (not rolled up yet) per rule, version, variant and event.
SELECT rule_id, CAST(COALESCE(version, 0) AS UNSIGNED) AS version, variant, event, COUNT(*) AS total
FROM `message_events`
WHERE created_at >= sqlc.arg(since_at)
GROUP BY rule_id, COALESCE(version, 0), variant, event;

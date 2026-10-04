-- AI usage + cost log (B-N6-05, goose 00032): one row per provider call. No content, no PII.

-- name: InsertUsage :exec
INSERT INTO `ai_usage_logs` (user_id, feature, op, provider, model, input_tokens, output_tokens, audio_bytes,
  image_bytes, cost_micros, latency_ms, ok, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SumCostSince :one
-- The global daily cap: the estimated cost of every call since `since` (start of the Tehran day).
SELECT CAST(COALESCE(SUM(cost_micros), 0) AS UNSIGNED) AS total FROM `ai_usage_logs` WHERE created_at >= ?;

-- name: UsageByDay :many
-- Admin aggregates: per Tehran day in [from, to).
SELECT CAST(DATE(created_at) AS CHAR(10)) AS day, COUNT(*) AS calls,
  CAST(COALESCE(SUM(ok), 0) AS UNSIGNED) AS ok_calls,
  CAST(COALESCE(SUM(input_tokens), 0) AS UNSIGNED) AS input_tokens,
  CAST(COALESCE(SUM(output_tokens), 0) AS UNSIGNED) AS output_tokens,
  CAST(COALESCE(SUM(cost_micros), 0) AS UNSIGNED) AS cost_micros
FROM `ai_usage_logs` WHERE created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
GROUP BY DATE(created_at) ORDER BY day;

-- name: UsageByFeature :many
SELECT feature, provider, model, COUNT(*) AS calls,
  CAST(COALESCE(SUM(ok), 0) AS UNSIGNED) AS ok_calls,
  CAST(COALESCE(SUM(input_tokens), 0) AS UNSIGNED) AS input_tokens,
  CAST(COALESCE(SUM(output_tokens), 0) AS UNSIGNED) AS output_tokens,
  CAST(COALESCE(SUM(audio_bytes), 0) AS UNSIGNED) AS audio_bytes,
  CAST(COALESCE(SUM(image_bytes), 0) AS UNSIGNED) AS image_bytes,
  CAST(COALESCE(SUM(cost_micros), 0) AS UNSIGNED) AS cost_micros,
  CAST(COALESCE(AVG(latency_ms), 0) AS UNSIGNED) AS avg_latency_ms,
  COUNT(DISTINCT user_id) AS users
FROM `ai_usage_logs` WHERE created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
GROUP BY feature, provider, model ORDER BY cost_micros DESC, feature, provider, model;

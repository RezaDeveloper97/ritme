-- Admin «اعتبار هوش مصنوعی» module (goose 00125, docs/ai-credits.md, admin-api.md §29): the model catalogue
-- (ai_models), the feature slots (ai_features), the credit packs (ai_credit_packs) and a user's credits (read only —
-- every credit write goes through internal/ai/credits.Service). Admin-only: nothing here is reachable from /api/v1.

-- ---------------------------------------------------------------------------
-- Models

-- name: AdminListAiModels :many
SELECT sqlc.embed(ai_models),
       (SELECT COUNT(*) FROM ai_features f
          WHERE f.model_id = ai_models.id OR f.fallback_model_id = ai_models.id) AS slots_count
FROM ai_models
ORDER BY ai_models.provider, ai_models.title, ai_models.id;

-- name: AdminGetAiModel :one
SELECT * FROM ai_models WHERE id = ?;

-- name: AdminAiModelSlots :many
-- The feature slots that point at the model (as primary or fallback).
SELECT `key` FROM ai_features WHERE model_id = sqlc.arg(id) OR fallback_model_id = sqlc.arg(id) ORDER BY sort_order, id;

-- name: AdminAiModelCodeTaken :one
SELECT EXISTS(SELECT 1 FROM ai_models WHERE code = sqlc.arg(code) AND id <> sqlc.arg(except_id)) AS taken;

-- name: AdminAiModelUpstreamTaken :one
SELECT EXISTS(SELECT 1 FROM ai_models WHERE provider = sqlc.arg(provider) AND upstream_model = sqlc.arg(upstream_model)
  AND id <> sqlc.arg(except_id)) AS taken;

-- name: AdminAiModelCodes :many
SELECT code FROM ai_models;

-- name: AdminAiModelUpstreams :many
SELECT upstream_model FROM ai_models WHERE provider = ?;

-- name: AdminCreateAiModel :execlastid
INSERT INTO ai_models (code, provider, upstream_model, title, capabilities, input_price_milli, output_price_milli,
  audio_price_milli, is_active, created_at, updated_at)
VALUES (sqlc.arg(code), sqlc.arg(provider), sqlc.arg(upstream_model), sqlc.arg(title), sqlc.arg(capabilities),
  sqlc.arg(input_price_milli), sqlc.arg(output_price_milli), sqlc.arg(audio_price_milli), sqlc.arg(is_active),
  sqlc.arg(now), sqlc.arg(now));

-- name: AdminInsertAiModelIfMissing :execrows
-- The gateway sync: a model the catalogue does not know yet, inactive until the admin prices it.
INSERT IGNORE INTO ai_models (code, provider, upstream_model, title, capabilities, input_price_milli,
  output_price_milli, audio_price_milli, is_active, created_at, updated_at)
VALUES (sqlc.arg(code), sqlc.arg(provider), sqlc.arg(upstream_model), sqlc.arg(title), sqlc.arg(capabilities), 0, 0, 0,
  0, sqlc.arg(now), sqlc.arg(now));

-- name: AdminUpdateAiModel :exec
UPDATE ai_models
SET provider = sqlc.arg(provider), upstream_model = sqlc.arg(upstream_model), title = sqlc.arg(title),
    capabilities = sqlc.arg(capabilities), input_price_milli = sqlc.arg(input_price_milli),
    output_price_milli = sqlc.arg(output_price_milli), audio_price_milli = sqlc.arg(audio_price_milli),
    is_active = sqlc.arg(is_active), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: AdminDeleteUnusedAiModel :execrows
-- Deletes only while no feature slot points at the model.
DELETE FROM ai_models
WHERE ai_models.id = sqlc.arg(id)
  AND NOT EXISTS (SELECT 1 FROM ai_features f WHERE f.model_id = ai_models.id OR f.fallback_model_id = ai_models.id);

-- ---------------------------------------------------------------------------
-- Feature slots

-- name: AdminListAiFeatures :many
SELECT * FROM ai_features ORDER BY sort_order, id;

-- name: AdminGetAiFeature :one
SELECT * FROM ai_features WHERE id = ?;

-- name: AdminUpdateAiFeature :exec
UPDATE ai_features
SET title = sqlc.arg(title), model_id = sqlc.narg(model_id), fallback_model_id = sqlc.narg(fallback_model_id),
    credit_cost = sqlc.arg(credit_cost), is_active = sqlc.arg(is_active), sort_order = sqlc.arg(sort_order),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- ---------------------------------------------------------------------------
-- Credit packs

-- name: AdminListAiCreditPacks :many
SELECT sqlc.embed(ai_credit_packs),
       (SELECT COUNT(*) FROM plus_invoices i WHERE i.credit_pack_id = ai_credit_packs.id) AS invoices_count,
       (SELECT COUNT(*) FROM plus_invoices ip WHERE ip.credit_pack_id = ai_credit_packs.id AND ip.status = 'paid') AS paid_count
FROM ai_credit_packs
ORDER BY ai_credit_packs.sort_order, ai_credit_packs.id;

-- name: AdminGetAiCreditPack :one
SELECT * FROM ai_credit_packs WHERE id = ?;

-- name: AdminAiCreditPackRefs :one
SELECT (SELECT COUNT(*) FROM plus_invoices i WHERE i.credit_pack_id = sqlc.arg(pack_id)) AS invoices_count,
       (SELECT COUNT(*) FROM plus_invoices ip WHERE ip.credit_pack_id = sqlc.arg(pack_id) AND ip.status = 'paid') AS paid_count;

-- name: AdminAiCreditPackCodeTaken :one
SELECT EXISTS(SELECT 1 FROM ai_credit_packs WHERE code = sqlc.arg(code) AND id <> sqlc.arg(except_id)) AS taken;

-- name: AdminNextAiCreditPackSortOrder :one
SELECT CAST(COALESCE(MAX(sort_order), 0) + 1 AS UNSIGNED) AS next_sort FROM ai_credit_packs;

-- name: AdminCreateAiCreditPack :execlastid
INSERT INTO ai_credit_packs (code, title, badge, credits, price_rials, is_highlighted, is_active, sort_order,
  created_at, updated_at)
VALUES (sqlc.arg(code), sqlc.arg(title), sqlc.narg(badge), sqlc.arg(credits), sqlc.arg(price_rials),
  sqlc.arg(is_highlighted), sqlc.arg(is_active), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: AdminUpdateAiCreditPack :exec
UPDATE ai_credit_packs
SET title = sqlc.arg(title), badge = sqlc.narg(badge), credits = sqlc.arg(credits), price_rials = sqlc.arg(price_rials),
    is_highlighted = sqlc.arg(is_highlighted), is_active = sqlc.arg(is_active), sort_order = sqlc.arg(sort_order),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: AdminDeactivateAiCreditPack :exec
UPDATE ai_credit_packs SET is_active = 0, is_highlighted = 0, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: AdminDeleteUnusedAiCreditPack :execrows
-- Deletes only while no invoice references the pack (invoices keep credit_pack_id for the history).
DELETE FROM ai_credit_packs
WHERE ai_credit_packs.id = sqlc.arg(id)
  AND NOT EXISTS (SELECT 1 FROM plus_invoices i WHERE i.credit_pack_id = ai_credit_packs.id);

-- ---------------------------------------------------------------------------
-- A user's credits (read only)

-- name: AdminAiCreditUserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE id = ?) AS found;

-- name: AdminUserAiCreditGrants :many
-- Every bucket of the user, newest first, with the period that bounds a plan / trial bucket.
SELECT g.id, g.kind, g.subscription_id, g.trial_id, g.invoice_id, g.amount, g.remaining, g.note, g.created_at,
       s.status AS subscription_status, s.starts_at AS subscription_starts_at, s.ends_at AS subscription_ends_at,
       t.started_at AS trial_started_at, t.ends_at AS trial_ends_at
FROM ai_credit_grants g
LEFT JOIN plus_subscriptions s ON s.id = g.subscription_id
LEFT JOIN plus_trials t ON t.id = g.trial_id
WHERE g.user_id = ?
ORDER BY g.id DESC
LIMIT 100;

-- name: AdminUserAiCreditSpends :many
SELECT id, feature, credits, plan_credits, extra_credits, status, created_at
FROM ai_credit_spends
WHERE user_id = ?
ORDER BY id DESC
LIMIT 30;

-- name: AdminUserAiCreditAdjustments :many
SELECT a.id, a.admin_id, a.note, a.details, a.created_at, ad.name AS admin_name
FROM plus_admin_actions a
LEFT JOIN admins ad ON ad.id = a.admin_id
WHERE a.action = 'credits.adjust' AND a.user_id = ?
ORDER BY a.id DESC
LIMIT 20;

-- Admin «اشتراک‌ها و پرداخت» module (B-N2-09): plans, discount codes, subscriptions, the payment log, refunds,
-- extensions and the plus_admin_actions ledger. Admin-only: these read every user's rows by design (the admin chain
-- guards them); nothing here is reachable from /api/v1.

-- ---------------------------------------------------------------------------
-- Plans

-- name: AdminListPlusPlans :many
SELECT sqlc.embed(plus_plans),
       (SELECT COUNT(*) FROM plus_invoices i WHERE i.plan_id = plus_plans.id) AS invoices_count,
       (SELECT COUNT(*) FROM plus_subscriptions s WHERE s.plan_id = plus_plans.id) AS subscriptions_count,
       (SELECT COUNT(*) FROM plus_subscriptions sa WHERE sa.plan_id = plus_plans.id
          AND sa.status IN ('active', 'canceled') AND sa.ends_at > sqlc.arg(now)) AS active_subscriptions
FROM plus_plans
ORDER BY plus_plans.sort_order, plus_plans.id;

-- name: AdminGetPlusPlan :one
SELECT * FROM plus_plans WHERE id = ?;

-- name: AdminPlusPlanRefs :one
SELECT (SELECT COUNT(*) FROM plus_invoices i WHERE i.plan_id = sqlc.arg(plan_id)) AS invoices_count,
       (SELECT COUNT(*) FROM plus_subscriptions s WHERE s.plan_id = sqlc.arg(plan_id)) AS subscriptions_count;

-- name: AdminPlusPlanCodeTaken :one
SELECT EXISTS(SELECT 1 FROM plus_plans WHERE code = sqlc.arg(code) AND id <> sqlc.arg(except_id)) AS taken;

-- name: AdminNextPlusPlanSortOrder :one
SELECT CAST(COALESCE(MAX(sort_order), 0) + 1 AS UNSIGNED) AS next_sort FROM plus_plans;

-- name: AdminCreatePlusPlan :execlastid
INSERT INTO plus_plans (code, title, badge, duration_months, price_rials, monthly_display_rials, is_highlighted,
  is_active, sort_order, created_at, updated_at)
VALUES (sqlc.arg(code), sqlc.arg(title), sqlc.narg(badge), sqlc.arg(duration_months), sqlc.arg(price_rials),
  sqlc.narg(monthly_display_rials), sqlc.arg(is_highlighted), sqlc.arg(is_active), sqlc.arg(sort_order),
  sqlc.arg(now), sqlc.arg(now));

-- name: AdminUpdatePlusPlan :exec
UPDATE plus_plans
SET title = sqlc.arg(title), badge = sqlc.narg(badge), duration_months = sqlc.arg(duration_months),
    price_rials = sqlc.arg(price_rials), monthly_display_rials = sqlc.narg(monthly_display_rials),
    is_highlighted = sqlc.arg(is_highlighted), is_active = sqlc.arg(is_active), sort_order = sqlc.arg(sort_order),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: AdminDeactivatePlusPlan :exec
UPDATE plus_plans SET is_active = 0, is_highlighted = 0, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: AdminDeleteUnusedPlusPlan :execrows
-- Deletes only while nothing references the plan (invoices / subscriptions keep plan_id for the history).
DELETE FROM plus_plans
WHERE plus_plans.id = sqlc.arg(id)
  AND NOT EXISTS (SELECT 1 FROM plus_invoices i WHERE i.plan_id = plus_plans.id)
  AND NOT EXISTS (SELECT 1 FROM plus_subscriptions s WHERE s.plan_id = plus_plans.id);

-- name: AdminPlusPlanOptions :many
SELECT id, code, title, duration_months, is_active FROM plus_plans ORDER BY sort_order, id;

-- ---------------------------------------------------------------------------
-- Discount codes

-- name: AdminCountPlusDiscounts :one
SELECT COUNT(*) FROM plus_discount_codes d
WHERE d.code LIKE sqlc.arg(pattern)
  AND d.is_active >= sqlc.arg(active_min) AND d.is_active <= sqlc.arg(active_max);

-- name: AdminListPlusDiscounts :many
SELECT sqlc.embed(d),
       (SELECT COUNT(*) FROM plus_invoices i WHERE i.discount_code_id = d.id AND i.status = 'paid') AS paid_uses,
       (SELECT COUNT(*) FROM plus_invoices ip WHERE ip.discount_code_id = d.id AND ip.status = 'pending'
          AND ip.expires_at > sqlc.arg(now)) AS pending_uses
FROM plus_discount_codes d
WHERE d.code LIKE sqlc.arg(pattern)
  AND d.is_active >= sqlc.arg(active_min) AND d.is_active <= sqlc.arg(active_max)
ORDER BY d.id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetPlusDiscount :one
SELECT * FROM plus_discount_codes WHERE id = ?;

-- name: AdminPlusDiscountUses :one
SELECT (SELECT COUNT(*) FROM plus_invoices i WHERE i.discount_code_id = sqlc.arg(id) AND i.status = 'paid') AS paid_uses,
       (SELECT COUNT(*) FROM plus_invoices ip WHERE ip.discount_code_id = sqlc.arg(id) AND ip.status = 'pending'
          AND ip.expires_at > sqlc.arg(now)) AS pending_uses,
       (SELECT COUNT(*) FROM plus_invoices ia WHERE ia.discount_code_id = sqlc.arg(id)) AS invoices_count;

-- name: AdminPlusDiscountCodeTaken :one
SELECT EXISTS(SELECT 1 FROM plus_discount_codes WHERE code = sqlc.arg(code) AND id <> sqlc.arg(except_id)) AS taken;

-- name: AdminCreatePlusDiscount :execlastid
INSERT INTO plus_discount_codes (code, kind, value, max_redemptions, per_user_limit, plan_ids, starts_at, expires_at,
  is_active, created_at, updated_at)
VALUES (sqlc.arg(code), sqlc.arg(kind), sqlc.arg(value), sqlc.narg(max_redemptions), sqlc.narg(per_user_limit),
  sqlc.narg(plan_ids), sqlc.narg(starts_at), sqlc.narg(expires_at), sqlc.arg(is_active), sqlc.arg(now), sqlc.arg(now));

-- name: AdminUpdatePlusDiscount :exec
UPDATE plus_discount_codes
SET kind = sqlc.arg(kind), value = sqlc.arg(value), max_redemptions = sqlc.narg(max_redemptions),
    per_user_limit = sqlc.narg(per_user_limit), plan_ids = sqlc.narg(plan_ids), starts_at = sqlc.narg(starts_at),
    expires_at = sqlc.narg(expires_at), is_active = sqlc.arg(is_active), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: AdminDeactivatePlusDiscount :exec
UPDATE plus_discount_codes SET is_active = 0, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: AdminDeleteUnusedPlusDiscount :execrows
DELETE FROM plus_discount_codes
WHERE plus_discount_codes.id = sqlc.arg(id)
  AND NOT EXISTS (SELECT 1 FROM plus_invoices i WHERE i.discount_code_id = plus_discount_codes.id);

-- ---------------------------------------------------------------------------
-- Subscriptions
--
-- Status filter (resolved in Go): status_a / status_b are the stored statuses to match, ends_after < ends_at <=
-- ends_until the window ("active" = stored active and ends_at > now; "expired" = active|canceled with ends_at <= now).

-- name: AdminCountPlusSubscriptions :one
SELECT COUNT(*) FROM plus_subscriptions s
JOIN users u ON u.id = s.user_id
WHERE (s.status LIKE sqlc.arg(status_a) OR s.status LIKE sqlc.arg(status_b))
  AND s.ends_at > sqlc.arg(ends_after) AND s.ends_at <= sqlc.arg(ends_until)
  AND IFNULL(s.plan_id, 0) >= sqlc.arg(plan_min) AND IFNULL(s.plan_id, 0) <= sqlc.arg(plan_max)
  AND s.starts_at >= sqlc.arg(starts_from) AND s.starts_at < sqlc.arg(starts_to)
  AND IFNULL(u.mobile, '') LIKE sqlc.arg(mobile);

-- name: AdminListPlusSubscriptions :many
SELECT s.id, s.user_id, s.plan_id, s.invoice_id, s.status, s.source, s.starts_at, s.ends_at, s.auto_renew,
       s.canceled_at, s.created_at, u.name AS user_name, u.mobile AS user_mobile,
       p.code AS plan_code, p.title AS plan_title, i.reference AS invoice_reference
FROM plus_subscriptions s
JOIN users u ON u.id = s.user_id
LEFT JOIN plus_plans p ON p.id = s.plan_id
LEFT JOIN plus_invoices i ON i.id = s.invoice_id
WHERE (s.status LIKE sqlc.arg(status_a) OR s.status LIKE sqlc.arg(status_b))
  AND s.ends_at > sqlc.arg(ends_after) AND s.ends_at <= sqlc.arg(ends_until)
  AND IFNULL(s.plan_id, 0) >= sqlc.arg(plan_min) AND IFNULL(s.plan_id, 0) <= sqlc.arg(plan_max)
  AND s.starts_at >= sqlc.arg(starts_from) AND s.starts_at < sqlc.arg(starts_to)
  AND IFNULL(u.mobile, '') LIKE sqlc.arg(mobile)
ORDER BY s.id DESC
LIMIT ? OFFSET ?;

-- name: AdminPlusSubscriptionCounts :one
SELECT
  (SELECT COUNT(*) FROM plus_subscriptions a WHERE a.status = 'active' AND a.ends_at > sqlc.arg(now)) AS active,
  (SELECT COUNT(*) FROM plus_subscriptions c WHERE c.status = 'canceled' AND c.ends_at > sqlc.arg(now)) AS canceled,
  (SELECT COUNT(*) FROM plus_subscriptions e WHERE e.status IN ('active', 'canceled') AND e.ends_at <= sqlc.arg(now)) AS expired,
  (SELECT COUNT(*) FROM plus_subscriptions r WHERE r.status = 'refunded') AS refunded;

-- name: AdminLockPlusSubscription :one
SELECT * FROM plus_subscriptions WHERE id = ? FOR UPDATE;

-- name: AdminSetPlusSubscriptionEnd :exec
UPDATE plus_subscriptions SET ends_at = sqlc.arg(ends_at), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: AdminShiftQueuedPlusSubscriptions :execrows
-- An extension pushes the user's queued periods (starting at or after the extended period's old end) back by the
-- same number of days, so periods never overlap.
UPDATE plus_subscriptions
SET starts_at = DATE_ADD(starts_at, INTERVAL sqlc.arg(days) DAY),
    ends_at = DATE_ADD(ends_at, INTERVAL sqlc.arg(days) DAY),
    updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id)
  AND id <> sqlc.arg(id)
  AND status IN ('active', 'canceled')
  AND starts_at >= sqlc.arg(old_end);

-- name: AdminRefundPlusSubscriptionsOfInvoice :execrows
UPDATE plus_subscriptions
SET status = 'refunded', auto_renew = 0, updated_at = sqlc.arg(now)
WHERE invoice_id = sqlc.arg(invoice_id) AND status IN ('active', 'canceled');

-- ---------------------------------------------------------------------------
-- Payment log (invoices + receipts). The receipt's card_pan is never selected.

-- name: AdminCountPlusInvoices :one
SELECT COUNT(*) FROM plus_invoices i
JOIN users u ON u.id = i.user_id
LEFT JOIN plus_receipts r ON r.invoice_id = i.id
WHERE i.status LIKE sqlc.arg(status)
  AND IFNULL(i.gateway, '') LIKE sqlc.arg(gateway)
  AND i.created_at >= sqlc.arg(created_from) AND i.created_at < sqlc.arg(created_to)
  AND (IFNULL(i.reference, '') LIKE sqlc.arg(q) OR IFNULL(r.ref_id, '') LIKE sqlc.arg(q_ref) OR IFNULL(u.mobile, '') LIKE sqlc.arg(q_mobile));

-- name: AdminPlusInvoiceSummary :one
SELECT CAST(COALESCE(SUM(CASE WHEN i.status = 'paid' THEN i.total_rials ELSE 0 END), 0) AS UNSIGNED) AS paid_rials,
       CAST(COALESCE(SUM(CASE WHEN i.status = 'refunded' THEN i.total_rials ELSE 0 END), 0) AS UNSIGNED) AS refunded_rials,
       CAST(COALESCE(SUM(CASE WHEN i.status = 'paid' THEN 1 ELSE 0 END), 0) AS UNSIGNED) AS paid_count
FROM plus_invoices i
JOIN users u ON u.id = i.user_id
LEFT JOIN plus_receipts r ON r.invoice_id = i.id
WHERE i.status LIKE sqlc.arg(status)
  AND IFNULL(i.gateway, '') LIKE sqlc.arg(gateway)
  AND i.created_at >= sqlc.arg(created_from) AND i.created_at < sqlc.arg(created_to)
  AND (IFNULL(i.reference, '') LIKE sqlc.arg(q) OR IFNULL(r.ref_id, '') LIKE sqlc.arg(q_ref) OR IFNULL(u.mobile, '') LIKE sqlc.arg(q_mobile));

-- name: AdminListPlusInvoices :many
SELECT i.id, i.reference, i.user_id, i.plan_id, i.duration_months, i.status, i.subtotal_rials, i.discount_rials,
       i.vat_rate_bps, i.vat_rials, i.total_rials, i.discount_code, i.gateway, i.expires_at, i.paid_at, i.created_at,
       u.name AS user_name, u.mobile AS user_mobile, p.code AS plan_code, p.title AS plan_title,
       r.ref_id AS receipt_ref_id, r.amount_rials AS receipt_amount_rials, r.paid_at AS receipt_paid_at
FROM plus_invoices i
JOIN users u ON u.id = i.user_id
LEFT JOIN plus_plans p ON p.id = i.plan_id
LEFT JOIN plus_receipts r ON r.invoice_id = i.id
WHERE i.status LIKE sqlc.arg(status)
  AND IFNULL(i.gateway, '') LIKE sqlc.arg(gateway)
  AND i.created_at >= sqlc.arg(created_from) AND i.created_at < sqlc.arg(created_to)
  AND (IFNULL(i.reference, '') LIKE sqlc.arg(q) OR IFNULL(r.ref_id, '') LIKE sqlc.arg(q_ref) OR IFNULL(u.mobile, '') LIKE sqlc.arg(q_mobile))
ORDER BY i.id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetPlusInvoice :one
SELECT i.id, i.reference, i.user_id, i.plan_id, i.duration_months, i.status, i.currency, i.subtotal_rials,
       i.discount_rials, i.vat_rate_bps, i.vat_rials, i.total_rials, i.discount_code, i.gateway, i.authority,
       i.expires_at, i.paid_at, i.created_at, i.updated_at,
       u.name AS user_name, u.mobile AS user_mobile, p.code AS plan_code, p.title AS plan_title,
       r.ref_id AS receipt_ref_id, r.gateway AS receipt_gateway, r.amount_rials AS receipt_amount_rials,
       r.paid_at AS receipt_paid_at
FROM plus_invoices i
JOIN users u ON u.id = i.user_id
LEFT JOIN plus_plans p ON p.id = i.plan_id
LEFT JOIN plus_receipts r ON r.invoice_id = i.id
WHERE i.id = ?;

-- name: AdminLockPlusInvoice :one
SELECT * FROM plus_invoices WHERE id = ? FOR UPDATE;

-- name: AdminPlusInvoiceReceipt :one
SELECT id, gateway, ref_id, amount_rials, paid_at FROM plus_receipts WHERE invoice_id = ?;

-- name: AdminMarkPlusInvoiceRefunded :execrows
UPDATE plus_invoices SET status = 'refunded', updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND status = 'paid';

-- name: AdminPlusInvoiceSubscriptions :many
SELECT id, status, source, starts_at, ends_at FROM plus_subscriptions WHERE invoice_id = ? ORDER BY id;

-- name: AdminPlusGateways :many
SELECT DISTINCT gateway FROM plus_invoices WHERE gateway IS NOT NULL ORDER BY gateway;

-- ---------------------------------------------------------------------------
-- Ledger

-- name: InsertPlusAdminAction :exec
INSERT INTO plus_admin_actions (admin_id, action, target_type, target_id, user_id, amount_rials, days, gateway,
  gateway_ref, note, details, created_at, updated_at)
VALUES (sqlc.narg(admin_id), sqlc.arg(action), sqlc.arg(target_type), sqlc.arg(target_id), sqlc.narg(user_id),
  sqlc.narg(amount_rials), sqlc.narg(days), sqlc.narg(gateway), sqlc.narg(gateway_ref), sqlc.narg(note),
  sqlc.narg(details), sqlc.arg(now), sqlc.arg(now));

-- name: ListPlusAdminActionsFor :many
SELECT a.id, a.admin_id, a.action, a.target_type, a.target_id, a.user_id, a.amount_rials, a.days, a.gateway,
       a.gateway_ref, a.note, a.details, a.created_at, ad.name AS admin_name
FROM plus_admin_actions a
LEFT JOIN admins ad ON ad.id = a.admin_id
WHERE a.target_type = sqlc.arg(target_type) AND a.target_id = sqlc.arg(target_id)
ORDER BY a.id DESC
LIMIT 50;

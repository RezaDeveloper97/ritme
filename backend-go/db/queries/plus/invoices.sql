-- Checkout invoices and receipts. Every user-facing read is scoped by user_id (IDOR).

-- name: LockDiscountCode :one
-- Serialises concurrent checkouts on one code so the redemption limits cannot be overrun.
SELECT * FROM plus_discount_codes
WHERE code = ?
FOR UPDATE;

-- name: CountDiscountRedemptions :one
-- Paid invoices, plus pending ones still holding a reservation (not expired).
SELECT COUNT(*) FROM plus_invoices
WHERE discount_code_id = sqlc.arg(discount_code_id)
  AND (status = 'paid' OR (status = 'pending' AND expires_at > sqlc.arg(now)));

-- name: CountUserDiscountRedemptions :one
SELECT COUNT(*) FROM plus_invoices
WHERE discount_code_id = sqlc.arg(discount_code_id)
  AND user_id = sqlc.arg(user_id)
  AND (status = 'paid' OR (status = 'pending' AND expires_at > sqlc.arg(now)));

-- name: InsertInvoice :execlastid
INSERT INTO plus_invoices (reference, user_id, plan_id, duration_months, status, currency, subtotal_rials, discount_rials,
  vat_rate_bps, vat_rials, total_rials, discount_code_id, discount_code, gateway, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, 'pending', 'IRR', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetInvoiceAuthority :exec
UPDATE plus_invoices SET authority = ?, updated_at = ?
WHERE id = ? AND status = 'pending';

-- name: GetUserInvoiceByReference :one
SELECT * FROM plus_invoices
WHERE reference = ? AND user_id = ?;

-- name: LockUserInvoiceByReference :one
SELECT * FROM plus_invoices
WHERE reference = ? AND user_id = ?
FOR UPDATE;

-- name: MarkInvoicePaid :exec
UPDATE plus_invoices SET status = 'paid', paid_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: MarkInvoiceFailed :exec
UPDATE plus_invoices SET status = 'failed', updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: MarkInvoicePaidFree :exec
-- A 100 % discount: nothing to charge, paid at checkout.
UPDATE plus_invoices SET status = 'paid', gateway = NULL, paid_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: ListUserPendingInvoices :many
-- Restore: pending checkouts that reached a gateway, newest first.
SELECT * FROM plus_invoices
WHERE user_id = ? AND status = 'pending' AND authority IS NOT NULL
ORDER BY id DESC
LIMIT 10;

-- name: ListUserInvoices :many
SELECT sqlc.embed(plus_invoices),
       r.ref_id AS receipt_ref_id, r.card_pan AS receipt_card_pan, r.amount_rials AS receipt_amount_rials,
       r.paid_at AS receipt_paid_at
FROM plus_invoices
LEFT JOIN plus_receipts r ON r.invoice_id = plus_invoices.id
WHERE plus_invoices.user_id = ?
ORDER BY plus_invoices.id DESC
LIMIT 50;

-- name: GetInvoiceReceipt :one
SELECT * FROM plus_receipts
WHERE invoice_id = ? AND user_id = ?;

-- name: InsertReceipt :exec
-- UNIQUE(gateway, ref_id): a bank reference pays one invoice only (replay guard).
INSERT INTO plus_receipts (invoice_id, user_id, gateway, ref_id, card_pan, amount_rials, paid_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

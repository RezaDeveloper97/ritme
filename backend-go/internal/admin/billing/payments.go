package billing

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/plus"
)

// Refund modes and error codes.
const (
	RefundGateway = "gateway" // ask the payment provider to send the money back
	RefundManual  = "manual"  // the money was returned outside Ritme (gateway panel, bank transfer): record it

	CodeNotRefundable      = "invoice_not_refundable" // not paid (pending / failed / already refunded) or nothing was charged
	CodeRefundNotSupported = "refund_not_supported"   // the provider cannot refund through Ritme → refund in its panel, then mark manually
	CodeRefundRejected     = "refund_rejected"        // the provider refused (already refunded there, amount, …)
	CodeRefundFailed       = "refund_failed"          // transport / provider failure; nothing changed
)

var invoiceStatuses = []string{plus.InvoicePending, plus.InvoicePaid, plus.InvoiceFailed, plus.InvoiceExpired, plus.InvoiceRefunded}

// ListPayments is GET /plus/payments?status=all|pending|paid|failed|expired|refunded&gateway=&from=&to=&q= — the
// payment log: invoices (newest first) with their verified receipt. q matches the invoice reference, the bank
// reference or part of the mobile number. Card numbers are never returned. summary = paid / refunded totals of
// the filtered rows.
func (h *Handlers) ListPayments(c fiber.Ctx) error {
	status := c.Query("status")
	statusPattern := "%"
	for _, s := range invoiceStatuses {
		if s == status {
			statusPattern = s
		}
	}
	if statusPattern == "%" {
		status = SubAll
	}
	gateway := strings.TrimSpace(c.Query("gateway"))
	gwPattern := form.Exact(gateway)
	lo, hi, from, to := dateRange(c)
	search := strings.TrimSpace(digits(c.Query("q")))
	q := form.Contains(search)
	created := func() (sql.NullTime, sql.NullTime) {
		return sql.NullTime{Time: lo, Valid: true}, sql.NullTime{Time: hi, Valid: true}
	}
	cf, ct := created()
	total, err := h.q.AdminCountPlusInvoices(c.Context(), store.AdminCountPlusInvoicesParams{
		Status: statusPattern, Gateway: sql.NullString{String: gwPattern, Valid: true}, CreatedFrom: cf, CreatedTo: ct,
		Q: q, QRef: q, QMobile: sql.NullString{String: q, Valid: true},
	})
	if err != nil {
		return err
	}
	summary, err := h.q.AdminPlusInvoiceSummary(c.Context(), store.AdminPlusInvoiceSummaryParams{
		Status: statusPattern, Gateway: sql.NullString{String: gwPattern, Valid: true}, CreatedFrom: cf, CreatedTo: ct,
		Q: q, QRef: q, QMobile: sql.NullString{String: q, Valid: true},
	})
	if err != nil {
		return err
	}
	pg := httpadmin.PageOf(c)
	rows, err := h.q.AdminListPlusInvoices(c.Context(), store.AdminListPlusInvoicesParams{
		Status: statusPattern, Gateway: sql.NullString{String: gwPattern, Valid: true}, CreatedFrom: cf, CreatedTo: ct,
		Q: q, QRef: q, QMobile: sql.NullString{String: q, Valid: true},
		Limit: int32(pg.PerPage), Offset: int32(min(pg.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	gateways, err := h.q.AdminPlusGateways(c.Context())
	if err != nil {
		return err
	}
	at := now(c)
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		var receipt any
		if r.ReceiptRefID.Valid {
			receipt = jsonx.Obj("ref_id", r.ReceiptRefID.String, "amount_rials", nullInt64(r.ReceiptAmountRials),
				"paid_at", httpadmin.Time(r.ReceiptPaidAt))
		}
		effective := r.Status
		if r.Status == plus.InvoicePending && !r.ExpiresAt.After(at) {
			effective = plus.InvoiceExpired
		}
		items = append(items, jsonx.Obj(
			"id", r.ID,
			"reference", r.Reference,
			"user", userJSON(r.UserID, r.UserName, r.UserMobile),
			"plan", planRef(r.PlanID, r.PlanCode, r.PlanTitle),
			"duration_months", r.DurationMonths,
			"status", r.Status,
			"effective_status", effective,
			"subtotal_rials", r.SubtotalRials,
			"discount_rials", r.DiscountRials,
			"vat_rate_bps", r.VatRateBps,
			"vat_rials", r.VatRials,
			"total_rials", r.TotalRials,
			"discount_code", httpadmin.NullString(r.DiscountCode),
			"gateway", httpadmin.NullString(r.Gateway),
			"receipt", receipt,
			"paid_at", httpadmin.Time(r.PaidAt),
			"created_at", httpadmin.Time(r.CreatedAt),
		))
	}
	gw := make([]string, 0, len(gateways))
	for _, g := range gateways {
		if g.Valid {
			gw = append(gw, g.String)
		}
	}
	page := httpadmin.Page(items, pg, int(total))
	page.Set("filters", jsonx.Obj("status", status, "gateway", gateway, "from", from, "to", to, "q", search))
	page.Set("summary", jsonx.Obj("paid_count", summary.PaidCount, "paid_rials", summary.PaidRials,
		"refunded_rials", summary.RefundedRials))
	page.Set("gateways", gw)
	page.Set("statuses", invoiceStatuses)
	return httpadmin.OK(c, page)
}

func (h *Handlers) paymentJSON(c fiber.Ctx, id uint64) (*jsonx.OrderedMap, error) {
	r, err := h.q.AdminGetPlusInvoice(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpadmin.NotFound("Payment")
	}
	if err != nil {
		return nil, err
	}
	subs, err := h.q.AdminPlusInvoiceSubscriptions(c.Context(), nullU64(id))
	if err != nil {
		return nil, err
	}
	subList := make([]any, 0, len(subs))
	for _, s := range subs {
		subList = append(subList, jsonx.Obj("id", s.ID, "status", s.Status, "source", s.Source,
			"starts_at", iso(s.StartsAt), "ends_at", iso(s.EndsAt)))
	}
	actions, err := h.actionsFor(c, "invoice", id)
	if err != nil {
		return nil, err
	}
	var receipt any
	if r.ReceiptRefID.Valid {
		receipt = jsonx.Obj("gateway", httpadmin.NullString(r.ReceiptGateway), "ref_id", r.ReceiptRefID.String,
			"amount_rials", nullInt64(r.ReceiptAmountRials), "paid_at", httpadmin.Time(r.ReceiptPaidAt))
	}
	gatewayRefund := h.refunder != nil && r.ReceiptGateway.Valid && r.ReceiptGateway.String == h.refunder.Name()
	return jsonx.Obj("payment", jsonx.Obj(
		"id", r.ID,
		"reference", r.Reference,
		"user", userJSON(r.UserID, r.UserName, r.UserMobile),
		"plan", planRef(r.PlanID, r.PlanCode, r.PlanTitle),
		"duration_months", r.DurationMonths,
		"status", r.Status,
		"currency", r.Currency,
		"subtotal_rials", r.SubtotalRials,
		"discount_rials", r.DiscountRials,
		"vat_rate_bps", r.VatRateBps,
		"vat_rials", r.VatRials,
		"total_rials", r.TotalRials,
		"discount_code", httpadmin.NullString(r.DiscountCode),
		"gateway", httpadmin.NullString(r.Gateway),
		"authority", httpadmin.NullString(r.Authority),
		"receipt", receipt,
		"expires_at", iso(r.ExpiresAt),
		"paid_at", httpadmin.Time(r.PaidAt),
		"created_at", httpadmin.Time(r.CreatedAt),
		"updated_at", httpadmin.Time(r.UpdatedAt),
		"subscriptions", subList,
		"refund", jsonx.Obj(
			"refundable", r.Status == plus.InvoicePaid,
			"gateway_available", gatewayRefund && r.TotalRials > 0,
			"amount_rials", r.TotalRials,
		),
		"actions", actions,
	)), nil
}

// ShowPayment is GET /plus/payments/:id: the invoice, its receipt (no card data), the subscription periods it
// bought and the admin actions on it.
func (h *Handlers) ShowPayment(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Payment")
	}
	body, err := h.paymentJSON(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, body)
}

// RefundPayment is POST /plus/payments/:id/refund {mode: gateway|manual, note} (super). Full refunds only:
//   - gateway: the payment provider refunds the receipt amount (payments.Gateway.Refund). A provider that cannot
//     refund through Ritme (Zarinpal) answers 422 refund_not_supported — refund in its panel, then mark manually.
//   - manual: records a refund made outside Ritme; the note (how / where) is required.
//
// Either way the invoice becomes refunded and the subscription period it bought is revoked (status refunded), in
// one transaction holding the invoice row lock (a second refund sees refunded → 422 invoice_not_refundable).
func (h *Handlers) RefundPayment(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Payment")
	}
	in := validation.Input(c)
	noteRule := "nullable|string|max:500"
	if m, _ := in.Get("mode"); m == RefundManual {
		noteRule = "required|string|min:3|max:500"
	}
	data, err := form.ValidateInput(c, in, validation.Rules{
		validation.F("mode", "required", validation.In(RefundGateway, RefundManual)),
		validation.F("note", noteRule),
	})
	if err != nil {
		return err
	}
	mode := httpadmin.String(data, "mode")
	note := strings.TrimSpace(httpadmin.String(data, "note"))
	stamp := httpadmin.DBTime(httpadmin.Now(c))
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		inv, err := q.AdminLockPlusInvoice(c.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			return httpadmin.NotFound("Payment")
		}
		if err != nil {
			return err
		}
		if inv.Status != plus.InvoicePaid {
			return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeNotRefundable, "Only a paid invoice can be refunded.")
		}
		rec := action{name: "invoice.refund_manual", targetType: "invoice", targetID: inv.ID, userID: inv.UserID,
			amount: inv.TotalRials, note: note, details: jsonx.Obj("reference", inv.Reference)}
		if mode == RefundGateway {
			if err := h.gatewayRefund(c, q, &inv, note, &rec); err != nil {
				return err
			}
		}
		n, err := q.AdminMarkPlusInvoiceRefunded(c.Context(), store.AdminMarkPlusInvoiceRefundedParams{Now: stamp, ID: inv.ID})
		if err != nil {
			return err
		}
		if n == 0 {
			return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeNotRefundable, "Only a paid invoice can be refunded.")
		}
		revoked, err := q.AdminRefundPlusSubscriptionsOfInvoice(c.Context(), store.AdminRefundPlusSubscriptionsOfInvoiceParams{
			Now: stamp, InvoiceID: nullU64(inv.ID),
		})
		if err != nil {
			return err
		}
		rec.details.(*jsonx.OrderedMap).Set("subscriptions_revoked", revoked)
		return h.record(c, q, rec)
	})
	if err != nil {
		return err
	}
	body, err := h.paymentJSON(c, id)
	if err != nil {
		return err
	}
	if mode == RefundManual {
		return httpadmin.OK(c, body, "Marked as refunded.")
	}
	return httpadmin.OK(c, body, "Refund sent through the gateway.")
}

// gatewayRefund asks the provider to refund inv's receipt in full and fills the ledger row. Runs inside the
// refund transaction (the invoice row is locked), so two admins cannot refund the same payment twice.
func (h *Handlers) gatewayRefund(c fiber.Ctx, q *store.Queries, inv *store.PlusInvoice, note string, rec *action) error {
	unsupported := func() error {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeRefundNotSupported,
			"This payment cannot be refunded through the gateway from Ritme. Refund it in the gateway's panel, then mark it refunded manually.")
	}
	if inv.TotalRials == 0 || !inv.Authority.Valid {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeNotRefundable,
			"Nothing was charged for this invoice; mark it refunded manually to revoke the subscription.")
	}
	receipt, err := q.AdminPlusInvoiceReceipt(c.Context(), inv.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return unsupported()
	}
	if err != nil {
		return err
	}
	if h.refunder == nil || receipt.Gateway != h.refunder.Name() {
		return unsupported()
	}
	res, err := h.refunder.Refund(c.Context(), payments.RefundRequest{
		Authority: inv.Authority.String, RefID: receipt.RefID, AmountRials: receipt.AmountRials, Reason: note,
	})
	switch {
	case errors.Is(err, payments.ErrRefundNotSupported):
		return unsupported()
	case errors.Is(err, payments.ErrRefundRejected), errors.Is(err, payments.ErrInvalidRequest):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeRefundRejected, "The gateway refused the refund.")
	case err != nil:
		h.logger.WarnContext(c.Context(), "plus admin: gateway refund failed", "invoice_id", inv.ID, "gateway", receipt.Gateway, "error", err.Error())
		return httpadmin.Fail(fiber.StatusBadGateway, CodeRefundFailed, "The gateway did not answer; nothing was changed. Try again.")
	}
	rec.name, rec.gateway, rec.ref = "invoice.refund", receipt.Gateway, res.RefundID
	if res.AmountRials > 0 {
		rec.amount = res.AmountRials
	}
	return nil
}

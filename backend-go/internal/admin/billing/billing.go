// Package billing is the admin «اشتراک‌ها و پرداخت» module (B-N2-09, docs/go-migration/admin-api.md §15) under
// /api/admin/v1/plus/*: Ritme Plus plans and discount codes (CRUD; a referenced row is deactivated instead of
// deleted), the trial-offer percent and the VAT override (plus_settings), the subscriptions list, the payment log
// (invoices + receipts, never card data), refunds (through the payment gateway, or marked manually with a note when
// the provider cannot refund) and subscription extensions.
//
// Roles: every read is open to any active admin (rows carry a masked mobile only, like the Admin_Users artboard);
// every write — plan prices, discount codes, settings, refunds, extensions — needs a super admin (all of them change
// what users pay or get). Writes go through the admin CSRF check and each one writes a plus_admin_actions row (the
// durable money trail, with the admin's note) besides the slog "admin audit" line.
package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/plus"
)

// Refunder is the payment-gateway port refunds use (*payments.Gateway satisfies it).
type Refunder interface {
	Name() string
	Refund(ctx context.Context, req payments.RefundRequest) (payments.RefundResult, error)
}

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Handlers serve /plus/*.
type Handlers struct {
	db       *sql.DB
	q        *store.Queries
	svc      *plus.Service
	refunder Refunder // nil = no payment provider configured (refunds can only be marked manually)
	logger   *slog.Logger
}

// New wires the module. svc is the Plus domain (settings); refunder may be nil.
func New(sqlDB *sql.DB, svc *plus.Service, refunder Refunder, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{db: sqlDB, q: store.New(sqlDB), svc: svc, refunder: refunder, logger: logger}
}

// Routes mounts the module. Reads: any active admin. Writes: super admin. Static paths before :id.
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	a, s := kit.Admin, kit.Super
	get, post, put, del := fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete

	route(get, "/plus/plans", a(h.ListPlans))
	route(post, "/plus/plans", s(h.StorePlan))
	route(get, "/plus/plans/:id", a(h.ShowPlan))
	route(put, "/plus/plans/:id", s(h.UpdatePlan))
	route(del, "/plus/plans/:id", s(h.DestroyPlan))

	route(get, "/plus/discount-codes", a(h.ListDiscounts))
	route(post, "/plus/discount-codes", s(h.StoreDiscount))
	route(get, "/plus/discount-codes/:id", a(h.ShowDiscount))
	route(put, "/plus/discount-codes/:id", s(h.UpdateDiscount))
	route(del, "/plus/discount-codes/:id", s(h.DestroyDiscount))

	route(get, "/plus/settings", a(h.ShowSettings))
	route(put, "/plus/settings", s(h.UpdateSettings))

	route(get, "/plus/subscriptions", a(h.ListSubscriptions))
	route(post, "/plus/subscriptions/:id/extend", s(h.ExtendSubscription))

	route(get, "/plus/payments", a(h.ListPayments))
	route(get, "/plus/payments/:id", a(h.ShowPayment))
	route(post, "/plus/payments/:id/refund", s(h.RefundPayment))
}

// ---------------------------------------------------------------------------
// Shared helpers

// now is the request time as plus_* rows store it: Tehran wall-clock, whole seconds.
func now(c fiber.Ctx) time.Time {
	return httpadmin.Now(c).In(civildate.Tehran).Truncate(time.Second)
}

// MaskMobile hides the middle of a mobile number the way the admin user list shows it: 0912•••6789. Numbers too
// short to keep 4 + 4 digits are fully hidden.
func MaskMobile(m sql.NullString) any {
	if !m.Valid || m.String == "" {
		return nil
	}
	r := []rune(m.String)
	if len(r) < 9 {
		return "•••"
	}
	return string(r[:4]) + "•••" + string(r[len(r)-4:])
}

// userJSON is the short subscriber object of the lists: id, name and the masked mobile.
func userJSON(id uint64, name, mobile sql.NullString) *jsonx.OrderedMap {
	return jsonx.Obj("id", id, "name", httpadmin.NullString(name), "mobile", MaskMobile(mobile))
}

// planRef is {id, code, title} or null when the plan was deleted.
func planRef(id sql.NullInt64, code sql.NullString, title db.NullRawJSON) any {
	if !id.Valid || !code.Valid {
		return nil
	}
	return jsonx.Obj("id", id.Int64, "code", code.String, "title", form.NullRaw(title))
}

// iso is a datetime column as ISO 8601 (+03:30).
func iso(t time.Time) any { return httpadmin.Time(sql.NullTime{Time: t, Valid: true}) }

func nullU64(v uint64) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(v), Valid: true} //nolint:gosec // G115: ids / rials fit in int64
}

func nullInt64(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func nullInt32(n sql.NullInt32) any {
	if !n.Valid {
		return nil
	}
	return n.Int32
}

// action is one plus_admin_actions row.
type action struct {
	name, targetType string
	targetID         uint64
	userID           uint64
	amount           uint64
	days             int
	gateway, ref     string
	note             string
	details          any
}

// record writes the ledger row (through q, so it joins the caller's transaction) and the slog audit line.
func (h *Handlers) record(c fiber.Ctx, q *store.Queries, a action) error {
	p := store.InsertPlusAdminActionParams{
		Action: a.name, TargetType: a.targetType, TargetID: a.targetID,
		Gateway:    sql.NullString{String: a.gateway, Valid: a.gateway != ""},
		GatewayRef: sql.NullString{String: a.ref, Valid: a.ref != ""},
		Note:       sql.NullString{String: truncate(a.note, 500), Valid: a.note != ""},
		Now:        httpadmin.DBTime(httpadmin.Now(c)),
	}
	if adm := httpadmin.CurrentAdmin(c); adm != nil {
		p.AdminID = nullU64(adm.ID)
	}
	if a.userID > 0 {
		p.UserID = nullU64(a.userID)
	}
	if a.amount > 0 {
		p.AmountRials = nullU64(a.amount)
	}
	if a.days > 0 {
		p.Days = sql.NullInt32{Int32: int32(a.days), Valid: true} //nolint:gosec // G115: validated ≤ MaxExtendDays
	}
	if a.details != nil {
		p.Details = db.NullRawJSON{V: form.JSON(a.details), Valid: true}
	}
	if err := q.InsertPlusAdminAction(c.Context(), p); err != nil {
		return err
	}
	attrs := []slog.Attr{}
	if a.userID > 0 {
		attrs = append(attrs, slog.Uint64("user_id", a.userID))
	}
	if a.amount > 0 {
		attrs = append(attrs, slog.Uint64("amount_rials", a.amount))
	}
	httpadmin.Audit(c, h.logger, "plus."+a.name, a.targetType, a.targetID, attrs...)
	return nil
}

// actionsJSON renders ledger rows for a detail page.
func actionsJSON(rows []store.ListPlusAdminActionsForRow) []any {
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		var details any
		if r.Details.Valid {
			details = form.Raw(r.Details.V)
		}
		out = append(out, jsonx.Obj(
			"id", r.ID,
			"action", r.Action,
			"admin", jsonx.Obj("id", nullInt64(r.AdminID), "name", httpadmin.NullString(r.AdminName)),
			"amount_rials", nullInt64(r.AmountRials),
			"days", nullInt32(r.Days),
			"gateway", httpadmin.NullString(r.Gateway),
			"gateway_ref", httpadmin.NullString(r.GatewayRef),
			"note", httpadmin.NullString(r.Note),
			"details", details,
			"created_at", httpadmin.Time(r.CreatedAt),
		))
	}
	return out
}

func (h *Handlers) actionsFor(c fiber.Ctx, targetType string, id uint64) ([]any, error) {
	rows, err := h.q.ListPlusAdminActionsFor(c.Context(), store.ListPlusAdminActionsForParams{TargetType: targetType, TargetID: id})
	if err != nil {
		return nil, err
	}
	return actionsJSON(rows), nil
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// inTx runs fn in a READ COMMITTED transaction (the row locks of refund / extend must see committed data).
func (h *Handlers) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := h.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	if err := fn(h.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// dateRange reads ?from=&to= (Y-m-d, Tehran days, both inclusive) as [from 00:00, to+1 00:00). Missing or invalid
// bounds are open; the applied values are echoed back (empty string = open).
func dateRange(c fiber.Ctx) (lo, hi time.Time, from, to string) {
	lo = time.Date(1970, 1, 2, 0, 0, 0, 0, civildate.Tehran)
	hi = time.Date(9999, 1, 1, 0, 0, 0, 0, civildate.Tehran)
	if d, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(c.Query("from")), civildate.Tehran); err == nil {
		lo, from = d, d.Format("2006-01-02")
	}
	if d, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(c.Query("to")), civildate.Tehran); err == nil {
		hi, to = d.AddDate(0, 0, 1), d.Format("2006-01-02")
	}
	return lo, hi, from, to
}

// rawJSON is a JSON column for output.
func rawJSON(raw json.RawMessage) any { return form.Raw(raw) }

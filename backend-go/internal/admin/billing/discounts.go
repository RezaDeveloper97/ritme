package billing

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/plus"
)

// Discount form limits.
const (
	MaxRedemptions   = 1_000_000
	MaxPerUserLimit  = 1_000
	discountCodeRule = "/^[A-Za-z0-9_-]+$/"
)

// Discount states (derived, for the list badge and the ?status= filter).
const (
	DiscountStateActive    = "active"
	DiscountStateScheduled = "scheduled"
	DiscountStateExpired   = "expired"
	DiscountStateExhausted = "exhausted"
	DiscountStateInactive  = "inactive"
)

func discountState(d *store.PlusDiscountCode, paid, pending int64, at time.Time) string {
	switch {
	case !d.IsActive:
		return DiscountStateInactive
	case d.ExpiresAt.Valid && !at.Before(d.ExpiresAt.Time):
		return DiscountStateExpired
	case d.StartsAt.Valid && at.Before(d.StartsAt.Time):
		return DiscountStateScheduled
	case d.MaxRedemptions.Valid && paid+pending >= int64(d.MaxRedemptions.Int32):
		return DiscountStateExhausted
	}
	return DiscountStateActive
}

func planIDs(col db.NullRawJSON) any {
	if !col.Valid {
		return nil
	}
	var ids []uint64
	if err := json.Unmarshal(col.V, &ids); err != nil {
		return []uint64{}
	}
	return ids
}

func discountJSON(d *store.PlusDiscountCode, paid, pending int64, at time.Time) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", d.ID,
		"code", d.Code,
		"kind", d.Kind,
		"value", d.Value,
		"max_redemptions", nullInt32(d.MaxRedemptions),
		"per_user_limit", nullInt32(d.PerUserLimit),
		"plan_ids", planIDs(d.PlanIds),
		"starts_at", httpadmin.Time(d.StartsAt),
		"expires_at", httpadmin.Time(d.ExpiresAt),
		"is_active", d.IsActive,
		"uses", jsonx.Obj("paid", paid, "pending", pending),
		"state", discountState(d, paid, pending, at),
		"created_at", httpadmin.Time(d.CreatedAt),
		"updated_at", httpadmin.Time(d.UpdatedAt),
	)
}

// ListDiscounts is GET /plus/discount-codes?q=&status=all|active|inactive&page=&per_page= (newest first).
// uses.paid = paid invoices with the code; uses.pending = checkouts still holding a reservation.
func (h *Handlers) ListDiscounts(c fiber.Ctx) error {
	search := strings.TrimSpace(c.Query("q"))
	status := c.Query("status")
	var only *bool
	switch status {
	case "active":
		only = new(bool)
		*only = true
	case "inactive":
		only = new(bool)
	default:
		status = "all"
	}
	lo, hi := form.BoolRange(only)
	pattern := form.Contains(search)
	total, err := h.q.AdminCountPlusDiscounts(c.Context(), store.AdminCountPlusDiscountsParams{Pattern: pattern, ActiveMin: lo, ActiveMax: hi})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	at := now(c)
	rows, err := h.q.AdminListPlusDiscounts(c.Context(), store.AdminListPlusDiscountsParams{
		Now: at, Pattern: pattern, ActiveMin: lo, ActiveMax: hi,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, discountJSON(&rows[i].PlusDiscountCode, rows[i].PaidUses, rows[i].PendingUses, at))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("q", search, "status", status))
	return httpadmin.OK(c, page)
}

func (h *Handlers) findDiscount(c fiber.Ctx) (store.PlusDiscountCode, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.PlusDiscountCode{}, httpadmin.NotFound("Discount code")
	}
	d, err := h.q.AdminGetPlusDiscount(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return d, httpadmin.NotFound("Discount code")
	}
	return d, err
}

func (h *Handlers) respondDiscount(c fiber.Ctx, id uint64, created bool, msg string) error {
	d, err := h.q.AdminGetPlusDiscount(c.Context(), id)
	if err != nil {
		return err
	}
	at := now(c)
	uses, err := h.q.AdminPlusDiscountUses(c.Context(), store.AdminPlusDiscountUsesParams{ID: nullU64(id), Now: at})
	if err != nil {
		return err
	}
	body := jsonx.Obj("discount_code", discountJSON(&d, uses.PaidUses, uses.PendingUses, at))
	var msgs []string
	if msg != "" {
		msgs = []string{msg}
	}
	if created {
		return httpadmin.Created(c, body, msgs...)
	}
	return httpadmin.OK(c, body, msgs...)
}

// ShowDiscount is GET /plus/discount-codes/:id.
func (h *Handlers) ShowDiscount(c fiber.Ctx) error {
	d, err := h.findDiscount(c)
	if err != nil {
		return err
	}
	return h.respondDiscount(c, d.ID, false, "")
}

func discountRules(create bool) validation.Rules {
	var r validation.Rules
	if create {
		r = append(r, validation.F("code", "required|string|min:3|max:64", validation.Regex(discountCodeRule)))
	}
	return append(r,
		validation.F("kind", "required", validation.In(plus.DiscountPercent, plus.DiscountAmount)),
		validation.F("value", "required|integer|min:1|max:"+strconv.Itoa(MaxPriceRials)),
		validation.F("max_redemptions", "nullable|integer|min:1|max:"+strconv.Itoa(MaxRedemptions)),
		validation.F("per_user_limit", "nullable|integer|min:1|max:"+strconv.Itoa(MaxPerUserLimit)),
		validation.F("plan_ids", "nullable|array"),
		validation.F("plan_ids.*", "integer|min:1"),
		validation.F("starts_at", "nullable|date"),
		validation.F("expires_at", "nullable|date|after_or_equal:starts_at"),
		validation.F("is_active", "nullable|boolean"),
	)
}

// discountChecks: a percent is 1–100; plan_ids must name existing plans; the code is unique (case-insensitive,
// the column's collation — checkout matches it the same way).
func (h *Handlers) discountChecks(c fiber.Ctx, create bool) form.Check {
	return func(in phpval.Map, add form.Add) error {
		if kind, _ := in.Get("kind"); phpval.ToString(kind) == plus.DiscountPercent {
			if v, ok := in.Get("value"); ok && v != nil && phpval.ToFloat(v) > 100 {
				add("value", form.Msg(c, "validation.max.numeric", "value", "max", "100"))
			}
		}
		if raw, ok := in.Get("plan_ids"); ok && raw != nil {
			if phpval.IsArray(raw) {
				_, vals := phpval.Entries(raw)
				opts, err := h.q.AdminPlusPlanOptions(c.Context())
				if err != nil {
					return err
				}
				known := make(map[uint64]bool, len(opts))
				for _, o := range opts {
					known[o.ID] = true
				}
				for _, v := range vals {
					f := phpval.ToFloat(v)
					if f <= 0 || !known[uint64(f)] {
						add("plan_ids", form.Msg(c, "validation.exists", "plan_ids"))
						break
					}
				}
			}
		}
		if create {
			code, _ := in.Get("code")
			if s, isStr := code.(string); isStr && s != "" {
				taken, err := h.q.AdminPlusDiscountCodeTaken(c.Context(), store.AdminPlusDiscountCodeTakenParams{Code: s, ExceptID: 0})
				if err != nil {
					return err
				}
				if taken {
					add("code", form.Msg(c, "validation.unique", "code"))
				}
			}
		}
		return nil
	}
}

func optInt32(data phpval.Map, key string) sql.NullInt32 {
	if !form.Has(data, key) {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(u64(data, key)), Valid: true} //nolint:gosec // G115: validated limits
}

func planIDsJSON(data phpval.Map) db.NullRawJSON {
	raw, ok := data.Get("plan_ids")
	if !ok || raw == nil {
		return db.NullRawJSON{}
	}
	_, vals := phpval.Entries(raw)
	if !phpval.IsArray(raw) || len(vals) == 0 {
		return db.NullRawJSON{} // an empty list = every plan
	}
	ids := make([]uint64, 0, len(vals))
	seen := map[uint64]bool{}
	for _, v := range vals {
		n := uint64(phpval.ToFloat(v))
		if !seen[n] { // duplicates collapse
			seen[n] = true
			ids = append(ids, n)
		}
	}
	return db.NullRawJSON{V: form.JSON(ids), Valid: true}
}

func timeOf(data phpval.Map, key string) sql.NullTime {
	if t, ok := form.ParseTime(form.Str(data, key).String); ok && form.Has(data, key) {
		return sql.NullTime{Time: t, Valid: true}
	}
	return sql.NullTime{}
}

// StoreDiscount is POST /plus/discount-codes (super). The code is stored upper-case.
func (h *Handlers) StoreDiscount(c fiber.Ctx) error {
	data, err := form.Validate(c, discountRules(true), h.discountChecks(c, true))
	if err != nil {
		return err
	}
	p := store.AdminCreatePlusDiscountParams{
		Code: strings.ToUpper(httpadmin.String(data, "code")), Kind: httpadmin.String(data, "kind"), Value: u64(data, "value"),
		MaxRedemptions: optInt32(data, "max_redemptions"), PerUserLimit: sql.NullInt32{Int32: 1, Valid: true},
		PlanIds: planIDsJSON(data), StartsAt: timeOf(data, "starts_at"), ExpiresAt: timeOf(data, "expires_at"),
		IsActive: boolOr(data, "is_active", true), Now: httpadmin.DBTime(httpadmin.Now(c)),
	}
	if _, sent := data.Get("per_user_limit"); sent {
		p.PerUserLimit = optInt32(data, "per_user_limit") // null = unlimited per user
	}
	var id uint64
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		lastID, err := q.AdminCreatePlusDiscount(c.Context(), p)
		if err != nil {
			return err
		}
		id = uint64(lastID) //nolint:gosec // G115: auto-increment ids are positive
		return h.record(c, q, action{name: "discount.create", targetType: "discount_code", targetID: id,
			details: jsonx.Obj("code", p.Code, "kind", p.Kind, "value", p.Value)})
	})
	if err != nil {
		return err
	}
	return h.respondDiscount(c, id, true, "Discount code created.")
}

// UpdateDiscount is PUT /plus/discount-codes/:id (super). The code itself is fixed (invoices keep a copy of it);
// optional fields that are absent keep their stored value.
func (h *Handlers) UpdateDiscount(c fiber.Ctx) error {
	cur, err := h.findDiscount(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, discountRules(false), h.discountChecks(c, false))
	if err != nil {
		return err
	}
	p := store.AdminUpdatePlusDiscountParams{
		Kind: httpadmin.String(data, "kind"), Value: u64(data, "value"),
		MaxRedemptions: cur.MaxRedemptions, PerUserLimit: cur.PerUserLimit, PlanIds: cur.PlanIds,
		StartsAt: form.KeepTime(data, "starts_at", cur.StartsAt), ExpiresAt: form.KeepTime(data, "expires_at", cur.ExpiresAt),
		IsActive: boolOr(data, "is_active", cur.IsActive), Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}
	if _, sent := data.Get("max_redemptions"); sent {
		p.MaxRedemptions = optInt32(data, "max_redemptions")
	}
	if _, sent := data.Get("per_user_limit"); sent {
		p.PerUserLimit = optInt32(data, "per_user_limit")
	}
	if _, sent := data.Get("plan_ids"); sent {
		p.PlanIds = planIDsJSON(data)
	}
	details := jsonx.Obj()
	if cur.Kind != p.Kind || cur.Value != p.Value {
		details.Set("value", jsonx.Obj("from", cur.Kind+":"+strconv.FormatUint(cur.Value, 10), "to", p.Kind+":"+strconv.FormatUint(p.Value, 10)))
	}
	if cur.IsActive != p.IsActive {
		details.Set("is_active", jsonx.Obj("from", cur.IsActive, "to", p.IsActive))
	}
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		if err := q.AdminUpdatePlusDiscount(c.Context(), p); err != nil {
			return err
		}
		return h.record(c, q, action{name: "discount.update", targetType: "discount_code", targetID: cur.ID, details: details})
	})
	if err != nil {
		return err
	}
	return h.respondDiscount(c, cur.ID, false, "Discount code updated.")
}

// DestroyDiscount is DELETE /plus/discount-codes/:id (super): a code any invoice used is deactivated instead
// (the invoices keep the reference): data.deleted false, data.deactivated true.
func (h *Handlers) DestroyDiscount(c fiber.Ctx) error {
	cur, err := h.findDiscount(c)
	if err != nil {
		return err
	}
	stamp := httpadmin.DBTime(httpadmin.Now(c))
	deleted := false
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		n, err := q.AdminDeleteUnusedPlusDiscount(c.Context(), cur.ID)
		if err != nil {
			return err
		}
		if n > 0 {
			deleted = true
			return h.record(c, q, action{name: "discount.delete", targetType: "discount_code", targetID: cur.ID,
				details: jsonx.Obj("code", cur.Code)})
		}
		if err := q.AdminDeactivatePlusDiscount(c.Context(), store.AdminDeactivatePlusDiscountParams{Now: stamp, ID: cur.ID}); err != nil {
			return err
		}
		return h.record(c, q, action{name: "discount.deactivate", targetType: "discount_code", targetID: cur.ID,
			details: jsonx.Obj("code", cur.Code)})
	})
	if err != nil {
		return err
	}
	if deleted {
		return httpadmin.OK(c, jsonx.Obj("id", cur.ID, "deleted", true, "deactivated", false), "Discount code deleted.")
	}
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID, "deleted", false, "deactivated", true),
		"The code was used on invoices, so it was deactivated instead of deleted.")
}

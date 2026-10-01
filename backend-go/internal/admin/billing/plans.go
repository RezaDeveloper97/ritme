package billing

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Plan form limits (rials). MinPriceRials is the smallest amount the bank gateways accept (Zarinpal: 1,000 toman).
const (
	MinPriceRials     = 10_000
	MaxPriceRials     = 100_000_000_000
	MaxDurationMonths = 36
	MaxSortOrder      = 100_000
)

const planCodePattern = "/^[a-z][a-z0-9_]*$/"

type planCounts struct{ invoices, subscriptions, active int64 }

func monthlyRials(p *store.PlusPlan) uint64 {
	if p.MonthlyDisplayRials.Valid && p.MonthlyDisplayRials.Int64 > 0 {
		return uint64(p.MonthlyDisplayRials.Int64)
	}
	if p.DurationMonths == 0 {
		return p.PriceRials
	}
	return p.PriceRials / uint64(p.DurationMonths)
}

func planJSON(p *store.PlusPlan, n planCounts) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", p.ID,
		"code", p.Code,
		"title", rawJSON(p.Title),
		"badge", form.NullRaw(p.Badge),
		"duration_months", p.DurationMonths,
		"price_rials", p.PriceRials,
		"monthly_display_rials", nullInt64(p.MonthlyDisplayRials),
		"monthly_price_rials", monthlyRials(p),
		"is_highlighted", p.IsHighlighted,
		"is_active", p.IsActive,
		"sort_order", p.SortOrder,
		"invoices_count", n.invoices,
		"subscriptions_count", n.subscriptions,
		"active_subscriptions", n.active,
		"created_at", httpadmin.Time(p.CreatedAt),
		"updated_at", httpadmin.Time(p.UpdatedAt),
	)
}

// ListPlans is GET /plus/plans: every plan (active or not) in display order with its reference counts. Plans are
// few, so the list is not paginated.
func (h *Handlers) ListPlans(c fiber.Ctx) error {
	rows, err := h.q.AdminListPlusPlans(c.Context(), now(c))
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		items = append(items, planJSON(&r.PlusPlan, planCounts{r.InvoicesCount, r.SubscriptionsCount, r.ActiveSubscriptions}))
	}
	next, err := h.q.AdminNextPlusPlanSortOrder(c.Context())
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj(
		"items", jsonx.List(items),
		"currency", "IRR",
		"limits", jsonx.Obj("min_price_rials", MinPriceRials, "max_price_rials", MaxPriceRials,
			"max_duration_months", MaxDurationMonths),
		"next_sort_order", next,
	))
}

func (h *Handlers) findPlan(c fiber.Ctx) (store.PlusPlan, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.PlusPlan{}, httpadmin.NotFound("Plan")
	}
	p, err := h.q.AdminGetPlusPlan(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return p, httpadmin.NotFound("Plan")
	}
	return p, err
}

func (h *Handlers) respondPlan(c fiber.Ctx, id uint64, created bool, msg string) error {
	p, err := h.q.AdminGetPlusPlan(c.Context(), id)
	if err != nil {
		return err
	}
	refs, err := h.q.AdminPlusPlanRefs(c.Context(), store.AdminPlusPlanRefsParams{PlanID: nullU64(id)})
	if err != nil {
		return err
	}
	body := jsonx.Obj("plan", planJSON(&p, planCounts{invoices: refs.InvoicesCount, subscriptions: refs.SubscriptionsCount}))
	var msgs []string
	if msg != "" {
		msgs = []string{msg}
	}
	if created {
		return httpadmin.Created(c, body, msgs...)
	}
	return httpadmin.OK(c, body, msgs...)
}

// ShowPlan is GET /plus/plans/:id.
func (h *Handlers) ShowPlan(c fiber.Ctx) error {
	p, err := h.findPlan(c)
	if err != nil {
		return err
	}
	return h.respondPlan(c, p.ID, false, "")
}

func planRules(c fiber.Ctx, create bool) validation.Rules {
	var r validation.Rules
	if create {
		r = append(r, validation.F("code", "required|string|max:64", validation.Regex(planCodePattern)))
	}
	r = append(r, form.Translatable(c, "title", true, "max:64")...)
	r = append(r, form.Translatable(c, "badge", false, "max:32")...)
	r = append(r,
		validation.F("duration_months", "required|integer|min:1|max:"+strconv.Itoa(MaxDurationMonths)),
		validation.F("price_rials", "required|integer|min:"+strconv.Itoa(MinPriceRials)+"|max:"+strconv.Itoa(MaxPriceRials)),
		validation.F("monthly_display_rials", "nullable|integer|min:1|max:"+strconv.Itoa(MaxPriceRials)),
		validation.F("is_highlighted", "nullable|boolean"),
		validation.F("is_active", "nullable|boolean"),
		validation.F("sort_order", "nullable|integer|min:0|max:"+strconv.Itoa(MaxSortOrder)),
	)
	return r
}

func (h *Handlers) checkPlanCode(c fiber.Ctx) form.Check {
	return func(in phpval.Map, add form.Add) error {
		code, ok := in.Get("code")
		s, isStr := code.(string)
		if !ok || !isStr || s == "" {
			return nil
		}
		taken, err := h.q.AdminPlusPlanCodeTaken(c.Context(), store.AdminPlusPlanCodeTakenParams{Code: s, ExceptID: 0})
		if err != nil {
			return err
		}
		if taken {
			add("code", form.Msg(c, "validation.unique", "code"))
		}
		return nil
	}
}

// translatable is a cleaned translatable value over the active languages (nil when no language has text).
func translatable(data phpval.Map, key string) db.NullRawJSON {
	col := form.NullJSON(data, key)
	if !col.Valid {
		return col
	}
	cleaned := i18n.Clean(col.V)
	if cleaned == nil {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: cleaned, Valid: true}
}

func u64(data phpval.Map, key string) uint64 {
	v, _ := data.Get(key)
	f := phpval.ToFloat(v)
	if f <= 0 {
		return 0
	}
	return uint64(f)
}

func boolOr(data phpval.Map, key string, def bool) bool {
	if !form.Has(data, key) {
		return def
	}
	return httpadmin.Bool(data, key)
}

// StorePlan is POST /plus/plans (super).
func (h *Handlers) StorePlan(c fiber.Ctx) error {
	data, err := form.Validate(c, planRules(c, true), h.checkPlanCode(c))
	if err != nil {
		return err
	}
	sortOrder := uint32(0)
	if form.Has(data, "sort_order") {
		sortOrder = uint32(u64(data, "sort_order")) //nolint:gosec // G115: validated ≤ MaxSortOrder
	} else {
		next, err := h.q.AdminNextPlusPlanSortOrder(c.Context())
		if err != nil {
			return err
		}
		sortOrder = uint32(next) //nolint:gosec // G115: small MAX()+1
	}
	title := translatable(data, "title")
	params := store.AdminCreatePlusPlanParams{
		Code: httpadmin.String(data, "code"), Title: title.V, Badge: translatable(data, "badge"),
		DurationMonths: uint16(u64(data, "duration_months")), //nolint:gosec // G115: validated ≤ 36
		PriceRials:     u64(data, "price_rials"),
		IsHighlighted:  boolOr(data, "is_highlighted", false), IsActive: boolOr(data, "is_active", true),
		SortOrder: sortOrder, Now: httpadmin.DBTime(httpadmin.Now(c)),
	}
	if form.Has(data, "monthly_display_rials") {
		params.MonthlyDisplayRials = nullU64(u64(data, "monthly_display_rials"))
	}
	var id uint64
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		lastID, err := q.AdminCreatePlusPlan(c.Context(), params)
		if err != nil {
			return err
		}
		id = uint64(lastID) //nolint:gosec // G115: auto-increment ids are positive
		return h.record(c, q, action{name: "plan.create", targetType: "plan", targetID: id,
			details: jsonx.Obj("code", params.Code, "price_rials", params.PriceRials, "duration_months", params.DurationMonths)})
	})
	if err != nil {
		return err
	}
	return h.respondPlan(c, id, true, "Plan created.")
}

// UpdatePlan is PUT /plus/plans/:id (super). The code is fixed at creation (it is the plan's stable id in
// invoices and analytics); optional fields that are absent keep their stored value. Existing invoices keep the
// price they were charged — a price change only affects new checkouts.
func (h *Handlers) UpdatePlan(c fiber.Ctx) error {
	cur, err := h.findPlan(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, planRules(c, false))
	if err != nil {
		return err
	}
	p := store.AdminUpdatePlusPlanParams{
		Title: translatable(data, "title").V, Badge: cur.Badge,
		DurationMonths: uint16(u64(data, "duration_months")), //nolint:gosec // G115: validated ≤ 36
		PriceRials:     u64(data, "price_rials"), MonthlyDisplayRials: cur.MonthlyDisplayRials,
		IsHighlighted: boolOr(data, "is_highlighted", cur.IsHighlighted), IsActive: boolOr(data, "is_active", cur.IsActive),
		SortOrder: cur.SortOrder, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}
	if _, sent := data.Get("badge"); sent {
		p.Badge = translatable(data, "badge")
	}
	if _, sent := data.Get("monthly_display_rials"); sent {
		p.MonthlyDisplayRials = sql.NullInt64{}
		if form.Has(data, "monthly_display_rials") {
			p.MonthlyDisplayRials = nullU64(u64(data, "monthly_display_rials"))
		}
	}
	if form.Has(data, "sort_order") {
		p.SortOrder = uint32(u64(data, "sort_order")) //nolint:gosec // G115: validated ≤ MaxSortOrder
	}
	details := jsonx.Obj()
	if cur.PriceRials != p.PriceRials {
		details.Set("price_rials", jsonx.Obj("from", cur.PriceRials, "to", p.PriceRials))
	}
	if cur.DurationMonths != p.DurationMonths {
		details.Set("duration_months", jsonx.Obj("from", cur.DurationMonths, "to", p.DurationMonths))
	}
	if cur.IsActive != p.IsActive {
		details.Set("is_active", jsonx.Obj("from", cur.IsActive, "to", p.IsActive))
	}
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		if err := q.AdminUpdatePlusPlan(c.Context(), p); err != nil {
			return err
		}
		return h.record(c, q, action{name: "plan.update", targetType: "plan", targetID: cur.ID, details: details})
	})
	if err != nil {
		return err
	}
	return h.respondPlan(c, cur.ID, false, "Plan updated.")
}

// DestroyPlan is DELETE /plus/plans/:id (super). A plan that invoices or subscriptions reference is deactivated
// instead (they keep plan_id for the history): data.deleted false, data.deactivated true.
func (h *Handlers) DestroyPlan(c fiber.Ctx) error {
	cur, err := h.findPlan(c)
	if err != nil {
		return err
	}
	stamp := httpadmin.DBTime(httpadmin.Now(c))
	deleted := false
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		n, err := q.AdminDeleteUnusedPlusPlan(c.Context(), cur.ID)
		if err != nil {
			return err
		}
		if n > 0 {
			deleted = true
			return h.record(c, q, action{name: "plan.delete", targetType: "plan", targetID: cur.ID,
				details: jsonx.Obj("code", cur.Code)})
		}
		if err := q.AdminDeactivatePlusPlan(c.Context(), store.AdminDeactivatePlusPlanParams{Now: stamp, ID: cur.ID}); err != nil {
			return err
		}
		return h.record(c, q, action{name: "plan.deactivate", targetType: "plan", targetID: cur.ID,
			details: jsonx.Obj("code", cur.Code)})
	})
	if err != nil {
		return err
	}
	if deleted {
		return httpadmin.OK(c, jsonx.Obj("id", cur.ID, "deleted", true, "deactivated", false), "Plan deleted.")
	}
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID, "deleted", false, "deactivated", true),
		"The plan has invoices or subscriptions, so it was deactivated instead of deleted.")
}

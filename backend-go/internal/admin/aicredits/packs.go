package aicredits

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Credit pack form limits (rials, like plus plans: MinPackPriceRials is the smallest amount the gateways accept).
const (
	MinPackPriceRials = 10_000
	MaxPackPriceRials = 100_000_000_000
	MaxPackCredits    = 1_000_000
)

const packCodePattern = "/^[a-z][a-z0-9_]*$/"

type packCounts struct{ invoices, paid int64 }

func packJSON(p *store.AiCreditPack, n packCounts) *jsonx.OrderedMap {
	var perCredit uint64
	if p.Credits > 0 {
		perCredit = p.PriceRials / uint64(p.Credits)
	}
	return jsonx.Obj(
		"id", p.ID,
		"code", p.Code,
		"title", form.Raw(p.Title),
		"badge", form.NullRaw(p.Badge),
		"credits", p.Credits,
		"price_rials", p.PriceRials,
		"price_per_credit_rials", perCredit,
		"is_highlighted", p.IsHighlighted,
		"is_active", p.IsActive,
		"sort_order", p.SortOrder,
		"invoices_count", n.invoices,
		"paid_count", n.paid,
		"created_at", httpadmin.Time(p.CreatedAt),
		"updated_at", httpadmin.Time(p.UpdatedAt),
	)
}

// ListPacks is GET /ai/credit-packs: every pack (active or not) in display order with its invoice counts. Packs are
// few, so the list is not paginated.
func (h *Handlers) ListPacks(c fiber.Ctx) error {
	rows, err := h.q.AdminListAiCreditPacks(c.Context())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, packJSON(&rows[i].AiCreditPack, packCounts{rows[i].InvoicesCount, rows[i].PaidCount}))
	}
	next, err := h.q.AdminNextAiCreditPackSortOrder(c.Context())
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj(
		"items", jsonx.List(items),
		"currency", "IRR",
		"limits", jsonx.Obj("min_price_rials", MinPackPriceRials, "max_price_rials", MaxPackPriceRials,
			"max_credits", MaxPackCredits),
		"next_sort_order", next,
	))
}

func (h *Handlers) findPack(c fiber.Ctx) (store.AiCreditPack, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.AiCreditPack{}, httpadmin.NotFound("Credit pack")
	}
	p, err := h.q.AdminGetAiCreditPack(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return p, httpadmin.NotFound("Credit pack")
	}
	return p, err
}

func (h *Handlers) respondPack(c fiber.Ctx, id uint64, created bool, msg string) error {
	p, err := h.q.AdminGetAiCreditPack(c.Context(), id)
	if err != nil {
		return err
	}
	refs, err := h.q.AdminAiCreditPackRefs(c.Context(), store.AdminAiCreditPackRefsParams{PackID: nullU64(id)})
	if err != nil {
		return err
	}
	body := jsonx.Obj("pack", packJSON(&p, packCounts{refs.InvoicesCount, refs.PaidCount}))
	if created {
		return httpadmin.Created(c, body, msg)
	}
	if msg == "" {
		return httpadmin.OK(c, body)
	}
	return httpadmin.OK(c, body, msg)
}

// ShowPack is GET /ai/credit-packs/:id.
func (h *Handlers) ShowPack(c fiber.Ctx) error {
	p, err := h.findPack(c)
	if err != nil {
		return err
	}
	return h.respondPack(c, p.ID, false, "")
}

func packRules(c fiber.Ctx, create bool) validation.Rules {
	var r validation.Rules
	if create {
		r = append(r, validation.F("code", "required|string|max:64", validation.Regex(packCodePattern)))
	}
	r = append(r, form.Translatable(c, "title", true, "max:64")...)
	r = append(r, form.Translatable(c, "badge", false, "max:32")...)
	return append(r,
		validation.F("credits", "required|integer|min:1|max:"+strconv.Itoa(MaxPackCredits)),
		validation.F("price_rials", "required|integer|min:"+strconv.Itoa(MinPackPriceRials)+"|max:"+strconv.Itoa(MaxPackPriceRials)),
		validation.F("is_highlighted", "nullable|boolean"),
		validation.F("is_active", "nullable|boolean"),
		validation.F("sort_order", "nullable|integer|min:0|max:"+strconv.Itoa(maxSortOrder)),
	)
}

func (h *Handlers) checkPackCode(c fiber.Ctx) form.Check {
	return func(in phpval.Map, add form.Add) error {
		code, _ := in.Get("code")
		s, isStr := code.(string)
		if !isStr || s == "" {
			return nil
		}
		taken, err := h.q.AdminAiCreditPackCodeTaken(c.Context(), store.AdminAiCreditPackCodeTakenParams{Code: s, ExceptID: 0})
		if err != nil {
			return err
		}
		if taken {
			add("code", form.Msg(c, "validation.unique", "code"))
		}
		return nil
	}
}

func u64(data phpval.Map, key string) uint64 {
	v, _ := data.Get(key)
	f := phpval.ToFloat(v)
	if f <= 0 {
		return 0
	}
	return uint64(f)
}

// StorePack is POST /ai/credit-packs (super).
func (h *Handlers) StorePack(c fiber.Ctx) error {
	data, err := form.Validate(c, packRules(c, true), h.checkPackCode(c))
	if err != nil {
		return err
	}
	sortOrder := uint32(0)
	if form.Has(data, "sort_order") {
		sortOrder = uint32(u64(data, "sort_order")) //nolint:gosec // G115: validated ≤ maxSortOrder
	} else {
		next, err := h.q.AdminNextAiCreditPackSortOrder(c.Context())
		if err != nil {
			return err
		}
		sortOrder = uint32(next) //nolint:gosec // G115: small MAX()+1
	}
	params := store.AdminCreateAiCreditPackParams{
		Code: httpadmin.String(data, "code"), Title: form.Clean(data, "title").V, Badge: form.Clean(data, "badge"),
		Credits:       uint32(u64(data, "credits")), //nolint:gosec // G115: validated ≤ MaxPackCredits
		PriceRials:    u64(data, "price_rials"),
		IsHighlighted: boolOr(data, "is_highlighted", false), IsActive: boolOr(data, "is_active", true),
		SortOrder: sortOrder, Now: httpadmin.DBTime(httpadmin.Now(c)),
	}
	var id uint64
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		lastID, err := q.AdminCreateAiCreditPack(c.Context(), params)
		if err != nil {
			return err
		}
		id = uint64(lastID) //nolint:gosec // G115: auto-increment ids are positive
		return h.record(c, q, ledger{name: "credit_pack.create", targetType: "credit_pack", targetID: id,
			details: jsonx.Obj("code", params.Code, "credits", params.Credits, "price_rials", params.PriceRials)})
	})
	if err != nil {
		return err
	}
	return h.respondPack(c, id, true, "Credit pack created.")
}

// UpdatePack is PUT /ai/credit-packs/:id (super). The code is fixed at creation; absent optional fields keep their
// value. Invoices keep the credits and price they were issued with.
func (h *Handlers) UpdatePack(c fiber.Ctx) error {
	cur, err := h.findPack(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, packRules(c, false))
	if err != nil {
		return err
	}
	p := store.AdminUpdateAiCreditPackParams{
		Title: form.Clean(data, "title").V, Badge: cur.Badge,
		Credits:       uint32(u64(data, "credits")), //nolint:gosec // G115: validated ≤ MaxPackCredits
		PriceRials:    u64(data, "price_rials"),
		IsHighlighted: boolOr(data, "is_highlighted", cur.IsHighlighted), IsActive: boolOr(data, "is_active", cur.IsActive),
		SortOrder: cur.SortOrder, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}
	if _, sent := data.Get("badge"); sent {
		p.Badge = form.Clean(data, "badge")
	}
	if form.Has(data, "sort_order") {
		p.SortOrder = uint32(u64(data, "sort_order")) //nolint:gosec // G115: validated ≤ maxSortOrder
	}
	details := jsonx.Obj()
	if cur.Credits != p.Credits {
		details.Set("credits", jsonx.Obj("from", cur.Credits, "to", p.Credits))
	}
	if cur.PriceRials != p.PriceRials {
		details.Set("price_rials", jsonx.Obj("from", cur.PriceRials, "to", p.PriceRials))
	}
	if cur.IsActive != p.IsActive {
		details.Set("is_active", jsonx.Obj("from", cur.IsActive, "to", p.IsActive))
	}
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		if err := q.AdminUpdateAiCreditPack(c.Context(), p); err != nil {
			return err
		}
		return h.record(c, q, ledger{name: "credit_pack.update", targetType: "credit_pack", targetID: cur.ID, details: details})
	})
	if err != nil {
		return err
	}
	return h.respondPack(c, cur.ID, false, "Credit pack updated.")
}

// DestroyPack is DELETE /ai/credit-packs/:id (super). A pack an invoice references is deactivated (and
// un-highlighted) instead: data.deleted false, data.deactivated true.
func (h *Handlers) DestroyPack(c fiber.Ctx) error {
	cur, err := h.findPack(c)
	if err != nil {
		return err
	}
	deleted := false
	err = h.inTx(c.Context(), func(q *store.Queries) error {
		n, err := q.AdminDeleteUnusedAiCreditPack(c.Context(), cur.ID)
		if err != nil {
			return err
		}
		if n > 0 {
			deleted = true
			return h.record(c, q, ledger{name: "credit_pack.delete", targetType: "credit_pack", targetID: cur.ID,
				details: jsonx.Obj("code", cur.Code)})
		}
		if err := q.AdminDeactivateAiCreditPack(c.Context(), store.AdminDeactivateAiCreditPackParams{
			Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID}); err != nil {
			return err
		}
		return h.record(c, q, ledger{name: "credit_pack.deactivate", targetType: "credit_pack", targetID: cur.ID,
			details: jsonx.Obj("code", cur.Code)})
	})
	if err != nil {
		return err
	}
	if deleted {
		return httpadmin.OK(c, jsonx.Obj("id", cur.ID, "deleted", true, "deactivated", false), "Credit pack deleted.")
	}
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID, "deleted", false, "deactivated", true),
		"The pack has invoices, so it was deactivated instead of deleted.")
}

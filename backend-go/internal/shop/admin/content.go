package admin

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/shop/checklist"
	"github.com/ritme/backend-go/internal/shop/order"
	"github.com/ritme/backend-go/internal/shop/store"
)

// Review statuses (shop_product_reviews.status); only published ones count in the rating cache.
const (
	ReviewPending   = "pending"
	ReviewPublished = "published"
	ReviewRejected  = "rejected"
)

var codeRe = regexp.MustCompile(`^[A-Z0-9_-]{3,32}$`)

// ─── reviews ───────────────────────────────────────────────────────────────────────────────────────────────────────

// ListReviews is GET /shop/reviews?status=pending|published|rejected|all&page= (default pending), newest first.
func (a *Admin) ListReviews(c fiber.Ctx) error {
	status := strings.TrimSpace(c.Query("status", ReviewPending))
	if status == "all" {
		status = ""
	}
	p := httpadmin.PageOf(c)
	rows, err := a.q.AdminListReviews(c, store.AdminListReviewsParams{Status: status, Limit: int32(p.PerPage), Offset: int32(p.Offset())}) //nolint:gosec // bounded
	if err != nil {
		return err
	}
	total, err := a.q.AdminCountReviews(c, store.AdminCountReviewsParams{Status: status})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonx.Obj("id", r.ID, "product_id", r.ProductID, "product_title", trans(r.ProductTitle),
			"author_name", r.AuthorName, "variant_label", ntrans(r.VariantLabel), "rating", r.Rating,
			"body", httpadmin.NullString(r.Body), "status", r.Status, "created_at", httpadmin.Time(r.CreatedAt)))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// ModerateReview is POST /shop/reviews/:id/moderate {status}; the product's rating cache is refreshed.
func (a *Admin) ModerateReview(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Review")
	}
	r, err := a.q.AdminGetReview(c, id)
	if err != nil {
		return notFound(err, "Review")
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("status", "required|string|in:"+ReviewPending+","+ReviewPublished+","+ReviewRejected),
	})
	if err != nil {
		return err
	}
	status := httpadmin.String(data, "status")
	err = a.inTx(c, func(q *store.Queries) error {
		if err := q.AdminSetReviewStatus(c, store.AdminSetReviewStatusParams{Status: status, UpdatedAt: a.now(c), ID: r.ID}); err != nil {
			return err
		}
		return q.RefreshProductRating(c, r.ProductID)
	})
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_review.moderate", "shop_review", r.ID)
	return httpadmin.OK(c, jsonx.Obj("id", r.ID, "status", status), "Review updated.")
}

// DestroyReview is DELETE /shop/reviews/:id (rating cache refreshed).
func (a *Admin) DestroyReview(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Review")
	}
	r, err := a.q.AdminGetReview(c, id)
	if err != nil {
		return notFound(err, "Review")
	}
	err = a.inTx(c, func(q *store.Queries) error {
		if _, err := q.AdminDeleteReview(c, r.ID); err != nil {
			return err
		}
		return q.RefreshProductRating(c, r.ProductID)
	})
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_review.delete", "shop_review", r.ID)
	return httpadmin.OK(c, jsonx.Obj("id", r.ID), "Review deleted.")
}

// ─── discount codes ────────────────────────────────────────────────────────────────────────────────────────────────

func codeJSON(d *store.ShopDiscountCode) *jsonx.OrderedMap {
	var maxUses, perUser any
	if d.MaxUses.Valid {
		maxUses = d.MaxUses.Int32
	}
	if d.PerUserLimit.Valid {
		perUser = d.PerUserLimit.Int16
	}
	return jsonx.Obj("id", d.ID, "code", d.Code, "kind", d.Kind, "value", d.Value, "max_discount", nullInt(d.MaxDiscount),
		"min_items_total", d.MinItemsTotal, "max_uses", maxUses, "used_count", d.UsedCount, "per_user_limit", perUser,
		"starts_at", httpadmin.Time(d.StartsAt), "ends_at", httpadmin.Time(d.EndsAt), "is_active", d.IsActive,
		"created_at", httpadmin.Time(d.CreatedAt), "updated_at", httpadmin.Time(d.UpdatedAt))
}

// ListDiscountCodes is GET /shop/discount-codes?page= (newest first).
func (a *Admin) ListDiscountCodes(c fiber.Ctx) error {
	p := httpadmin.PageOf(c)
	rows, err := a.q.AdminListDiscountCodes(c, store.AdminListDiscountCodesParams{Limit: int32(p.PerPage), Offset: int32(p.Offset())}) //nolint:gosec // bounded
	if err != nil {
		return err
	}
	total, err := a.q.AdminCountDiscountCodes(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, codeJSON(&rows[i]))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

func (a *Admin) discountCode(c fiber.Ctx) (store.ShopDiscountCode, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.ShopDiscountCode{}, httpadmin.NotFound("Discount code")
	}
	d, err := a.q.GetDiscountCode(c, id)
	return d, notFound(err, "Discount code")
}

// ShowDiscountCode is GET /shop/discount-codes/:id.
func (a *Admin) ShowDiscountCode(c fiber.Ctx) error {
	d, err := a.discountCode(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("discount_code", codeJSON(&d)))
}

func (a *Admin) validateCode(c fiber.Ctx, exceptID uint64) (store.AdminCreateDiscountCodeParams, error) {
	rules := validation.Rules{
		validation.F("code", "required|string|max:32"),
		validation.F("kind", "required|string|in:"+order.DiscountPercent+","+order.DiscountFixed),
		validation.F("value", "required|integer|min:1|max:10000000000"),
		validation.F("max_discount", "nullable|integer|min:1|max:10000000000"),
		validation.F("min_items_total", "nullable|integer|min:0|max:10000000000"),
		validation.F("max_uses", "nullable|integer|min:1|max:1000000"),
		validation.F("per_user_limit", "nullable|integer|min:1|max:100"),
		validation.F("starts_at", "nullable|date"),
		validation.F("ends_at", "nullable|date"),
		validation.F("is_active", "nullable|boolean"),
	}
	data, err := form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		if httpadmin.String(in, "kind") == order.DiscountPercent && phpval.ToFloat(get(in, "value")) > 100 {
			add("value", form.Msg(c, "validation.max.numeric", "value", "max", "100"))
		}
		s, sok := form.ParseTime(httpadmin.String(in, "starts_at"))
		e, eok := form.ParseTime(httpadmin.String(in, "ends_at"))
		if sok && eok && !e.After(s) {
			add("ends_at", form.Msg(c, "validation.after", "ends_at", "date", httpadmin.AttributeName(c, "starts_at")))
		}
		code := strings.ToUpper(strings.TrimSpace(httpadmin.String(in, "code")))
		if code == "" {
			return nil
		}
		if !codeRe.MatchString(code) {
			add("code", form.Msg(c, "validation.regex", "code"))
			return nil
		}
		n, err := a.q.AdminDiscountCodeTaken(c, store.AdminDiscountCodeTakenParams{Code: code, ExceptID: exceptID})
		if err != nil {
			return err
		}
		if n > 0 {
			add("code", form.Msg(c, "validation.unique", "code"))
		}
		return nil
	})
	if err != nil {
		return store.AdminCreateDiscountCodeParams{}, err
	}
	tm := func(key string) sql.NullTime {
		if t, ok := form.ParseTime(httpadmin.String(data, key)); ok {
			return httpadmin.DBTime(t)
		}
		return sql.NullTime{}
	}
	var maxUses sql.NullInt32
	if form.Has(data, "max_uses") {
		maxUses = sql.NullInt32{Int32: form.Int32(data, "max_uses", 0), Valid: true}
	}
	now := a.now(c)
	return store.AdminCreateDiscountCodeParams{
		Code: strings.ToUpper(strings.TrimSpace(httpadmin.String(data, "code"))), Kind: httpadmin.String(data, "kind"),
		Value: uint64Of(data, "value"), MaxDiscount: nullInt64(data, "max_discount"),
		MinItemsTotal: uint64Of(data, "min_items_total"), MaxUses: maxUses, PerUserLimit: form.NullInt16(data, "per_user_limit"),
		StartsAt: tm("starts_at"), EndsAt: tm("ends_at"), IsActive: httpadmin.Bool(data, "is_active"),
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// StoreDiscountCode is POST /shop/discount-codes (codes are stored upper-case).
func (a *Admin) StoreDiscountCode(c fiber.Ctx) error {
	p, err := a.validateCode(c, 0)
	if err != nil {
		return err
	}
	id, err := a.q.AdminCreateDiscountCode(c, p)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_discount_code.create", "shop_discount_code", uint64(id)) //nolint:gosec // G115
	d, err := a.q.GetDiscountCode(c, uint64(id))                                                //nolint:gosec // G115
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("discount_code", codeJSON(&d)), "Discount code created.")
}

// UpdateDiscountCode is PUT /shop/discount-codes/:id (used_count is kept).
func (a *Admin) UpdateDiscountCode(c fiber.Ctx) error {
	cur, err := a.discountCode(c)
	if err != nil {
		return err
	}
	p, err := a.validateCode(c, cur.ID)
	if err != nil {
		return err
	}
	if err := a.q.AdminUpdateDiscountCode(c, store.AdminUpdateDiscountCodeParams{Code: p.Code, Kind: p.Kind, Value: p.Value,
		MaxDiscount: p.MaxDiscount, MinItemsTotal: p.MinItemsTotal, MaxUses: p.MaxUses, PerUserLimit: p.PerUserLimit,
		StartsAt: p.StartsAt, EndsAt: p.EndsAt, IsActive: p.IsActive, UpdatedAt: p.UpdatedAt, ID: cur.ID}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_discount_code.update", "shop_discount_code", cur.ID)
	d, err := a.q.GetDiscountCode(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("discount_code", codeJSON(&d)), "Discount code updated.")
}

// DestroyDiscountCode is DELETE /shop/discount-codes/:id (orders keep their totals; the reference is cleared).
func (a *Admin) DestroyDiscountCode(c fiber.Ctx) error {
	cur, err := a.discountCode(c)
	if err != nil {
		return err
	}
	if _, err := a.q.AdminDeleteDiscountCode(c, cur.ID); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_discount_code.delete", "shop_discount_code", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Discount code deleted.")
}

// ─── layette template ──────────────────────────────────────────────────────────────────────────────────────────────

func layetteJSON(it *store.ShopLayetteItem) *jsonx.OrderedMap {
	var qty any
	if it.SuggestedQty.Valid {
		qty = it.SuggestedQty.Int16
	}
	return jsonx.Obj("id", it.ID, "group_key", it.GroupKey, "title", trans(it.Title), "note", ntrans(it.Note),
		"suggested_qty", qty, "is_essential", it.IsEssential, "is_first_days", it.IsFirstDays,
		"category_id", nullInt(it.CategoryID), "sort_order", it.SortOrder, "is_active", it.IsActive,
		"created_at", httpadmin.Time(it.CreatedAt), "updated_at", httpadmin.Time(it.UpdatedAt))
}

// ListLayetteItems is GET /shop/layette-items (every row, by group then order).
func (a *Admin) ListLayetteItems(c fiber.Ctx) error {
	rows, err := a.q.AdminListLayetteItems(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, layetteJSON(&rows[i]))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

func (a *Admin) layetteItem(c fiber.Ctx) (store.ShopLayetteItem, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.ShopLayetteItem{}, httpadmin.NotFound("Checklist item")
	}
	it, err := a.q.AdminGetLayetteItem(c, id)
	return it, notFound(err, "Checklist item")
}

// ShowLayetteItem is GET /shop/layette-items/:id.
func (a *Admin) ShowLayetteItem(c fiber.Ctx) error {
	it, err := a.layetteItem(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("item", layetteJSON(&it)))
}

func (a *Admin) validateLayette(c fiber.Ctx) (store.AdminCreateLayetteItemParams, error) {
	rules := form.Translatable(c, "title", true, "max:120")
	rules = append(rules, form.Translatable(c, "note", false, "max:300")...)
	rules = append(rules,
		validation.F("group_key", "required|string|in:"+strings.Join(checklist.Groups, ",")),
		validation.F("suggested_qty", "nullable|integer|min:1|max:99"),
		validation.F("is_essential", "nullable|boolean"),
		validation.F("is_first_days", "nullable|boolean"),
		validation.F("category_id", "nullable|integer"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable|boolean"),
	)
	data, err := form.Validate(c, rules,
		exists(c, "category_id", func(id uint64) error { _, err := a.q.AdminGetCategory(c, id); return err }))
	if err != nil {
		return store.AdminCreateLayetteItemParams{}, err
	}
	cat := sql.NullInt64{}
	if id, ok := parseID(httpadmin.String(data, "category_id")); ok {
		cat = idOf(id)
	}
	now := a.now(c)
	return store.AdminCreateLayetteItemParams{GroupKey: httpadmin.String(data, "group_key"),
		Title: string(form.ReqJSON(data, "title")), Note: nraw(form.Clean(data, "note")),
		SuggestedQty: form.NullInt16(data, "suggested_qty"), IsEssential: httpadmin.Bool(data, "is_essential"),
		IsFirstDays: httpadmin.Bool(data, "is_first_days"), CategoryID: cat, SortOrder: form.Int32(data, "sort_order", 0),
		IsActive: httpadmin.Bool(data, "is_active"), CreatedAt: now, UpdatedAt: now}, nil
}

// StoreLayetteItem is POST /shop/layette-items.
func (a *Admin) StoreLayetteItem(c fiber.Ctx) error {
	p, err := a.validateLayette(c)
	if err != nil {
		return err
	}
	id, err := a.q.AdminCreateLayetteItem(c, p)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_layette_item.create", "shop_layette_item", uint64(id)) //nolint:gosec // G115
	it, err := a.q.AdminGetLayetteItem(c, uint64(id))                                         //nolint:gosec // G115
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("item", layetteJSON(&it)), "Checklist item created.")
}

// UpdateLayetteItem is PUT /shop/layette-items/:id.
func (a *Admin) UpdateLayetteItem(c fiber.Ctx) error {
	cur, err := a.layetteItem(c)
	if err != nil {
		return err
	}
	p, err := a.validateLayette(c)
	if err != nil {
		return err
	}
	if err := a.q.AdminUpdateLayetteItem(c, store.AdminUpdateLayetteItemParams{GroupKey: p.GroupKey, Title: p.Title, Note: p.Note,
		SuggestedQty: p.SuggestedQty, IsEssential: p.IsEssential, IsFirstDays: p.IsFirstDays, CategoryID: p.CategoryID,
		SortOrder: p.SortOrder, IsActive: p.IsActive, UpdatedAt: p.UpdatedAt, ID: cur.ID}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_layette_item.update", "shop_layette_item", cur.ID)
	it, err := a.q.AdminGetLayetteItem(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("item", layetteJSON(&it)), "Checklist item updated.")
}

// DestroyLayetteItem is DELETE /shop/layette-items/:id. Users' ticks on the row go with it, so the screen offers
// deactivation first (an inactive row is hidden from new checklists).
func (a *Admin) DestroyLayetteItem(c fiber.Ctx) error {
	cur, err := a.layetteItem(c)
	if err != nil {
		return err
	}
	if _, err := a.q.AdminDeleteLayetteItem(c, cur.ID); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_layette_item.delete", "shop_layette_item", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Checklist item deleted.")
}

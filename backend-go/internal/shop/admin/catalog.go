package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/shop/checklist"
	"github.com/ritme/backend-go/internal/shop/order"
	"github.com/ritme/backend-go/internal/shop/store"
)

// Catalog limits and vocabularies (docs/canvas-build/shop.md → data shapes).
const (
	imageDir      = "shop"
	imageMaxKB    = 4096
	imageMinDim   = 200
	maxImages     = 8
	maxAttributes = 30
	maxBadges     = 10
	lowStock      = 5  // GET /shop/stock?low=1
	slotSpanDays  = 62 // one generate call
	maxWindows    = 6
)

// SkinTypes are the keys a beauty product may list in skin_types (the client labels them).
var SkinTypes = []string{"oily", "combination", "normal", "dry", "sensitive"}

var (
	slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
	keyRe  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
	hhmmRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
)

// Options is GET /shop/commerce-options: sellers, the category tree and every vocabulary the screens select from.
func (a *Admin) Options(c fiber.Ctx) error {
	sellers, err := a.q.AdminListSellers(c)
	if err != nil {
		return err
	}
	ss := make([]*jsonx.OrderedMap, 0, len(sellers))
	for i := range sellers {
		s := &sellers[i]
		ss = append(ss, jsonx.Obj("id", s.ID, "slug", s.Slug, "name", trans(s.Name), "is_active", s.IsActive))
	}
	cats, err := a.q.AdminListCategoryTree(c)
	if err != nil {
		return err
	}
	cs := make([]*jsonx.OrderedMap, 0, len(cats))
	for _, cat := range cats {
		cs = append(cs, jsonx.Obj("id", cat.ID, "slug", cat.Slug, "title", trans(cat.Title),
			"parent_id", nullInt(cat.ParentID), "is_active", cat.IsActive))
	}
	return httpadmin.OK(c, jsonx.Obj("sellers", ss, "categories", cs,
		"layette_groups", checklist.Groups, "skin_types", SkinTypes,
		"shipment_statuses", []string{order.ShipmentAwaiting, order.ShipmentConfirmed, order.ShipmentShipped, order.StatusDelivered, order.StatusCancelled},
		"return_statuses", []string{ReturnRequested, ReturnApproved, ReturnRejected},
		"review_statuses", []string{ReviewPending, ReviewPublished, ReviewRejected},
		"discount_kinds", []string{order.DiscountPercent, order.DiscountFixed},
		"low_stock", lowStock,
		"image", jsonx.Obj("max_kb", imageMaxKB, "min_width", imageMinDim, "min_height", imageMinDim,
			"types", []string{"jpeg", "png", "webp"}, "max_count", maxImages)))
}

// ─── sellers ───────────────────────────────────────────────────────────────────────────────────────────────────────

func sellerJSON(s *store.AdminListSellersRow) *jsonx.OrderedMap {
	return jsonx.Obj("id", s.ID, "slug", s.Slug, "name", trans(s.Name), "is_verified", s.IsVerified,
		"ship_min_days", s.ShipMinDays, "ship_max_days", s.ShipMaxDays, "discreet_packaging", s.DiscreetPackaging,
		"shipping_fee", s.ShippingFee, "free_shipping_over", nullInt(s.FreeShippingOver), "cod_max_total", nullInt(s.CodMaxTotal),
		"return_days", s.ReturnDays, "sort_order", s.SortOrder, "is_active", s.IsActive, "products_count", s.ProductsCount,
		"created_at", httpadmin.Time(s.CreatedAt), "updated_at", httpadmin.Time(s.UpdatedAt))
}

func (a *Admin) sellerRow(c fiber.Ctx, id uint64) (*store.AdminListSellersRow, error) {
	rows, err := a.q.AdminListSellers(c)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i], nil
		}
	}
	return nil, httpadmin.NotFound("Seller")
}

// ListSellers is GET /shop/sellers (display order, with product counts).
func (a *Admin) ListSellers(c fiber.Ctx) error {
	rows, err := a.q.AdminListSellers(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, sellerJSON(&rows[i]))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

// ShowSeller is GET /shop/sellers/:id.
func (a *Admin) ShowSeller(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Seller")
	}
	s, err := a.sellerRow(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("seller", sellerJSON(s)))
}

func (a *Admin) validateSeller(c fiber.Ctx, exceptID uint64) (store.AdminCreateSellerParams, error) {
	rules := form.Translatable(c, "name", true, "max:120")
	rules = append(rules,
		validation.F("slug", "required|string|max:64"),
		validation.F("is_verified", "nullable|boolean"),
		validation.F("ship_min_days", "required|integer|min:0|max:60"),
		validation.F("ship_max_days", "required|integer|min:0|max:60"),
		validation.F("discreet_packaging", "nullable|boolean"),
		validation.F("shipping_fee", "required|integer|min:0|max:100000000"),
		validation.F("free_shipping_over", "nullable|integer|min:0|max:10000000000"),
		validation.F("cod_max_total", "nullable|integer|min:0|max:10000000000"),
		validation.F("return_days", "required|integer|min:0|max:60"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable|boolean"),
	)
	data, err := form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		if lo, hi := phpval.ToFloat(get(in, "ship_min_days")), phpval.ToFloat(get(in, "ship_max_days")); hi < lo {
			add("ship_max_days", form.Msg(c, "validation.gte.numeric", "ship_max_days", "value", phpval.ToString(get(in, "ship_min_days"))))
		}
		slug := strings.TrimSpace(httpadmin.String(in, "slug"))
		if slug == "" {
			return nil
		}
		if !slugRe.MatchString(slug) {
			add("slug", form.Msg(c, "validation.regex", "slug"))
			return nil
		}
		n, err := a.q.AdminSellerSlugTaken(c, store.AdminSellerSlugTakenParams{Slug: slug, ExceptID: exceptID})
		if err != nil {
			return err
		}
		if n > 0 {
			add("slug", form.Msg(c, "validation.unique", "slug"))
		}
		return nil
	})
	if err != nil {
		return store.AdminCreateSellerParams{}, err
	}
	now := a.now(c)
	return store.AdminCreateSellerParams{
		Slug: strings.TrimSpace(httpadmin.String(data, "slug")), Name: string(form.ReqJSON(data, "name")),
		IsVerified: httpadmin.Bool(data, "is_verified"), ShipMinDays: uint8(form.Int(data, "ship_min_days", 1)), //nolint:gosec // validated 0..60
		ShipMaxDays: uint8(form.Int(data, "ship_max_days", 3)), DiscreetPackaging: httpadmin.Bool(data, "discreet_packaging"), //nolint:gosec // validated
		ShippingFee: uint64Of(data, "shipping_fee"), FreeShippingOver: nullInt64(data, "free_shipping_over"),
		CodMaxTotal: nullInt64(data, "cod_max_total"), ReturnDays: uint8(form.Int(data, "return_days", 7)), //nolint:gosec // validated
		SortOrder: form.Int32(data, "sort_order", 0), IsActive: httpadmin.Bool(data, "is_active"), CreatedAt: now, UpdatedAt: now,
	}, nil
}

// get reads a raw input value (nil when missing).
func get(in phpval.Map, key string) any {
	v, _ := in.Get(key)
	return v
}

// StoreSeller is POST /shop/sellers.
func (a *Admin) StoreSeller(c fiber.Ctx) error {
	p, err := a.validateSeller(c, 0)
	if err != nil {
		return err
	}
	id, err := a.q.AdminCreateSeller(c, p)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_seller.create", "shop_seller", uint64(id)) //nolint:gosec // G115
	s, err := a.sellerRow(c, uint64(id))                                          //nolint:gosec // G115
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("seller", sellerJSON(s)), "Seller created.")
}

// UpdateSeller is PUT /shop/sellers/:id.
func (a *Admin) UpdateSeller(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Seller")
	}
	if _, err := a.sellerRow(c, id); err != nil {
		return err
	}
	p, err := a.validateSeller(c, id)
	if err != nil {
		return err
	}
	if err := a.q.AdminUpdateSeller(c, store.AdminUpdateSellerParams{Slug: p.Slug, Name: p.Name, IsVerified: p.IsVerified,
		ShipMinDays: p.ShipMinDays, ShipMaxDays: p.ShipMaxDays, DiscreetPackaging: p.DiscreetPackaging,
		ShippingFee: p.ShippingFee, FreeShippingOver: p.FreeShippingOver, CodMaxTotal: p.CodMaxTotal,
		ReturnDays: p.ReturnDays, SortOrder: p.SortOrder, IsActive: p.IsActive, UpdatedAt: p.UpdatedAt, ID: id}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_seller.update", "shop_seller", id)
	s, err := a.sellerRow(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("seller", sellerJSON(s)), "Seller updated.")
}

// DestroySeller is DELETE /shop/sellers/:id; 422 in_use while products or orders reference the seller (deactivate
// it instead).
func (a *Admin) DestroySeller(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Seller")
	}
	s, err := a.sellerRow(c, id)
	if err != nil {
		return err
	}
	ships, err := a.q.AdminCountSellerShipments(c, id)
	if err != nil {
		return err
	}
	if s.ProductsCount > 0 || ships > 0 {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse,
			"This seller still has products or orders. Deactivate it instead.")
	}
	if _, err := a.q.AdminDeleteSeller(c, id); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_seller.delete", "shop_seller", id)
	return httpadmin.OK(c, jsonx.Obj("id", id), "Seller deleted.")
}

// ─── categories ────────────────────────────────────────────────────────────────────────────────────────────────────

// SetCategoryParent is PUT /shop/categories/:id/parent {parent_id|null}: a two-level tree (a root such as layette /
// beauty, then its sub-categories). The parent must be a root; a category with children stays a root.
func (a *Admin) SetCategoryParent(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Category")
	}
	if _, err := a.q.AdminGetCategory(c, id); err != nil {
		return notFound(err, "Category")
	}
	data, err := form.Validate(c, validation.Rules{validation.F("parent_id", "nullable|integer")},
		func(in phpval.Map, add form.Add) error {
			v := get(in, "parent_id")
			if v == nil || phpval.ToString(v) == "" {
				return nil
			}
			pid, ok := parseID(phpval.ToString(v))
			if !ok || pid == id {
				add("parent_id", form.Msg(c, "validation.exists", "parent_id"))
				return nil
			}
			parent, err := a.q.AdminGetCategory(c, pid)
			if errors.Is(err, sql.ErrNoRows) || (err == nil && parent.ParentID.Valid) {
				add("parent_id", form.Msg(c, "validation.exists", "parent_id"))
				return nil
			}
			if err != nil {
				return err
			}
			kids, err := a.q.AdminCountChildCategories(c, idOf(id))
			if err != nil {
				return err
			}
			if kids > 0 {
				add("parent_id", form.Msg(c, "validation.prohibited", "parent_id"))
			}
			return nil
		})
	if err != nil {
		return err
	}
	parent := sql.NullInt64{}
	if pid, ok := parseID(httpadmin.String(data, "parent_id")); ok {
		parent = idOf(pid)
	}
	if err := a.q.AdminSetCategoryParent(c, store.AdminSetCategoryParentParams{ParentID: parent, UpdatedAt: a.now(c), ID: id}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_category.parent", "shop_category", id)
	cat, err := a.q.AdminGetCategory(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("category", jsonx.Obj("id", cat.ID, "slug", cat.Slug, "title", trans(cat.Title),
		"parent_id", nullInt(cat.ParentID), "is_active", cat.IsActive)), "Category updated.")
}

// ─── product commerce fields + images ──────────────────────────────────────────────────────────────────────────────

func decodeList(s sql.NullString) []any {
	if !s.Valid {
		return []any{}
	}
	var out []any
	if json.Unmarshal([]byte(s.String), &out) != nil || out == nil {
		return []any{}
	}
	return out
}

func imagePaths(s sql.NullString) []string {
	out := []string{}
	for _, v := range decodeList(s) {
		if p, ok := v.(string); ok && p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (a *Admin) commerceJSON(p *store.ShopProduct, variants int) *jsonx.OrderedMap {
	images := []*jsonx.OrderedMap{}
	for _, path := range imagePaths(p.Images) {
		images = append(images, jsonx.Obj("path", path, "url", a.imageURL(path)))
	}
	var expires any
	if p.ExpiresOn.Valid {
		expires = p.ExpiresOn.Date.String()
	}
	attrs := any(jsonx.Obj())
	if p.Attributes.Valid {
		if v := trans(p.Attributes.String); v != nil {
			attrs = v
		}
	}
	skins := []string{}
	for _, v := range decodeList(p.SkinTypes) {
		if s, ok := v.(string); ok {
			skins = append(skins, s)
		}
	}
	badges := any([]any{})
	if p.Badges.Valid {
		if v := trans(p.Badges.String); v != nil {
			badges = v
		}
	}
	return jsonx.Obj("id", p.ID, "title", trans(p.Title), "category_id", p.CategoryID, "seller_id", nullInt(p.SellerID),
		"subtitle", ntrans(p.Subtitle), "description", ntrans(p.Description), "images", images,
		"attributes", attrs, "irc_code", httpadmin.NullString(p.IrcCode), "expires_on", expires,
		"skin_types", skins, "badges", badges, "discreet_packaging", p.DiscreetPackaging,
		"rating_avg", httpadmin.NullString(p.RatingAvg), "rating_count", p.RatingCount, "purchase_count", p.PurchaseCount,
		"variants_count", variants, "is_link_out", p.ExternalUrl.Valid && p.ExternalUrl.String != "" && variants == 0)
}

func (a *Admin) product(c fiber.Ctx) (store.ShopProduct, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.ShopProduct{}, httpadmin.NotFound("Product")
	}
	p, err := a.q.AdminGetProduct(c, id)
	return p, notFound(err, "Product")
}

func (a *Admin) commerceOut(c fiber.Ctx, id uint64, msg ...string) error {
	p, err := a.q.AdminGetProduct(c, id)
	if err != nil {
		return err
	}
	vs, err := a.q.AdminListVariantsByProduct(c, id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("product", a.commerceJSON(&p, len(vs))), msg...)
}

// ShowCommerce is GET /shop/products/:id/commerce.
func (a *Admin) ShowCommerce(c fiber.Ctx) error {
	p, err := a.product(c)
	if err != nil {
		return err
	}
	return a.commerceOut(c, p.ID)
}

// isTextOrTranslations: a plain string or a {code: string} object (attributes values, badges).
func isTextOrTranslations(v any) bool {
	if _, ok := v.(string); ok {
		return true
	}
	if !phpval.IsArray(v) {
		return false
	}
	keys, vals := phpval.Entries(v)
	if len(keys) == 0 {
		return false
	}
	for i, k := range keys {
		if _, ok := vals[i].(string); !ok || k == "" {
			return false
		}
	}
	return true
}

// UpdateCommerce is PUT /shop/products/:id/commerce: seller, subtitle, description, attributes, IRC code, expiry,
// skin types, badges, plain packaging and the gallery order (images: a subset of the stored paths; removed files are
// deleted). Uploading a new image is POST /shop/products/:id/images.
func (a *Admin) UpdateCommerce(c fiber.Ctx) error {
	cur, err := a.product(c)
	if err != nil {
		return err
	}
	current := imagePaths(cur.Images)
	rules := form.Translatable(c, "subtitle", false, "max:200")
	rules = append(rules, form.Translatable(c, "description", false, "max:5000")...)
	rules = append(rules,
		validation.F("seller_id", "nullable|integer"),
		validation.F("attributes", "nullable|array"),
		validation.F("irc_code", "nullable|string|max:40"),
		validation.F("expires_on", "nullable|date_format:Y-m-d"),
		validation.F("skin_types", "nullable|array"),
		validation.F("skin_types.*", "string|in:"+strings.Join(SkinTypes, ",")),
		validation.F("badges", "nullable|array|max:"+itoa(maxBadges)),
		validation.F("images", "nullable|array|max:"+itoa(maxImages)),
		validation.F("images.*", "string"),
		validation.F("discreet_packaging", "nullable|boolean"),
	)
	data, err := form.Validate(c, rules,
		exists(c, "seller_id", func(id uint64) error { _, err := a.q.GetSeller(c, id); return err }),
		func(in phpval.Map, add form.Add) error {
			if v := get(in, "attributes"); v != nil && phpval.IsArray(v) {
				keys, vals := phpval.Entries(v)
				if len(keys) > maxAttributes {
					add("attributes", form.Msg(c, "validation.max.array", "attributes", "max", itoa(maxAttributes)))
				}
				for i, k := range keys {
					if !keyRe.MatchString(k) || !isTextOrTranslations(vals[i]) {
						add("attributes", form.Msg(c, "validation.array", "attributes"))
						break
					}
				}
			}
			if v := get(in, "badges"); v != nil && phpval.IsArray(v) {
				_, vals := phpval.Entries(v)
				for _, b := range vals {
					if !isTextOrTranslations(b) {
						add("badges", form.Msg(c, "validation.array", "badges"))
						break
					}
				}
			}
			if v := get(in, "images"); v != nil && phpval.IsArray(v) {
				_, vals := phpval.Entries(v)
				for _, p := range vals {
					if !slices.Contains(current, phpval.ToString(p)) {
						add("images", form.Msg(c, "validation.in", "images"))
						break
					}
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	in := validation.Input(c) // the JSON blobs as sent (the engine's validated copy keeps only ruled keys)
	attrs, badges, skins := sql.NullString{}, sql.NullString{}, sql.NullString{}
	if v := get(in, "attributes"); v != nil && phpval.Count(v) > 0 {
		attrs = sql.NullString{String: string(form.JSON(v)), Valid: true}
	}
	if v := get(in, "badges"); v != nil && phpval.Count(v) > 0 {
		_, vals := phpval.Entries(v)
		badges = sql.NullString{String: string(form.JSON(vals)), Valid: true}
	}
	if v := get(in, "skin_types"); v != nil && phpval.Count(v) > 0 {
		_, vals := phpval.Entries(v)
		uniq := []string{}
		for _, s := range vals {
			if k := phpval.ToString(s); !slices.Contains(uniq, k) {
				uniq = append(uniq, k)
			}
		}
		skins = sql.NullString{String: string(form.JSON(uniq)), Valid: true}
	}
	images := current
	if v := get(in, "images"); v != nil {
		_, vals := phpval.Entries(v)
		images = []string{}
		for _, p := range vals {
			if s := phpval.ToString(p); !slices.Contains(images, s) {
				images = append(images, s)
			}
		}
	}
	expires := civildate.NullDate{}
	if s := trimmed(data, "expires_on"); s.Valid {
		if d, err := civildate.Parse(s.String); err == nil {
			expires = civildate.NullDate{Date: d, Valid: true}
		}
	}
	seller := sql.NullInt64{}
	if id, ok := parseID(httpadmin.String(data, "seller_id")); ok {
		seller = idOf(id)
	}
	if err := a.q.AdminUpdateProductCommerce(c, store.AdminUpdateProductCommerceParams{
		SellerID: seller, Subtitle: nraw(form.Clean(data, "subtitle")), Description: nraw(form.Clean(data, "description")),
		Images: imagesColumn(images), Attributes: attrs, IrcCode: trimmed(data, "irc_code"), ExpiresOn: expires,
		SkinTypes: skins, Badges: badges, DiscreetPackaging: httpadmin.Bool(data, "discreet_packaging"),
		UpdatedAt: a.now(c), ID: cur.ID,
	}); err != nil {
		return err
	}
	for _, old := range current {
		if !slices.Contains(images, old) && (!cur.ImagePath.Valid || cur.ImagePath.String != old) {
			a.deleteFile(c, old)
		}
	}
	httpadmin.Audit(c, a.logger, "shop_product.commerce", "shop_product", cur.ID)
	return a.commerceOut(c, cur.ID, "Product updated.")
}

func imagesColumn(paths []string) sql.NullString {
	if len(paths) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(form.JSON(paths)), Valid: true}
}

func (a *Admin) deleteFile(c fiber.Ctx, rel string) {
	if rel == "" {
		return
	}
	if err := a.disk.Delete(rel); err != nil {
		a.logger.WarnContext(c.Context(), "shop admin: could not delete image", slog.String("path", rel), slog.String("error", err.Error()))
	}
}

// UploadImage is POST /shop/products/:id/images (multipart `image`): appends one gallery image (≤ 8).
func (a *Admin) UploadImage(c fiber.Ctx) error {
	cur, err := a.product(c)
	if err != nil {
		return err
	}
	paths := imagePaths(cur.Images)
	var img *media.Image
	if _, err := form.Validate(c, validation.Rules{}, func(_ phpval.Map, add form.Add) error {
		if len(paths) >= maxImages {
			add("image", form.Msg(c, "validation.max.array", "image", "max", itoa(maxImages)))
		}
		return nil
	}, form.ImageRule{Field: "image", Required: true, MaxKB: imageMaxKB, MinWidth: imageMinDim, MinHeight: imageMinDim}.Check(c, &img)); err != nil {
		return err
	}
	rel, err := a.disk.Store(a.opt, imageDir, *img)
	if err != nil {
		return err
	}
	if err := a.q.AdminSetProductImages(c, store.AdminSetProductImagesParams{Images: imagesColumn(append(paths, rel)),
		UpdatedAt: a.now(c), ID: cur.ID}); err != nil {
		a.deleteFile(c, rel)
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_product.image_add", "shop_product", cur.ID)
	return a.commerceOut(c, cur.ID, "Image added.")
}

// ─── variants + stock ──────────────────────────────────────────────────────────────────────────────────────────────

func variantJSON(v *store.ShopProductVariant) *jsonx.OrderedMap {
	return jsonx.Obj("id", v.ID, "product_id", v.ProductID, "sku", httpadmin.NullString(v.Sku), "size", ntrans(v.Size),
		"color", ntrans(v.Color), "volume", ntrans(v.Volume), "price", v.Price, "compare_at_price", nullInt(v.CompareAtPrice),
		"stock", v.Stock, "sort_order", v.SortOrder, "is_active", v.IsActive,
		"created_at", httpadmin.Time(v.CreatedAt), "updated_at", httpadmin.Time(v.UpdatedAt))
}

// ListVariants is GET /shop/products/:id/variants (every variant, display order).
func (a *Admin) ListVariants(c fiber.Ctx) error {
	p, err := a.product(c)
	if err != nil {
		return err
	}
	rows, err := a.q.AdminListVariantsByProduct(c, p.ID)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, variantJSON(&rows[i]))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

func (a *Admin) validateVariant(c fiber.Ctx, exceptID uint64) (store.AdminCreateVariantParams, error) {
	var rules validation.Rules
	for _, f := range []string{"size", "color", "volume"} {
		rules = append(rules, form.Translatable(c, f, false, "max:60")...)
	}
	rules = append(rules,
		validation.F("sku", "nullable|string|max:64"),
		validation.F("price", "required|integer|min:1|max:10000000000"),
		validation.F("compare_at_price", "nullable|integer|min:1|max:10000000000"),
		validation.F("stock", "required|integer|min:0|max:1000000"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable|boolean"),
	)
	data, err := form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		sku := strings.TrimSpace(httpadmin.String(in, "sku"))
		if sku == "" {
			return nil
		}
		n, err := a.q.AdminSkuTaken(c, store.AdminSkuTakenParams{Sku: sql.NullString{String: sku, Valid: true}, ExceptID: exceptID})
		if err != nil {
			return err
		}
		if n > 0 {
			add("sku", form.Msg(c, "validation.unique", "sku"))
		}
		return nil
	})
	if err != nil {
		return store.AdminCreateVariantParams{}, err
	}
	now := a.now(c)
	return store.AdminCreateVariantParams{Sku: trimmed(data, "sku"), Size: nraw(form.Clean(data, "size")),
		Color: nraw(form.Clean(data, "color")), Volume: nraw(form.Clean(data, "volume")), Price: uint64Of(data, "price"),
		CompareAtPrice: nullInt64(data, "compare_at_price"), Stock: uint32(uint64Of(data, "stock")), //nolint:gosec // ≤ 1e6
		SortOrder: form.Int32(data, "sort_order", 0), IsActive: httpadmin.Bool(data, "is_active"), CreatedAt: now, UpdatedAt: now}, nil
}

// StoreVariant is POST /shop/products/:id/variants. The first active variant makes a link-out product internal.
func (a *Admin) StoreVariant(c fiber.Ctx) error {
	p, err := a.product(c)
	if err != nil {
		return err
	}
	in, err := a.validateVariant(c, 0)
	if err != nil {
		return err
	}
	in.ProductID = p.ID
	// CB-SHOP-12 F-7: an omitted is_active means active on create (an API caller must not ship an invisible variant
	// by accident); an explicit false still creates it inactive. Update keeps the full-replace form semantics.
	if _, sent := validation.Input(c).Get("is_active"); !sent {
		in.IsActive = true
	}
	id, err := a.q.AdminCreateVariant(c, in)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_variant.create", "shop_variant", uint64(id), slog.Uint64("product_id", p.ID)) //nolint:gosec // G115
	v, err := a.q.GetVariant(c, uint64(id))                                                                          //nolint:gosec // G115
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("variant", variantJSON(&v)), "Variant created.")
}

func (a *Admin) variant(c fiber.Ctx) (store.ShopProductVariant, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.ShopProductVariant{}, httpadmin.NotFound("Variant")
	}
	v, err := a.q.GetVariant(c, id)
	return v, notFound(err, "Variant")
}

// UpdateVariant is PUT /shop/variants/:id (full replace, stock included).
func (a *Admin) UpdateVariant(c fiber.Ctx) error {
	cur, err := a.variant(c)
	if err != nil {
		return err
	}
	in, err := a.validateVariant(c, cur.ID)
	if err != nil {
		return err
	}
	if err := a.q.AdminUpdateVariant(c, store.AdminUpdateVariantParams{Sku: in.Sku, Size: in.Size, Color: in.Color,
		Volume: in.Volume, Price: in.Price, CompareAtPrice: in.CompareAtPrice, Stock: in.Stock, SortOrder: in.SortOrder,
		IsActive: in.IsActive, UpdatedAt: in.UpdatedAt, ID: cur.ID}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_variant.update", "shop_variant", cur.ID,
		slog.Uint64("stock_from", uint64(cur.Stock)), slog.Uint64("stock_to", uint64(in.Stock)))
	v, err := a.q.GetVariant(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("variant", variantJSON(&v)), "Variant updated.")
}

// DestroyVariant is DELETE /shop/variants/:id (order lines keep their snapshot; carts drop the line).
func (a *Admin) DestroyVariant(c fiber.Ctx) error {
	cur, err := a.variant(c)
	if err != nil {
		return err
	}
	if _, err := a.q.AdminDeleteVariant(c, cur.ID); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_variant.delete", "shop_variant", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Variant deleted.")
}

// SetStock is POST /shop/variants/:id/stock {stock}: the quick stock edit of the stock screen.
func (a *Admin) SetStock(c fiber.Ctx) error {
	cur, err := a.variant(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{validation.F("stock", "required|integer|min:0|max:1000000")})
	if err != nil {
		return err
	}
	stock := uint32(uint64Of(data, "stock")) //nolint:gosec // ≤ 1e6
	if err := a.q.AdminSetVariantStock(c, store.AdminSetVariantStockParams{Stock: stock, UpdatedAt: a.now(c), ID: cur.ID}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_variant.stock", "shop_variant", cur.ID,
		slog.Uint64("stock_from", uint64(cur.Stock)), slog.Uint64("stock_to", uint64(stock)))
	v, err := a.q.GetVariant(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("variant", variantJSON(&v)), "Stock updated.")
}

// ListStock is GET /shop/stock?q=&low=1&page=: variants with their product, lowest stock first.
func (a *Admin) ListStock(c fiber.Ctx) error {
	maxStock := uint32(4_294_967_295)
	if c.Query("low") == "1" {
		maxStock = lowStock
	}
	pattern := "%" + form.EscapeLike(strings.TrimSpace(c.Query("q"))) + "%"
	p := httpadmin.PageOf(c)
	rows, err := a.q.AdminListStock(c, store.AdminListStockParams{MaxStock: maxStock,
		SkuPattern: sql.NullString{String: pattern, Valid: true}, TitlePattern: pattern,
		Limit: int32(p.PerPage), Offset: int32(p.Offset())}) //nolint:gosec // bounded paging
	if err != nil {
		return err
	}
	total, err := a.q.AdminCountStock(c, store.AdminCountStockParams{MaxStock: maxStock,
		SkuPattern: sql.NullString{String: pattern, Valid: true}, TitlePattern: pattern})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonx.Obj("id", r.ID, "product_id", r.ProductID, "product_title", trans(r.ProductTitle),
			"seller_id", nullInt(r.SellerID), "sku", httpadmin.NullString(r.Sku), "size", ntrans(r.Size),
			"color", ntrans(r.Color), "volume", ntrans(r.Volume), "price", r.Price, "stock", r.Stock,
			"is_active", r.IsActive, "low", r.Stock <= lowStock))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// ─── delivery slots ────────────────────────────────────────────────────────────────────────────────────────────────

func slotJSON(s *store.ShopDeliverySlot) *jsonx.OrderedMap {
	var capacity any
	if s.Capacity.Valid {
		capacity = s.Capacity.Int16
	}
	return jsonx.Obj("id", s.ID, "seller_id", s.SellerID, "slot_date", s.SlotDate.String(), "starts_at", hhmm(s.StartsAt),
		"ends_at", hhmm(s.EndsAt), "capacity", capacity, "booked", s.Booked, "is_active", s.IsActive)
}

func hhmm(t string) string {
	if len(t) >= 5 {
		return t[:5]
	}
	return t
}

// ListSlots is GET /shop/slots?seller_id=&from=&to= (default: today + 14 days).
func (a *Admin) ListSlots(c fiber.Ctx) error {
	sellerID, _ := parseID(c.Query("seller_id"))
	from := civildate.InTehran(httpadmin.Now(c))
	if d, err := civildate.Parse(c.Query("from")); err == nil {
		from = d
	}
	to := from.AddDays(14)
	if d, err := civildate.Parse(c.Query("to")); err == nil && !d.Before(from) && from.DiffDays(d) <= slotSpanDays {
		to = d
	}
	items := []*jsonx.OrderedMap{}
	if sellerID > 0 {
		rows, err := a.q.AdminListSlots(c, store.AdminListSlotsParams{SellerID: sellerID, DateFrom: from, DateTo: to})
		if err != nil {
			return err
		}
		for i := range rows {
			items = append(items, slotJSON(&rows[i]))
		}
	}
	return httpadmin.OK(c, jsonx.Obj("items", items, "from", from.String(), "to", to.String()))
}

// GenerateSlots is POST /shop/slots/generate {seller_id, from, to, weekdays[0..6]?, windows[{starts_at, ends_at,
// capacity?}]}: the template — every listed weekday in the range gets each window; existing slots are kept.
func (a *Admin) GenerateSlots(c fiber.Ctx) error {
	rules := validation.Rules{
		validation.F("seller_id", "required|integer"),
		validation.F("from", "required|date_format:Y-m-d"),
		validation.F("to", "required|date_format:Y-m-d"),
		validation.F("weekdays", "nullable|array"),
		validation.F("weekdays.*", "integer|min:0|max:6"),
		validation.F("windows", "required|array|min:1|max:"+itoa(maxWindows)),
		validation.F("windows.*.starts_at", "required|string"),
		validation.F("windows.*.ends_at", "required|string"),
		validation.F("windows.*.capacity", "nullable|integer|min:1|max:1000"),
	}
	data, err := form.Validate(c, rules,
		exists(c, "seller_id", func(id uint64) error { _, err := a.q.GetSeller(c, id); return err }),
		func(in phpval.Map, add form.Add) error {
			from, e1 := civildate.Parse(httpadmin.String(in, "from"))
			to, e2 := civildate.Parse(httpadmin.String(in, "to"))
			if e1 == nil && e2 == nil && (to.Before(from) || from.DiffDays(to) > slotSpanDays) {
				add("to", form.Msg(c, "validation.after_or_equal", "to", "date", from.String()))
			}
			if w := get(in, "windows"); w != nil && phpval.IsArray(w) {
				_, vals := phpval.Entries(w)
				for _, v := range vals {
					s, _ := phpval.Get(v, "starts_at")
					e, _ := phpval.Get(v, "ends_at")
					ss, es := phpval.ToString(s), phpval.ToString(e)
					if !hhmmRe.MatchString(ss) || !hhmmRe.MatchString(es) || es <= ss {
						add("windows", form.Msg(c, "validation.date_format", "windows", "format", "H:i"))
						break
					}
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	in := validation.Input(c)
	sellerID, _ := parseID(httpadmin.String(data, "seller_id"))
	from, _ := civildate.Parse(httpadmin.String(data, "from"))
	to, _ := civildate.Parse(httpadmin.String(data, "to"))
	days := []time.Weekday{}
	if w := get(in, "weekdays"); w != nil && phpval.Count(w) > 0 {
		_, vals := phpval.Entries(w)
		for _, v := range vals {
			days = append(days, time.Weekday(int(phpval.ToFloat(v))))
		}
	}
	type window struct {
		start, end string
		capacity   sql.NullInt16
	}
	windows := []window{}
	_, vals := phpval.Entries(get(in, "windows"))
	for _, v := range vals {
		s, _ := phpval.Get(v, "starts_at")
		e, _ := phpval.Get(v, "ends_at")
		w := window{start: phpval.ToString(s) + ":00", end: phpval.ToString(e) + ":00"}
		if cv, ok := phpval.Get(v, "capacity"); ok && cv != nil && phpval.ToString(cv) != "" {
			w.capacity = sql.NullInt16{Int16: int16(phpval.ToFloat(cv)), Valid: true}
		}
		windows = append(windows, w)
	}
	now := a.now(c)
	created := int64(0)
	err = a.inTx(c, func(q *store.Queries) error {
		for d := from; !d.After(to); d = d.AddDays(1) {
			if len(days) > 0 && !slices.Contains(days, d.Weekday()) {
				continue
			}
			for _, w := range windows {
				n, err := q.AdminInsertSlot(c, store.AdminInsertSlotParams{SellerID: sellerID, SlotDate: d, StartsAt: w.start,
					EndsAt: w.end, Capacity: w.capacity, CreatedAt: now, UpdatedAt: now})
				if err != nil {
					return err
				}
				created += n
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_slot.generate", "shop_seller", sellerID, slog.Int64("created", created))
	return httpadmin.Created(c, jsonx.Obj("created", created), "Delivery times created.")
}

func (a *Admin) slot(c fiber.Ctx) (store.ShopDeliverySlot, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.ShopDeliverySlot{}, httpadmin.NotFound("Slot")
	}
	s, err := a.q.AdminGetSlot(c, id)
	return s, notFound(err, "Slot")
}

// UpdateSlot is PUT /shop/slots/:id {capacity|null, is_active}; capacity may not go below the bookings.
func (a *Admin) UpdateSlot(c fiber.Ctx) error {
	cur, err := a.slot(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("capacity", "nullable|integer|min:1|max:1000"),
		validation.F("is_active", "nullable|boolean"),
	}, func(in phpval.Map, add form.Add) error {
		if v := get(in, "capacity"); v != nil && phpval.ToString(v) != "" && phpval.ToFloat(v) < float64(cur.Booked) {
			add("capacity", form.Msg(c, "validation.min.numeric", "capacity", "min", itoa(int(cur.Booked))))
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := a.q.AdminUpdateSlot(c, store.AdminUpdateSlotParams{Capacity: form.NullInt16(data, "capacity"),
		IsActive: httpadmin.Bool(data, "is_active"), UpdatedAt: a.now(c), ID: cur.ID}); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_slot.update", "shop_slot", cur.ID)
	s, err := a.q.AdminGetSlot(c, cur.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("slot", slotJSON(&s)), "Delivery time updated.")
}

// DestroySlot is DELETE /shop/slots/:id; 422 slot_booked once an order chose it (deactivate it instead).
func (a *Admin) DestroySlot(c fiber.Ctx) error {
	cur, err := a.slot(c)
	if err != nil {
		return err
	}
	n, err := a.q.AdminDeleteSlot(c, cur.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeSlotBooked,
			"Orders already chose this delivery time. Deactivate it instead.")
	}
	httpadmin.Audit(c, a.logger, "shop_slot.delete", "shop_slot", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Delivery time deleted.")
}

func itoa(n int) string { return strconv.Itoa(n) }

package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/shop/store"
)

// ErrorCodeTeenCommercialBlocked is the error_code of the teen-mode 403.
const ErrorCodeTeenCommercialBlocked = "teen_commercial_blocked"

// Paging and section sizes.
const (
	defaultPerPage    = 20
	maxPerPage        = 50
	reviewsPerPage    = 10
	detailReviews     = 3
	sectionProducts   = 4
	maxSections       = 6
	boughtTogetherMax = 6
)

// CommercialPolicy decides whether a user may see commercial content (false for a teen-mode account); routes inject
// teen.Policy, nil allows everyone.
type CommercialPolicy interface {
	AllowsCommercial(ctx context.Context, userID uint64) (bool, error)
}

// Handlers serve /api/v1/shop/catalog/* (auth:api; the user id is only read for the teen policy).
type Handlers struct {
	svc    *Service
	policy CommercialPolicy
	appURL string
}

// NewHandlers wires the handlers.
func NewHandlers(svc *Service, policy CommercialPolicy, appURL string) *Handlers {
	return &Handlers{svc: svc, policy: policy, appURL: appURL}
}

// gate answers 403 teen_commercial_blocked for a teen-mode account and returns the locale and category tree.
func (h *Handlers) gate(c fiber.Ctx) (Locale, *Tree, error) {
	loc := Locale{Code: i18n.ResolveLocale(c, ""), Default: i18n.LanguagesOf(c).DefaultCode()}
	if h.policy != nil {
		uid, ok := auth.CurrentUserID(c)
		if !ok {
			return loc, nil, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
		}
		allowed, err := h.policy.AllowsCommercial(c, uid)
		if err != nil {
			return loc, nil, fmt.Errorf("catalog: policy: %w", err)
		}
		if !allowed {
			return loc, nil, httpx.Fail(fiber.StatusForbidden, T("errors.teen_commercial_blocked", i18n.Locale(c)),
				"error_code", ErrorCodeTeenCommercialBlocked)
		}
	}
	t, err := h.svc.Tree(c)
	return loc, t, err
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (h *Handlers) url(path string) any {
	if path == "" {
		return nil
	}
	return content.PublicURL(h.appURL, path)
}

func (h *Handlers) categoryJSON(l Locale, t *Tree, c *store.ShopCategory) *jsonx.OrderedMap {
	return jsonx.Obj("id", c.ID, "slug", c.Slug, "title", l.Pick(c.Title), "icon", nullable(c.Icon.String),
		"parent", nullable(t.Parent(c)))
}

// subCategoriesJSON lists categories with their catalog product counts (subtree), from items.
func (h *Handlers) subCategoriesJSON(l Locale, t *Tree, cats []*store.ShopCategory, items []*Item) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(cats))
	for _, c := range cats {
		sub := t.Subtree(c.ID)
		n := 0
		for _, it := range items {
			if slices.Contains(sub, it.Product.CategoryID) {
				n++
			}
		}
		out = append(out, h.categoryJSON(l, t, c).Set("product_count", n))
	}
	return out
}

func badges(l Locale, raw string) []string {
	out := []string{}
	var list []json.RawMessage
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &list) != nil {
		return out
	}
	for _, b := range list {
		if s := l.Pick(string(b)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// images is image_path then the images JSON array (paths), de-duplicated, as public URLs.
func (h *Handlers) images(p *store.ShopProduct) []any {
	paths := []string{}
	if p.ImagePath.Valid && p.ImagePath.String != "" {
		paths = append(paths, p.ImagePath.String)
	}
	var more []string
	if p.Images.Valid {
		_ = json.Unmarshal([]byte(p.Images.String), &more)
	}
	for _, m := range more {
		if m != "" && !slices.Contains(paths, m) {
			paths = append(paths, m)
		}
	}
	out := make([]any, 0, len(paths))
	for _, p := range paths {
		out = append(out, h.url(p))
	}
	return out
}

func (h *Handlers) sellerCard(l Locale, s *store.ShopSeller) any {
	if s == nil {
		return nil
	}
	return jsonx.Obj("id", s.ID, "slug", s.Slug, "name", l.Pick(s.Name), "is_verified", s.IsVerified)
}

func nullInt(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func (h *Handlers) sellerDetail(l Locale, s *store.ShopSeller) any {
	if s == nil {
		return nil
	}
	return jsonx.Obj("id", s.ID, "slug", s.Slug, "name", l.Pick(s.Name), "logo_url", h.url(s.LogoPath.String),
		"is_verified", s.IsVerified, "ship_min_days", s.ShipMinDays, "ship_max_days", s.ShipMaxDays,
		"shipping_fee", s.ShippingFee, "free_shipping_over", nullInt(s.FreeShippingOver),
		"return_days", s.ReturnDays, "discreet_packaging", s.DiscreetPackaging)
}

// cardJSON is the product tile of homes, lists and bought-together.
func (h *Handlers) cardJSON(l Locale, it *Item) *jsonx.OrderedMap {
	p := &it.Product
	imgs := h.images(p)
	var img any
	if len(imgs) > 0 {
		img = imgs[0]
	}
	return jsonx.Obj("id", p.ID, "title", l.Pick(p.Title), "subtitle", nullable(l.Pick(p.Subtitle.String)),
		"category", it.Category, "image_url", img, "price", it.Price(), "compare_at_price", it.CompareAt(),
		"rating_avg", it.Rating(), "rating_count", p.RatingCount, "purchase_count", p.PurchaseCount,
		"badges", badges(l, p.Badges.String), "in_stock", it.InStock(), "discreet_packaging", p.DiscreetPackaging,
		"seller", h.sellerCard(l, it.Seller))
}

func (h *Handlers) cards(l Locale, items []*Item) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(items))
	for _, it := range items {
		out = append(out, h.cardJSON(l, it))
	}
	return out
}

// attributeOrder puts the board's rows first; other keys follow alphabetically.
var attributeOrder = []string{"material", "wash", "size_guide", "ingredients", "usage", "warnings"}

func attributesJSON(l Locale, raw string) *jsonx.OrderedMap {
	m := Attributes(raw)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := slices.Index(attributeOrder, keys[i]), slices.Index(attributeOrder, keys[j])
		switch {
		case a >= 0 && b >= 0:
			return a < b
		case a >= 0 || b >= 0:
			return a >= 0
		}
		return keys[i] < keys[j]
	})
	out := jsonx.NewObject()
	for _, k := range keys {
		if v := l.Pick(string(m[k])); v != "" {
			out.Set(k, v)
		}
	}
	return out
}

func variantJSON(l Locale, v *store.ShopProductVariant) *jsonx.OrderedMap {
	size, color, volume := l.Pick(v.Size.String), l.Pick(v.Color.String), l.Pick(v.Volume.String)
	parts := []string{}
	for _, s := range []string{size, color, volume} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	var compare any
	if v.CompareAtPrice.Valid && v.CompareAtPrice.Int64 > 0 && uint64(v.CompareAtPrice.Int64) > v.Price {
		compare = v.CompareAtPrice.Int64
	}
	return jsonx.Obj("id", v.ID, "sku", nullable(v.Sku.String), "size", nullable(size), "color", nullable(color),
		"volume", nullable(volume), "label", strings.Join(parts, " · "), "price", v.Price, "compare_at_price", compare,
		"stock", v.Stock, "in_stock", v.Stock > 0)
}

func reviewJSON(l Locale, r *store.ShopProductReview) *jsonx.OrderedMap {
	var at any
	if r.CreatedAt.Valid {
		at = jsonx.ISO8601(r.CreatedAt.Time)
	}
	return jsonx.Obj("id", r.ID, "author_name", r.AuthorName, "rating", r.Rating, "body", nullable(r.Body.String),
		"variant_label", nullable(l.Pick(r.VariantLabel.String)), "verified_purchase", r.OrderItemID.Valid,
		"created_at", at)
}

// Home is GET /shop/catalog/home?root=<slug> → {root, roots, sub_categories, sections}. root defaults to the first
// root category; a sub-category or unknown slug is 404. sections are the root's sub-categories that have products
// (at most 6), each with its 4 best sellers.
func (h *Handlers) Home(c fiber.Ctx) error {
	l, t, err := h.gate(c)
	if err != nil {
		return err
	}
	roots := t.Roots()
	var root *store.ShopCategory
	if slug := strings.TrimSpace(c.Query("root")); slug != "" {
		cat, ok := t.BySlug(slug)
		if !ok || cat.ParentID.Valid {
			return httpx.NotFound()
		}
		root = cat
	} else if len(roots) > 0 {
		root = roots[0]
	}
	if root == nil {
		return httpx.NotFound()
	}
	items, err := h.svc.Items(c, t, t.Subtree(root.ID))
	if err != nil {
		return err
	}
	rootsJSON := make([]*jsonx.OrderedMap, 0, len(roots))
	for _, r := range roots {
		rootsJSON = append(rootsJSON, h.categoryJSON(l, t, r))
	}
	sections := []*jsonx.OrderedMap{}
	for _, sub := range t.Children(root.ID) {
		if len(sections) >= maxSections {
			break
		}
		ids := t.Subtree(sub.ID)
		top := []*Item{}
		for _, it := range items { // already most-bought first
			if len(top) < sectionProducts && slices.Contains(ids, it.Product.CategoryID) {
				top = append(top, it)
			}
		}
		if len(top) > 0 {
			sections = append(sections, jsonx.Obj("category", h.categoryJSON(l, t, sub), "products", h.cards(l, top)))
		}
	}
	return httpx.OK(c, jsonx.Obj("root", h.categoryJSON(l, t, root), "roots", rootsJSON,
		"sub_categories", h.subCategoriesJSON(l, t, t.Children(root.ID), items), "sections", sections))
}

func uintQuery(c fiber.Ctx, key string) uint64 {
	n, err := strconv.ParseUint(strings.TrimSpace(c.Query(key)), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func paging(c fiber.Ctx, def int) (page, per int) {
	page, per = 1, def
	if n, err := strconv.Atoi(c.Query("page")); err == nil && n >= 1 && n <= 100000 {
		page = n
	}
	if n, err := strconv.Atoi(c.Query("per_page")); err == nil && n >= 1 {
		per = min(n, maxPerPage)
	}
	return page, per
}

// ParseFilter reads the list query: q, size, color, volume, material, skin_type (comma-separated lists),
// price_min, price_max (toman) and sort (unknown → popular). Invalid numbers are ignored, like bloom's paging.
func ParseFilter(c fiber.Ctx) Filter {
	f := Filter{Query: strings.TrimSpace(c.Query("q")), Sizes: SplitList(c.Query("size")),
		Colors: SplitList(c.Query("color")), Volumes: SplitList(c.Query("volume")),
		Materials: SplitList(c.Query("material")), SkinTypes: SplitList(c.Query("skin_type")),
		PriceMin: uintQuery(c, "price_min"), PriceMax: uintQuery(c, "price_max"), Sort: c.Query("sort")}
	if !slices.Contains(Sorts, f.Sort) {
		f.Sort = SortPopular
	}
	if len([]rune(f.Query)) > 100 {
		f.Query = string([]rune(f.Query)[:100])
	}
	return f
}

func facetJSON(opts []FacetOption) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(opts))
	for _, o := range opts {
		out = append(out, jsonx.Obj("value", o.Value, "label", o.Label, "count", o.Count))
	}
	return out
}

// Products is GET /shop/catalog/products?category=<slug>&q&size&color&volume&material&skin_type&price_min&price_max
// &sort&page&per_page → {category, sub_categories, filters, sort, products, meta}. Without category the whole catalog
// lists and sub_categories are the roots. filters (choices with counts, price range) describe the category before
// filtering. An unknown or inactive category is 404.
func (h *Handlers) Products(c fiber.Ctx) error {
	l, t, err := h.gate(c)
	if err != nil {
		return err
	}
	var catJSON any
	subs, ids := t.Roots(), t.All()
	if slug := strings.TrimSpace(c.Query("category")); slug != "" {
		cat, ok := t.BySlug(slug)
		if !ok {
			return httpx.NotFound()
		}
		catJSON, subs, ids = h.categoryJSON(l, t, cat), t.Children(cat.ID), t.Subtree(cat.ID)
	}
	all, err := h.svc.Items(c, t, ids)
	if err != nil {
		return err
	}
	f := ParseFilter(c)
	matched := f.Apply(l, all)
	page, per := paging(c, defaultPerPage)
	from := min((page-1)*per, len(matched))
	pageItems := matched[from:min(from+per, len(matched))]
	fc := BuildFacets(l, all)
	var priceMin, priceMax any
	if len(all) > 0 {
		priceMin, priceMax = fc.PriceMin, fc.PriceMax
	}
	filters := jsonx.Obj("sizes", facetJSON(fc.Sizes), "colors", facetJSON(fc.Colors), "volumes", facetJSON(fc.Volumes),
		"materials", facetJSON(fc.Materials), "skin_types", facetJSON(fc.SkinTypes),
		"price", jsonx.Obj("min", priceMin, "max", priceMax))
	cards := h.cards(l, pageItems)
	pg := httpx.NewPage(cards, len(matched), per, page, "")
	return httpx.OK(c, jsonx.Obj("category", catJSON, "sub_categories", h.subCategoriesJSON(l, t, subs, all),
		"filters", filters, "sort", f.Sort, "products", cards, "meta", pg.Meta()))
}

func productID(c fiber.Ctx) (uint64, bool) {
	s := c.Params("id")
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil && n > 0 && len(s) < 20
}

func (h *Handlers) load(c fiber.Ctx) (Locale, *Tree, *Item, error) {
	l, t, err := h.gate(c)
	if err != nil {
		return l, nil, nil, err
	}
	id, ok := productID(c)
	if !ok {
		return l, nil, nil, httpx.NotFound()
	}
	it, err := h.svc.Product(c, t, id)
	if errors.Is(err, ErrNotFound) {
		return l, nil, nil, httpx.NotFound()
	}
	return l, t, it, err
}

// Product is GET /shop/catalog/products/:id → {product} with variants (active, with stock), seller terms,
// attributes, beauty fields (irc_code, expires_on, skin_types), the latest published reviews and bought-together
// cards. A link-out product (no variants), an inactive product or category is 404.
func (h *Handlers) Product(c fiber.Ctx) error {
	l, t, it, err := h.load(c)
	if err != nil {
		return err
	}
	p := &it.Product
	reviews, _, err := h.svc.Reviews(c, p.ID, detailReviews, 0)
	if err != nil {
		return err
	}
	revJSON := make([]*jsonx.OrderedMap, 0, len(reviews))
	for i := range reviews {
		revJSON = append(revJSON, reviewJSON(l, &reviews[i]))
	}
	variants := make([]*jsonx.OrderedMap, 0, len(it.Variants))
	for i := range it.Variants {
		variants = append(variants, variantJSON(l, &it.Variants[i]))
	}
	var defaultVariant any
	if v := it.cheapest(); v != nil {
		defaultVariant = v.ID
	}
	var expires any
	if p.ExpiresOn.Valid {
		expires = p.ExpiresOn.Date
	}
	together, err := h.svc.BoughtTogether(c, t, it, boughtTogetherMax)
	if err != nil {
		return err
	}
	out := h.cardJSON(l, it).
		Set("description", nullable(l.Pick(p.Description.String))).
		Set("images", h.images(p)).
		Set("attributes", attributesJSON(l, p.Attributes.String)).
		Set("irc_code", nullable(p.IrcCode.String)).
		Set("expires_on", expires).
		Set("skin_types", SkinTypes(p.SkinTypes.String)).
		Set("seller", h.sellerDetail(l, it.Seller)).
		Set("variants", variants).
		Set("default_variant_id", defaultVariant).
		Set("reviews", jsonx.Obj("rating_avg", it.Rating(), "rating_count", p.RatingCount, "items", revJSON)).
		Set("bought_together", h.cards(l, together))
	return httpx.OK(c, jsonx.Obj("product", out))
}

// Reviews is GET /shop/catalog/products/:id/reviews?page&per_page → {reviews, meta}: published reviews, newest first.
func (h *Handlers) Reviews(c fiber.Ctx) error {
	l, _, it, err := h.load(c)
	if err != nil {
		return err
	}
	page, per := paging(c, reviewsPerPage)
	rows, total, err := h.svc.Reviews(c, it.Product.ID, per, (page-1)*per)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, reviewJSON(l, &rows[i]))
	}
	pg := httpx.NewPage(items, total, per, page, "")
	return httpx.OK(c, jsonx.Obj("reviews", items, "meta", pg.Meta()))
}

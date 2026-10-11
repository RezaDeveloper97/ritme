package shop

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Public list paging.
const (
	defaultPerPage = 20
	maxPerPage     = 50
)

// Handlers serve the public /api/v1/shop endpoints (auth:api; the user id is only used for the teen policy and is
// never stored).
type Handlers struct {
	store  *Store
	policy CommercialPolicy
	appURL string
	clock  clock.Clock
}

// NewHandlers wires the handlers; policy may be nil (everyone allowed).
func NewHandlers(store *Store, policy CommercialPolicy, appURL string, base clock.Clock) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	return &Handlers{store: store, policy: policy, appURL: appURL, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// allowed applies the commercial policy (teen mode has no shop).
func (h *Handlers) allowed(c fiber.Ctx) (bool, error) {
	if h.policy == nil {
		return true, nil
	}
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return false, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	ok, err := h.policy.AllowsCommercial(c, uid)
	if err != nil {
		return false, fmt.Errorf("shop: policy: %w", err)
	}
	return ok, nil
}

func pick(raw json.RawMessage, c fiber.Ctx) string {
	return i18n.PickString(raw, i18n.ResolveLocale(c, ""), i18n.LanguagesOf(c).DefaultCode())
}

func nullStr(s sql.NullString) any {
	if !s.Valid || s.String == "" {
		return nil
	}
	return s.String
}

func (h *Handlers) imageURL(p sql.NullString) any {
	if !p.Valid || p.String == "" {
		return nil
	}
	return content.PublicURL(h.appURL, p.String)
}

func (h *Handlers) categoryJSON(c fiber.Ctx, cat *Category) *jsonx.OrderedMap {
	return jsonx.Obj("id", cat.ID, "slug", cat.Slug, "title", pick(cat.Title, c), "icon", nullStr(cat.Icon),
		"product_count", cat.Products)
}

func (h *Handlers) productJSON(c fiber.Ctx, p *Product) *jsonx.OrderedMap {
	var link any
	if l := LinkURL(p.ExternalURL); l != "" {
		link = l
	}
	return jsonx.Obj("id", p.ID, "category", p.CategorySlug, "title", pick(p.Title, c),
		"image_url", h.imageURL(p.ImagePath), "price_text", nullStr(p.PriceText), "badge", nullStr(p.Badge),
		"partner_label", nullStr(p.PartnerLabel), "link_url", link)
}

// Categories is GET /shop/categories → {categories:[{id, slug, title, icon, product_count}]} in display order.
// A teen-mode account gets none. Only active categories; the count is of active products.
func (h *Handlers) Categories(c fiber.Ctx) error {
	ok, err := h.allowed(c)
	if err != nil {
		return err
	}
	items := []*jsonx.OrderedMap{}
	if ok {
		cats, err := h.store.ListCategories(c, true)
		if err != nil {
			return fmt.Errorf("shop: categories: %w", err)
		}
		for i := range cats {
			items = append(items, h.categoryJSON(c, &cats[i]))
		}
	}
	return httpx.OK(c, jsonx.Obj("categories", items))
}

// Products is GET /shop/products?category=<slug>&page=&per_page= → {category, products, meta}. An unknown or
// inactive category is 404. Products without a link-out still list (link_url null).
func (h *Handlers) Products(c fiber.Ctx) error {
	ok, err := h.allowed(c)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.NotFound()
	}
	f := ProductFilter{ActiveOnly: true}
	var catJSON any
	if slug := strings.TrimSpace(c.Query("category")); slug != "" {
		cat, err := h.store.GetCategory(c, slug, false, true)
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound()
		}
		if err != nil {
			return fmt.Errorf("shop: category: %w", err)
		}
		f.CategoryID = cat.ID
		catJSON = h.categoryJSON(c, &cat)
	}
	page, per := 1, defaultPerPage
	if n, err := strconv.Atoi(c.Query("page")); err == nil && n >= 1 && n <= 100000 {
		page = n
	}
	if n, err := strconv.Atoi(c.Query("per_page")); err == nil && n >= 1 {
		per = min(n, maxPerPage)
	}
	rows, total, err := h.store.ListProducts(c, f, per, (page-1)*per, h.now(c))
	if err != nil {
		return fmt.Errorf("shop: products: %w", err)
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, h.productJSON(c, &rows[i]))
	}
	pg := httpx.NewPage(items, total, per, page, "")
	return httpx.OK(c, jsonx.Obj("category", catJSON, "products", items, "meta", pg.Meta()))
}

func productID(c fiber.Ctx) (uint64, bool) {
	s := c.Params("id")
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil && n > 0 && len(s) < 20
}

// Product is GET /shop/products/:id → {product}.
func (h *Handlers) Product(c fiber.Ctx) error {
	p, err := h.load(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("product", h.productJSON(c, &p)))
}

func (h *Handlers) load(c fiber.Ctx) (Product, error) {
	var zero Product
	ok, err := h.allowed(c)
	if err != nil {
		return zero, err
	}
	id, valid := productID(c)
	if !ok || !valid {
		return zero, httpx.NotFound()
	}
	p, err := h.store.GetProduct(c, id, true, h.now(c))
	if errors.Is(err, sql.ErrNoRows) {
		return zero, httpx.NotFound()
	}
	if err != nil {
		return zero, fmt.Errorf("shop: product: %w", err)
	}
	return p, nil
}

// Click is POST /shop/products/:id/click: counts one link-out click for today (Tehran day) and answers
// {counted:true, link_url}. 404 for a product without a link-out. Only the per-product daily total is stored.
func (h *Handlers) Click(c fiber.Ctx) error {
	p, err := h.load(c)
	if err != nil {
		return err
	}
	link := LinkURL(p.ExternalURL)
	if link == "" {
		return httpx.NotFound()
	}
	if err := h.store.CountClick(c, p.ID, civildate.InTehran(h.now(c))); err != nil {
		return fmt.Errorf("shop: click: %w", err)
	}
	return httpx.OK(c, jsonx.Obj("counted", true, "link_url", link))
}

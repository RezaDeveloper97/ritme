package shop

import (
	"database/sql"
	"errors"
	"log/slog"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// ImageDir is the public-disk folder of product images; limits mirror the banner upload (jpeg/png/webp, ≤ 4 MB).
const (
	ImageDir    = "shop"
	imageMaxKB  = 4096
	imageMinDim = 200
	// CodeInUse is the 422 error_code of deleting a category that still has products.
	CodeInUse = "in_use"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Admin serves /api/admin/v1/shop/* (capability «content», see httpadmin.capabilityOf).
type Admin struct {
	store  *Store
	disk   *media.Disk
	opt    media.Optimizer
	appURL string
	logger *slog.Logger
}

// NewAdmin builds the admin handlers.
func NewAdmin(db *sql.DB, disk *media.Disk, appURL string, logger *slog.Logger) *Admin {
	if logger == nil {
		logger = slog.Default()
	}
	if disk == nil {
		disk = media.NewDisk("")
	}
	return &Admin{store: NewStore(db), disk: disk, opt: media.DefaultOptimizer, appURL: appURL, logger: logger}
}

// Routes registers the endpoints; static paths before :id.
func (a *Admin) Routes(route Route, kit *httpadmin.Kit) {
	k := kit.Admin
	route(fiber.MethodGet, "/shop/options", k(a.Options))
	route(fiber.MethodGet, "/shop/categories", k(a.ListCategories))
	route(fiber.MethodPost, "/shop/categories", k(a.StoreCategory))
	route(fiber.MethodGet, "/shop/categories/:id", k(a.ShowCategory))
	route(fiber.MethodPut, "/shop/categories/:id", k(a.UpdateCategory))
	route(fiber.MethodDelete, "/shop/categories/:id", k(a.DestroyCategory))
	route(fiber.MethodGet, "/shop/products", k(a.ListProducts))
	route(fiber.MethodPost, "/shop/products", k(a.StoreProduct))
	route(fiber.MethodGet, "/shop/products/:id", k(a.ShowProduct))
	route(fiber.MethodPut, "/shop/products/:id", k(a.UpdateProduct))
	route(fiber.MethodPost, "/shop/products/:id", k(a.UpdateProduct)) // multipart update (admin-api.md §7)
	route(fiber.MethodPost, "/shop/products/:id/toggle", k(a.ToggleProduct))
	route(fiber.MethodDelete, "/shop/products/:id", k(a.DestroyProduct))
}

func (a *Admin) now(c fiber.Ctx) sql.NullTime { return httpadmin.DBTime(httpadmin.Now(c)) }

func nullable(data phpval.Map, key string) sql.NullString {
	s := form.Str(data, key)
	if s.Valid && strings.TrimSpace(s.String) == "" {
		return sql.NullString{}
	}
	if s.Valid {
		s.String = strings.TrimSpace(s.String)
	}
	return s
}

func (a *Admin) categoryJSON(cat *Category) *jsonx.OrderedMap {
	return jsonx.Obj("id", cat.ID, "slug", cat.Slug, "title", form.Raw(cat.Title), "icon", httpadmin.NullString(cat.Icon),
		"sort_order", cat.SortOrder, "is_active", cat.IsActive, "products_count", cat.Products,
		"created_at", httpadmin.Time(cat.CreatedAt), "updated_at", httpadmin.Time(cat.UpdatedAt))
}

func (a *Admin) productJSON(p *Product) *jsonx.OrderedMap {
	var imageURL any
	if p.ImagePath.Valid && p.ImagePath.String != "" {
		imageURL = content.PublicURL(a.appURL, p.ImagePath.String)
	}
	return jsonx.Obj("id", p.ID, "category_id", p.CategoryID, "category_slug", p.CategorySlug,
		"title", form.Raw(p.Title), "image_path", httpadmin.NullString(p.ImagePath), "image_url", imageURL,
		"price_text", httpadmin.NullString(p.PriceText), "badge", httpadmin.NullString(p.Badge),
		"external_url", httpadmin.NullString(p.ExternalURL), "partner_label", httpadmin.NullString(p.PartnerLabel),
		"sort_order", p.SortOrder, "is_active", p.IsActive, "clicks_30d", p.Clicks30d,
		"created_at", httpadmin.Time(p.CreatedAt), "updated_at", httpadmin.Time(p.UpdatedAt))
}

// Options is GET /shop/options: the categories for selects and the image limits.
func (a *Admin) Options(c fiber.Ctx) error {
	cats, err := a.store.ListCategories(c, false)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(cats))
	for i := range cats {
		items = append(items, a.categoryJSON(&cats[i]))
	}
	return httpadmin.OK(c, jsonx.Obj("categories", items,
		"image", jsonx.Obj("max_kb", imageMaxKB, "min_width", imageMinDim, "min_height", imageMinDim,
			"types", []string{"jpeg", "png", "webp"})))
}

// ---------------------------------------------------------------------------
// Categories

// ListCategories is GET /shop/categories (all, display order).
func (a *Admin) ListCategories(c fiber.Ctx) error {
	cats, err := a.store.ListCategories(c, false)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(cats))
	for i := range cats {
		items = append(items, a.categoryJSON(&cats[i]))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

func (a *Admin) findCategory(c fiber.Ctx) (Category, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return Category{}, httpadmin.NotFound("Category")
	}
	cat, err := a.store.GetCategory(c, uitoa(id), true, false)
	if errors.Is(err, sql.ErrNoRows) {
		return Category{}, httpadmin.NotFound("Category")
	}
	return cat, err
}

// ShowCategory is GET /shop/categories/:id.
func (a *Admin) ShowCategory(c fiber.Ctx) error {
	cat, err := a.findCategory(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("category", a.categoryJSON(&cat)))
}

func (a *Admin) validateCategory(c fiber.Ctx, cur *Category) (CategoryInput, error) {
	rules := form.Translatable(c, "title", true, "max:120")
	rules = append(rules,
		validation.F("slug", "required|string|max:64"),
		validation.F("icon", "nullable|string|max:32"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
	)
	var exceptID uint64
	if cur != nil {
		exceptID = cur.ID
	}
	data, err := form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		slug := strings.TrimSpace(httpadmin.String(in, "slug"))
		if slug == "" {
			return nil
		}
		if !slugRe.MatchString(slug) {
			add("slug", form.Msg(c, "validation.regex", "slug"))
			return nil
		}
		taken, err := a.store.SlugTaken(c, slug, exceptID)
		if err != nil {
			return err
		}
		if taken {
			add("slug", form.Msg(c, "validation.unique", "slug"))
		}
		return nil
	})
	if err != nil {
		return CategoryInput{}, err
	}
	return CategoryInput{
		Slug: strings.TrimSpace(httpadmin.String(data, "slug")), Title: form.ReqJSON(data, "title"),
		Icon: nullable(data, "icon"), SortOrder: form.Int32(data, "sort_order", 0), IsActive: httpadmin.Bool(data, "is_active"),
	}, nil
}

// StoreCategory is POST /shop/categories.
func (a *Admin) StoreCategory(c fiber.Ctx) error {
	in, err := a.validateCategory(c, nil)
	if err != nil {
		return err
	}
	id, err := a.store.CreateCategory(c, in, a.now(c).Time)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_category.create", "shop_category", id)
	cat, err := a.store.GetCategory(c, uitoa(id), true, false)
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("category", a.categoryJSON(&cat)), "Category created.")
}

// UpdateCategory is PUT /shop/categories/:id.
func (a *Admin) UpdateCategory(c fiber.Ctx) error {
	cur, err := a.findCategory(c)
	if err != nil {
		return err
	}
	in, err := a.validateCategory(c, &cur)
	if err != nil {
		return err
	}
	if err := a.store.UpdateCategory(c, cur.ID, in, a.now(c).Time); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_category.update", "shop_category", cur.ID)
	cat, err := a.store.GetCategory(c, uitoa(cur.ID), true, false)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("category", a.categoryJSON(&cat)), "Category updated.")
}

// DestroyCategory is DELETE /shop/categories/:id; 422 in_use while the category has products.
func (a *Admin) DestroyCategory(c fiber.Ctx) error {
	cur, err := a.findCategory(c)
	if err != nil {
		return err
	}
	switch err := a.store.DeleteCategory(c, cur.ID); {
	case errors.Is(err, ErrInUse):
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse,
			"This category still has products. Move or delete them first.")
	case errors.Is(err, sql.ErrNoRows):
		return httpadmin.NotFound("Category")
	case err != nil:
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_category.delete", "shop_category", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Category deleted.")
}

// ---------------------------------------------------------------------------
// Products

// ListProducts is GET /shop/products?category_id=&status=all|active|inactive&q=&page=&per_page=.
func (a *Admin) ListProducts(c fiber.Ctx) error {
	f := ProductFilter{Query: c.Query("q"), Status: c.Query("status")}
	if n := strings.TrimSpace(c.Query("category_id")); n != "" {
		if id, ok := parseID(n); ok {
			f.CategoryID = id
		}
	}
	p := httpadmin.PageOf(c)
	rows, total, err := a.store.ListProducts(c, f, p.PerPage, p.Offset(), httpadmin.Now(c))
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, a.productJSON(&rows[i]))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, total))
}

func (a *Admin) findProduct(c fiber.Ctx) (Product, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return Product{}, httpadmin.NotFound("Product")
	}
	p, err := a.store.GetProduct(c, id, false, httpadmin.Now(c))
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, httpadmin.NotFound("Product")
	}
	return p, err
}

// ShowProduct is GET /shop/products/:id.
func (a *Admin) ShowProduct(c fiber.Ctx) error {
	p, err := a.findProduct(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("product", a.productJSON(&p)))
}

func (a *Admin) validateProduct(c fiber.Ctx, requireImage bool) (ProductInput, *media.Image, error) {
	rules := form.Translatable(c, "title", true, "max:160")
	rules = append(rules,
		validation.F("category_id", "required|integer"),
		validation.F("price_text", "nullable|string|max:80"),
		validation.F("badge", "nullable|string|max:60"),
		validation.F("external_url", "nullable|string|max:1000"),
		validation.F("partner_label", "nullable|string|max:120"),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
		validation.F("remove_image", "nullable"),
	)
	var img *media.Image
	data, err := form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		if u := strings.TrimSpace(httpadmin.String(in, "external_url")); u != "" {
			parsed := LinkURL(sql.NullString{String: u, Valid: true})
			if !form.IsURL(u) || parsed == "" {
				add("external_url", form.Msg(c, "validation.url", "external_url"))
			}
		}
		if id, ok := in.Get("category_id"); ok && id != nil {
			n, ok := parseID(phpval.ToString(id))
			exists := false
			if ok {
				var err error
				if exists, err = a.store.CategoryExists(c, n); err != nil {
					return err
				}
			}
			if !exists {
				add("category_id", form.Msg(c, "validation.exists", "category_id"))
			}
		}
		return nil
	}, form.ImageRule{Field: "image", Required: requireImage, MaxKB: imageMaxKB,
		MinWidth: imageMinDim, MinHeight: imageMinDim}.Check(c, &img))
	if err != nil {
		return ProductInput{}, nil, err
	}
	cat, _ := parseID(httpadmin.String(data, "category_id"))
	return ProductInput{
		CategoryID: cat, Title: form.ReqJSON(data, "title"), PriceText: nullable(data, "price_text"),
		Badge: nullable(data, "badge"), ExternalURL: nullable(data, "external_url"),
		PartnerLabel: nullable(data, "partner_label"), SortOrder: form.Int32(data, "sort_order", 0),
		IsActive: httpadmin.Bool(data, "is_active"),
	}, img, nil
}

func (a *Admin) deleteFile(c fiber.Ctx, rel string) {
	if rel == "" {
		return
	}
	if err := a.disk.Delete(rel); err != nil {
		a.logger.WarnContext(c.Context(), "shop: could not delete image", slog.String("path", rel), slog.String("error", err.Error()))
	}
}

// StoreProduct is POST /shop/products (multipart or JSON; `image` optional).
func (a *Admin) StoreProduct(c fiber.Ctx) error {
	in, img, err := a.validateProduct(c, false)
	if err != nil {
		return err
	}
	if img != nil {
		rel, err := a.disk.Store(a.opt, ImageDir, *img)
		if err != nil {
			return err
		}
		in.ImagePath = sql.NullString{String: rel, Valid: true}
	}
	id, err := a.store.CreateProduct(c, in, a.now(c).Time)
	if err != nil {
		if in.ImagePath.Valid {
			a.deleteFile(c, in.ImagePath.String)
		}
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_product.create", "shop_product", id)
	p, err := a.store.GetProduct(c, id, false, httpadmin.Now(c))
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("product", a.productJSON(&p)), "Product created.")
}

// UpdateProduct is PUT (or multipart POST) /shop/products/:id; a new `image` replaces the file, `remove_image` drops it.
func (a *Admin) UpdateProduct(c fiber.Ctx) error {
	cur, err := a.findProduct(c)
	if err != nil {
		return err
	}
	in, img, err := a.validateProduct(c, false)
	if err != nil {
		return err
	}
	in.ImagePath = cur.ImagePath
	switch {
	case img != nil:
		rel, err := a.disk.Store(a.opt, ImageDir, *img)
		if err != nil {
			return err
		}
		in.ImagePath = sql.NullString{String: rel, Valid: true}
	case httpadmin.Bool(validation.Input(c), "remove_image"):
		in.ImagePath = sql.NullString{}
	}
	if err := a.store.UpdateProduct(c, cur.ID, in, a.now(c).Time); err != nil {
		if img != nil {
			a.deleteFile(c, in.ImagePath.String)
		}
		return err
	}
	if cur.ImagePath.Valid && in.ImagePath != cur.ImagePath {
		a.deleteFile(c, cur.ImagePath.String)
	}
	httpadmin.Audit(c, a.logger, "shop_product.update", "shop_product", cur.ID)
	p, err := a.store.GetProduct(c, cur.ID, false, httpadmin.Now(c))
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("product", a.productJSON(&p)), "Product updated.")
}

// ToggleProduct is POST /shop/products/:id/toggle (is_active).
func (a *Admin) ToggleProduct(c fiber.Ctx) error {
	cur, err := a.findProduct(c)
	if err != nil {
		return err
	}
	if err := a.store.ToggleProduct(c, cur.ID, a.now(c).Time); err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_product.toggle", "shop_product", cur.ID)
	p, err := a.store.GetProduct(c, cur.ID, false, httpadmin.Now(c))
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("product", a.productJSON(&p)), "Status changed.")
}

// DestroyProduct is DELETE /shop/products/:id (the image file goes too).
func (a *Admin) DestroyProduct(c fiber.Ctx) error {
	cur, err := a.findProduct(c)
	if err != nil {
		return err
	}
	if err := a.store.DeleteProduct(c, cur.ID); errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Product")
	} else if err != nil {
		return err
	}
	if cur.ImagePath.Valid {
		a.deleteFile(c, cur.ImagePath.String)
	}
	httpadmin.Audit(c, a.logger, "shop_product.delete", "shop_product", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Product deleted.")
}

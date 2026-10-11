// Package shop is the thin, extendable shop catalog (bloom B-N10-03, deviations D-91): admin-managed categories and
// link-out products, a public read API under /api/v1/shop and the admin CRUD under /api/admin/v1/shop.
//
// Rules (docs/night-bloom/extras.md §0, §2):
//   - the shop is separate from health data: this package imports no health package and no query here joins a health
//     table; teen-mode accounts are excluded through an injected CommercialPolicy only;
//   - no per-user history: link-out clicks are counted per product and Tehran day, without any user id;
//   - the outgoing link carries no user identifier, only a static utm_source=ritme;
//   - no cart, no payment: CB-SHOP-01 extends these tables (external_url stays nullable, link-out rows keep rendering).
package shop

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// UTMSource is the only tracking parameter added to an outgoing link.
const UTMSource = "ritme"

// CommercialPolicy decides whether a user may see commercial content (false for a teen-mode account). The shop never
// reads the answer's reason; routes inject teen.Policy.
type CommercialPolicy interface {
	AllowsCommercial(ctx context.Context, userID uint64) (bool, error)
}

// Category is a shop_categories row.
type Category struct {
	ID        uint64
	Slug      string
	Title     json.RawMessage
	Icon      sql.NullString
	SortOrder int32
	IsActive  bool
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
	// Products is the number of products shown with the row (public: active only; admin: all).
	Products int64
}

// Product is a shop_products row.
type Product struct {
	ID           uint64
	CategoryID   uint64
	CategorySlug string
	Title        json.RawMessage
	ImagePath    sql.NullString
	PriceText    sql.NullString
	Badge        sql.NullString
	ExternalURL  sql.NullString
	PartnerLabel sql.NullString
	SortOrder    int32
	IsActive     bool
	CreatedAt    sql.NullTime
	UpdatedAt    sql.NullTime
	// Clicks30d is the link-out clicks of the last 30 days (admin list/show only).
	Clicks30d int64
}

// Store reads and writes the shop tables with plain SQL (nothing else is touched).
type Store struct{ db *sql.DB }

// NewStore wraps db.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

const categoryCols = "c.id, c.slug, c.title, c.icon, c.sort_order, c.is_active, c.created_at, c.updated_at"

func scanCategory(row interface{ Scan(...any) error }, withCount bool) (Category, error) {
	var c Category
	var title string
	dest := []any{&c.ID, &c.Slug, &title, &c.Icon, &c.SortOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt}
	if withCount {
		dest = append(dest, &c.Products)
	}
	if err := row.Scan(dest...); err != nil {
		return c, err
	}
	c.Title = json.RawMessage(title)
	return c, nil
}

// ListCategories returns the categories in display order with their product counts (activeOnly: public view, only
// active categories and active products). CB-SHOP-02: the public view lists root categories only, so the
// sub-categories CB-SHOP-01 added (parent_id) stay out of bloom's two-tab strip; the catalog API serves the tree.
func (s *Store) ListCategories(ctx context.Context, activeOnly bool) ([]Category, error) {
	q := `SELECT ` + categoryCols + `,
	  (SELECT COUNT(*) FROM shop_products p WHERE p.category_id = c.id AND (? = 0 OR p.is_active = 1)) AS n
	  FROM shop_categories c WHERE (? = 0 OR (c.is_active = 1 AND c.parent_id IS NULL)) ORDER BY c.sort_order, c.id`
	flag := 0
	if activeOnly {
		flag = 1
	}
	rows, err := s.db.QueryContext(ctx, q, flag, flag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		c, err := scanCategory(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCategory is one category by id (admin) or by slug-or-id (public, activeOnly).
func (s *Store) GetCategory(ctx context.Context, ref string, byID, activeOnly bool) (Category, error) {
	col := "c.slug"
	if byID {
		col = "c.id"
	}
	q := `SELECT ` + categoryCols + `,
	  (SELECT COUNT(*) FROM shop_products p WHERE p.category_id = c.id AND (? = 0 OR p.is_active = 1)) AS n
	  FROM shop_categories c WHERE ` + col + ` = ? AND (? = 0 OR c.is_active = 1) LIMIT 1`
	flag := 0
	if activeOnly {
		flag = 1
	}
	return scanCategory(s.db.QueryRowContext(ctx, q, flag, ref, flag), true)
}

// SlugTaken reports whether another category uses slug.
func (s *Store) SlugTaken(ctx context.Context, slug string, exceptID uint64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM shop_categories WHERE slug = ? AND id <> ?", slug, exceptID).Scan(&n)
	return n > 0, err
}

// CategoryInput is the writable part of a category.
type CategoryInput struct {
	Slug      string
	Title     json.RawMessage
	Icon      sql.NullString
	SortOrder int32
	IsActive  bool
}

// CreateCategory inserts and returns the id.
func (s *Store) CreateCategory(ctx context.Context, in CategoryInput, now time.Time) (uint64, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO shop_categories (slug, title, icon, sort_order, is_active, created_at, updated_at) VALUES (?,?,?,?,?,?,?)",
		in.Slug, string(in.Title), in.Icon, in.SortOrder, in.IsActive, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err //nolint:gosec // G115: auto-increment ids are positive
}

// UpdateCategory rewrites the writable columns.
func (s *Store) UpdateCategory(ctx context.Context, id uint64, in CategoryInput, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE shop_categories SET slug=?, title=?, icon=?, sort_order=?, is_active=?, updated_at=? WHERE id=?",
		in.Slug, string(in.Title), in.Icon, in.SortOrder, in.IsActive, now, id)
	return err
}

// ErrInUse is returned when deleting a category that still has products.
var ErrInUse = errors.New("shop: category has products")

// DeleteCategory removes an empty category.
func (s *Store) DeleteCategory(ctx context.Context, id uint64) error {
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM shop_products WHERE category_id = ?", id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrInUse
	}
	res, err := s.db.ExecContext(ctx, "DELETE FROM shop_categories WHERE id = ?", id)
	if err != nil {
		return err
	}
	if k, _ := res.RowsAffected(); k == 0 {
		return sql.ErrNoRows
	}
	return nil
}

const productCols = `p.id, p.category_id, c.slug, p.title, p.image_path, p.price_text, p.badge, p.external_url,
	p.partner_label, p.sort_order, p.is_active, p.created_at, p.updated_at,
	(SELECT COALESCE(SUM(k.clicks), 0) FROM shop_click_counts k WHERE k.product_id = p.id AND k.day >= ?) AS clicks`

func scanProduct(row interface{ Scan(...any) error }) (Product, error) {
	var p Product
	var title string
	err := row.Scan(&p.ID, &p.CategoryID, &p.CategorySlug, &title, &p.ImagePath, &p.PriceText, &p.Badge, &p.ExternalURL,
		&p.PartnerLabel, &p.SortOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.Clicks30d)
	p.Title = json.RawMessage(title)
	return p, err
}

// ProductFilter narrows a product list.
type ProductFilter struct {
	CategoryID uint64 // 0 = any
	ActiveOnly bool   // public: active product in an active category
	Status     string // admin: "", "active", "inactive"
	Query      string // admin: LIKE over the title JSON and partner label
}

func (f ProductFilter) where() (string, []any) {
	w, args := " WHERE 1=1", []any{}
	if f.CategoryID > 0 {
		w += " AND p.category_id = ?"
		args = append(args, f.CategoryID)
	}
	if f.ActiveOnly {
		w += " AND p.is_active = 1 AND c.is_active = 1"
	}
	switch f.Status {
	case "active":
		w += " AND p.is_active = 1"
	case "inactive":
		w += " AND p.is_active = 0"
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
		w += " AND (p.title LIKE ? OR IFNULL(p.partner_label,'') LIKE ?)"
		args = append(args, like, like)
	}
	return w, args
}

func since30(now time.Time) civildate.Date { return civildate.InTehran(now.AddDate(0, 0, -30)) }

// ListProducts returns one page and the total.
func (s *Store) ListProducts(ctx context.Context, f ProductFilter, limit, offset int, now time.Time) ([]Product, int, error) {
	w, args := f.where()
	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM shop_products p JOIN shop_categories c ON c.id = p.category_id"+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	all := append([]any{since30(now)}, args...)
	all = append(all, limit, offset)
	rows, err := s.db.QueryContext(ctx, "SELECT "+productCols+ //nolint:gosec // G202: only constants and a fixed WHERE built from ? placeholders
		" FROM shop_products p JOIN shop_categories c ON c.id = p.category_id"+w+
		" ORDER BY c.sort_order, p.sort_order, p.id LIMIT ? OFFSET ?", all...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// GetProduct is one product by id (activeOnly: public rules).
func (s *Store) GetProduct(ctx context.Context, id uint64, activeOnly bool, now time.Time) (Product, error) {
	w := " WHERE p.id = ?"
	if activeOnly {
		w += " AND p.is_active = 1 AND c.is_active = 1"
	}
	return scanProduct(s.db.QueryRowContext(ctx, "SELECT "+productCols+
		" FROM shop_products p JOIN shop_categories c ON c.id = p.category_id"+w+" LIMIT 1", since30(now), id))
}

// ProductInput is the writable part of a product.
type ProductInput struct {
	CategoryID   uint64
	Title        json.RawMessage
	ImagePath    sql.NullString
	PriceText    sql.NullString
	Badge        sql.NullString
	ExternalURL  sql.NullString
	PartnerLabel sql.NullString
	SortOrder    int32
	IsActive     bool
}

// CategoryExists reports whether a category id exists.
func (s *Store) CategoryExists(ctx context.Context, id uint64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM shop_categories WHERE id = ?", id).Scan(&n)
	return n > 0, err
}

// CreateProduct inserts and returns the id.
func (s *Store) CreateProduct(ctx context.Context, in ProductInput, now time.Time) (uint64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO shop_products (category_id, title, image_path, price_text, badge,
		external_url, partner_label, sort_order, is_active, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		in.CategoryID, string(in.Title), in.ImagePath, in.PriceText, in.Badge, in.ExternalURL, in.PartnerLabel,
		in.SortOrder, in.IsActive, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err //nolint:gosec // G115: auto-increment ids are positive
}

// UpdateProduct rewrites the writable columns.
func (s *Store) UpdateProduct(ctx context.Context, id uint64, in ProductInput, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE shop_products SET category_id=?, title=?, image_path=?, price_text=?, badge=?,
		external_url=?, partner_label=?, sort_order=?, is_active=?, updated_at=? WHERE id=?`,
		in.CategoryID, string(in.Title), in.ImagePath, in.PriceText, in.Badge, in.ExternalURL, in.PartnerLabel,
		in.SortOrder, in.IsActive, now, id)
	return err
}

// ToggleProduct flips is_active.
func (s *Store) ToggleProduct(ctx context.Context, id uint64, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE shop_products SET is_active = 1 - is_active, updated_at = ? WHERE id = ?", now, id)
	return err
}

// DeleteProduct removes a product (its click counters go with it).
func (s *Store) DeleteProduct(ctx context.Context, id uint64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM shop_products WHERE id = ?", id)
	if err != nil {
		return err
	}
	if k, _ := res.RowsAffected(); k == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CountClick adds one link-out click to the product's counter for day. No user id, session or IP is stored.
func (s *Store) CountClick(ctx context.Context, productID uint64, day civildate.Date) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO shop_click_counts (product_id, day, clicks) VALUES (?, ?, 1)
		ON DUPLICATE KEY UPDATE clicks = clicks + 1`, productID, day)
	return err
}

// LinkURL is the outgoing link for a product: its external_url plus a static utm_source (never a user identifier);
// "" when the product has no link-out or the stored URL is not http(s).
func LinkURL(external sql.NullString) string {
	if !external.Valid || external.String == "" {
		return ""
	}
	u, err := url.Parse(external.String)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	q := u.Query()
	if q.Get("utm_source") == "" {
		q.Set("utm_source", UTMSource)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func uitoa(n uint64) string { return strconv.FormatUint(n, 10) }

func parseID(s string) (uint64, bool) {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n, err == nil && n > 0
}

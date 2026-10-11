package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/shop/store"
)

// MaxRows caps the products one category (subtree) loads for in-memory filtering, faceting and sorting. The MVP
// catalog is admin-curated and far smaller; past this the list shows the most-bought MaxRows.
const MaxRows = 1000

// ErrNotFound is a missing / inactive category or product, or a link-out product (no variants).
var ErrNotFound = errors.New("catalog: not found")

// Service loads catalog data through the sqlc store.
type Service struct{ q *store.Queries }

// NewService wraps db.
func NewService(db store.DBTX) *Service { return &Service{q: store.New(db)} }

// Tree is the active category tree.
type Tree struct {
	byID     map[uint64]*store.ShopCategory
	bySlug   map[string]*store.ShopCategory
	children map[uint64][]*store.ShopCategory
	roots    []*store.ShopCategory
}

// Tree loads the active categories.
func (s *Service) Tree(ctx context.Context) (*Tree, error) {
	cats, err := s.q.ListActiveCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("catalog: categories: %w", err)
	}
	t := &Tree{byID: map[uint64]*store.ShopCategory{}, bySlug: map[string]*store.ShopCategory{},
		children: map[uint64][]*store.ShopCategory{}}
	for i := range cats {
		c := &cats[i]
		t.byID[c.ID], t.bySlug[c.Slug] = c, c
	}
	for i := range cats {
		c := &cats[i]
		if !c.ParentID.Valid {
			t.roots = append(t.roots, c)
		} else if _, ok := t.byID[uint64(c.ParentID.Int64)]; ok { //nolint:gosec // G115: ids are positive
			t.children[uint64(c.ParentID.Int64)] = append(t.children[uint64(c.ParentID.Int64)], c) //nolint:gosec // G115
		}
	}
	return t, nil
}

// Roots are the active top-level categories (layette, beauty…).
func (t *Tree) Roots() []*store.ShopCategory { return t.roots }

// Children are the active direct sub-categories of id.
func (t *Tree) Children(id uint64) []*store.ShopCategory { return t.children[id] }

// BySlug is an active category.
func (t *Tree) BySlug(slug string) (*store.ShopCategory, bool) { c, ok := t.bySlug[slug]; return c, ok }

// Parent is the parent's slug ("" for a root).
func (t *Tree) Parent(c *store.ShopCategory) string {
	if !c.ParentID.Valid {
		return ""
	}
	if p, ok := t.byID[uint64(c.ParentID.Int64)]; ok { //nolint:gosec // G115: ids are positive
		return p.Slug
	}
	return ""
}

// Subtree is id plus every active descendant (cycle-safe).
func (t *Tree) Subtree(id uint64) []uint64 {
	out := []uint64{id}
	for i := 0; i < len(out); i++ {
		for _, c := range t.children[out[i]] {
			if !slices.Contains(out, c.ID) {
				out = append(out, c.ID)
			}
		}
	}
	return out
}

// All is every active category id.
func (t *Tree) All() []uint64 {
	out := make([]uint64, 0, len(t.byID))
	for id := range t.byID {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// Items loads the catalog products of the given categories, most bought first.
func (s *Service) Items(ctx context.Context, t *Tree, categoryIDs []uint64) ([]*Item, error) {
	if len(categoryIDs) == 0 {
		return []*Item{}, nil
	}
	rows, err := s.q.ListCatalogProductsInCategories(ctx, store.ListCatalogProductsInCategoriesParams{
		CategoryIds: categoryIDs, Limit: MaxRows})
	if err != nil {
		return nil, fmt.Errorf("catalog: products: %w", err)
	}
	return s.hydrate(ctx, t, rows)
}

// Product is one catalog product (ErrNotFound for a missing, inactive or link-out product).
func (s *Service) Product(ctx context.Context, t *Tree, id uint64) (*Item, error) {
	rows, err := s.q.ListCatalogProductsByIDs(ctx, []uint64{id})
	if err != nil {
		return nil, fmt.Errorf("catalog: product: %w", err)
	}
	items, err := s.hydrate(ctx, t, rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items[0], nil
}

// hydrate attaches variants, sellers and category slugs; rows whose category is not in the active tree are dropped.
func (s *Service) hydrate(ctx context.Context, t *Tree, rows []store.ShopProduct) ([]*Item, error) {
	items := make([]*Item, 0, len(rows))
	ids, sellerIDs := []uint64{}, []uint64{}
	byID := map[uint64]*Item{}
	for _, p := range rows {
		cat, ok := t.byID[p.CategoryID]
		if !ok {
			continue
		}
		it := &Item{Product: p, Category: cat.Slug}
		items = append(items, it)
		ids = append(ids, p.ID)
		byID[p.ID] = it
		if p.SellerID.Valid && !slices.Contains(sellerIDs, uint64(p.SellerID.Int64)) { //nolint:gosec // G115
			sellerIDs = append(sellerIDs, uint64(p.SellerID.Int64)) //nolint:gosec // G115: ids are positive
		}
	}
	if len(ids) == 0 {
		return items, nil
	}
	variants, err := s.q.ListActiveVariantsByProducts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("catalog: variants: %w", err)
	}
	for _, v := range variants {
		if it, ok := byID[v.ProductID]; ok {
			it.Variants = append(it.Variants, v)
		}
	}
	if len(sellerIDs) > 0 {
		sellers, err := s.q.ListSellersByIDs(ctx, sellerIDs)
		if err != nil {
			return nil, fmt.Errorf("catalog: sellers: %w", err)
		}
		for i := range sellers {
			for _, it := range items {
				if it.Product.SellerID.Valid && uint64(it.Product.SellerID.Int64) == sellers[i].ID { //nolint:gosec // G115
					it.Seller = &sellers[i]
				}
			}
		}
	}
	return items, nil
}

// BoughtTogether is up to n products ordered together with it (most shared orders first), topped up with the
// best sellers of its category.
func (s *Service) BoughtTogether(ctx context.Context, t *Tree, it *Item, n int) ([]*Item, error) {
	pairs, err := s.q.ListBoughtTogether(ctx, store.ListBoughtTogetherParams{
		ProductID: sql.NullInt64{Int64: int64(it.Product.ID), Valid: true}, Limit: int32(n * 2)}) //nolint:gosec // G115: small n
	if err != nil {
		return nil, fmt.Errorf("catalog: bought together: %w", err)
	}
	ids := []uint64{}
	for _, p := range pairs {
		if p.ProductID.Valid {
			ids = append(ids, uint64(p.ProductID.Int64)) //nolint:gosec // G115: ids are positive
		}
	}
	out := []*Item{}
	if len(ids) > 0 {
		rows, err := s.q.ListCatalogProductsByIDs(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("catalog: bought together: %w", err)
		}
		found, err := s.hydrate(ctx, t, rows)
		if err != nil {
			return nil, err
		}
		for _, id := range ids { // keep the shared-orders ranking
			for _, f := range found {
				if f.Product.ID == id && len(out) < n {
					out = append(out, f)
				}
			}
		}
	}
	if len(out) < n {
		same, err := s.Items(ctx, t, []uint64{it.Product.CategoryID})
		if err != nil {
			return nil, err
		}
		for _, f := range same {
			if len(out) >= n {
				break
			}
			if f.Product.ID != it.Product.ID && !slices.ContainsFunc(out, func(o *Item) bool { return o.Product.ID == f.Product.ID }) {
				out = append(out, f)
			}
		}
	}
	return out, nil
}

// Reviews is one page of published reviews and their total.
func (s *Service) Reviews(ctx context.Context, productID uint64, limit, offset int) ([]store.ShopProductReview, int, error) {
	total, err := s.q.CountPublishedReviews(ctx, productID)
	if err != nil {
		return nil, 0, fmt.Errorf("catalog: reviews: %w", err)
	}
	rows, err := s.q.ListPublishedReviewsPage(ctx, store.ListPublishedReviewsPageParams{
		ProductID: productID, Limit: int32(limit), Offset: int32(offset)}) //nolint:gosec // G115: bounded paging
	if err != nil {
		return nil, 0, fmt.Errorf("catalog: reviews: %w", err)
	}
	return rows, int(total), nil
}

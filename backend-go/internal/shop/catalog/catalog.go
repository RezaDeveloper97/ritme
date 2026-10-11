// Package catalog is the internal shop's read API (CB-SHOP-02): shop homes (layette / beauty roots with their
// sub-categories and best-seller sections), the product list with filters (size, color, volume, material, skin type,
// price) and sort, the product detail (variants, stock, seller, attributes, reviews, bought-together) and the review
// list. It sits next to bloom's link-out catalog (package shop, B-N10-03) and never changes its routes or shapes:
// link-out rows (no variants) are not catalog products.
//
// Rules (docs/canvas-build/shop.md): no health package is imported and no query joins a health table; teen-mode
// accounts get 403 teen_commercial_blocked through an injected CommercialPolicy; money is integer toman; translatable
// columns are picked for the request locale with the default-language fallback, and filter VALUES are the
// default-language text (stable across UI languages) while LABELS are localized.
package catalog

import (
	"bytes"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/shop/store"
)

// Sort keys of the product list (default SortPopular, the board's «پرفروش‌ترین»).
const (
	SortPopular   = "popular"
	SortRating    = "rating"
	SortNewest    = "newest"
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
)

// Sorts lists the accepted sort keys (OpenAPI enum).
var Sorts = []string{SortPopular, SortRating, SortNewest, SortPriceAsc, SortPriceDesc}

// Locale is the request language and the default language (fallback for translatable columns).
type Locale struct{ Code, Default string }

// Pick returns a translatable JSON value (object keyed by language code, or a plain string) for the locale.
func (l Locale) Pick(raw string) string {
	b := bytes.TrimSpace([]byte(raw))
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return ""
	}
	if b[0] == '"' {
		var s string
		_ = json.Unmarshal(b, &s)
		return s
	}
	return i18n.PickString(b, l.Code, l.Default)
}

// Value is the default-language text of a translatable value: the stable filter key.
func (l Locale) Value(raw string) string {
	return Locale{Code: l.Default, Default: l.Default}.Pick(raw)
}

// Item is a catalog product with its active variants and seller, plus the derived card price.
type Item struct {
	Product  store.ShopProduct
	Variants []store.ShopProductVariant
	Seller   *store.ShopSeller
	Category string // category slug
}

// cheapest is the variant the card price shows: the cheapest in stock, else the cheapest.
func (it *Item) cheapest() *store.ShopProductVariant {
	var best *store.ShopProductVariant
	for i := range it.Variants {
		v := &it.Variants[i]
		switch {
		case best == nil:
			best = v
		case (v.Stock > 0) != (best.Stock > 0):
			if v.Stock > 0 {
				best = v
			}
		case v.Price < best.Price:
			best = v
		}
	}
	return best
}

// Price is the card price (0 without variants).
func (it *Item) Price() uint64 {
	if v := it.cheapest(); v != nil {
		return v.Price
	}
	return 0
}

// CompareAt is the struck-through price of the card variant (nil unless above the price).
func (it *Item) CompareAt() any {
	if v := it.cheapest(); v != nil && v.CompareAtPrice.Valid && v.CompareAtPrice.Int64 > 0 &&
		uint64(v.CompareAtPrice.Int64) > v.Price {
		return v.CompareAtPrice.Int64
	}
	return nil
}

// InStock reports whether any active variant has stock.
func (it *Item) InStock() bool {
	return slices.ContainsFunc(it.Variants, func(v store.ShopProductVariant) bool { return v.Stock > 0 })
}

// Rating is the cached average as a number (nil without reviews).
func (it *Item) Rating() any {
	if !it.Product.RatingAvg.Valid || it.Product.RatingCount == 0 {
		return nil
	}
	f, err := strconv.ParseFloat(it.Product.RatingAvg.String, 64)
	if err != nil {
		return nil
	}
	return f
}

func (it *Item) rating() float64 {
	if f, ok := it.Rating().(float64); ok {
		return f
	}
	return 0
}

// Attributes decodes the attributes JSON object (key → translatable value or string).
func Attributes(raw string) map[string]json.RawMessage {
	m := map[string]json.RawMessage{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &m)
	}
	return m
}

// SkinTypes decodes the skin_types JSON array of keys (oily, combination, normal, dry, sensitive…).
func SkinTypes(raw string) []string {
	out := []string{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

// Filter narrows a product list. Empty slices / zero bounds mean "any". Multi-valued filters match any value.
type Filter struct {
	Query     string
	Sizes     []string
	Colors    []string
	Volumes   []string
	Materials []string
	SkinTypes []string
	PriceMin  uint64
	PriceMax  uint64 // 0 = no upper bound
	Sort      string
}

func anyOf(set []string, v string) bool { return len(set) == 0 || (v != "" && slices.Contains(set, v)) }

// variantMatches applies the variant-level filters together (one variant must satisfy size, color, volume and price).
func (f *Filter) variantMatches(l Locale, v *store.ShopProductVariant) bool {
	return anyOf(f.Sizes, l.Value(v.Size.String)) && anyOf(f.Colors, l.Value(v.Color.String)) &&
		anyOf(f.Volumes, l.Value(v.Volume.String)) &&
		v.Price >= f.PriceMin && (f.PriceMax == 0 || v.Price <= f.PriceMax)
}

// Matches reports whether the item passes every filter.
func (f *Filter) Matches(l Locale, it *Item) bool {
	if q := strings.ToLower(strings.TrimSpace(f.Query)); q != "" {
		title := strings.ToLower(l.Pick(it.Product.Title) + " " + l.Value(it.Product.Title))
		if !strings.Contains(title, q) {
			return false
		}
	}
	if len(f.Materials) > 0 && !anyOf(f.Materials, l.Value(string(Attributes(it.Product.Attributes.String)["material"]))) {
		return false
	}
	if len(f.SkinTypes) > 0 && !slices.ContainsFunc(SkinTypes(it.Product.SkinTypes.String), func(s string) bool {
		return slices.Contains(f.SkinTypes, s)
	}) {
		return false
	}
	for i := range it.Variants {
		if f.variantMatches(l, &it.Variants[i]) {
			return true
		}
	}
	return false
}

// Apply filters and sorts items (stable; ties by id descending, newest first).
func (f *Filter) Apply(l Locale, items []*Item) []*Item {
	out := make([]*Item, 0, len(items))
	for _, it := range items {
		if f.Matches(l, it) {
			out = append(out, it)
		}
	}
	less := func(a, b *Item) bool {
		switch f.Sort {
		case SortPriceAsc:
			if a.Price() != b.Price() {
				return a.Price() < b.Price()
			}
		case SortPriceDesc:
			if a.Price() != b.Price() {
				return a.Price() > b.Price()
			}
		case SortRating:
			if a.rating() != b.rating() {
				return a.rating() > b.rating()
			}
			if a.Product.RatingCount != b.Product.RatingCount {
				return a.Product.RatingCount > b.Product.RatingCount
			}
		case SortNewest:
			at, bt := a.Product.CreatedAt.Time, b.Product.CreatedAt.Time
			if !at.Equal(bt) {
				return at.After(bt)
			}
		default:
			if a.Product.PurchaseCount != b.Product.PurchaseCount {
				return a.Product.PurchaseCount > b.Product.PurchaseCount
			}
		}
		return a.Product.ID > b.Product.ID
	}
	sort.SliceStable(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

// FacetOption is one filter choice: value (default-language key), localized label and the number of products.
type FacetOption struct {
	Value string
	Label string
	Count int
}

// Facets are the filter choices of a product set (computed before filtering, so options stay stable).
type Facets struct {
	Sizes, Colors, Volumes, Materials, SkinTypes []FacetOption
	PriceMin, PriceMax                           uint64
}

type facetAcc struct {
	order []string
	opts  map[string]*FacetOption
}

func (a *facetAcc) add(value, label string, seen map[string]bool) {
	if value == "" || seen[value] {
		return
	}
	seen[value] = true
	if a.opts == nil {
		a.opts = map[string]*FacetOption{}
	}
	o, ok := a.opts[value]
	if !ok {
		o = &FacetOption{Value: value, Label: label}
		a.opts[value] = o
		a.order = append(a.order, value)
	}
	o.Count++
}

func (a *facetAcc) list() []FacetOption {
	out := make([]FacetOption, 0, len(a.order))
	for _, v := range a.order {
		out = append(out, *a.opts[v])
	}
	return out
}

// BuildFacets collects the options in variant display order (sort_order) of the first product carrying them.
func BuildFacets(l Locale, items []*Item) Facets {
	var sizes, colors, volumes, materials, skins facetAcc
	var f Facets
	first := true
	for _, it := range items {
		sSeen, cSeen, vSeen, mSeen, kSeen := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
		for i := range it.Variants {
			v := &it.Variants[i]
			sizes.add(l.Value(v.Size.String), l.Pick(v.Size.String), sSeen)
			colors.add(l.Value(v.Color.String), l.Pick(v.Color.String), cSeen)
			volumes.add(l.Value(v.Volume.String), l.Pick(v.Volume.String), vSeen)
			if first || v.Price < f.PriceMin {
				f.PriceMin = v.Price
			}
			if first || v.Price > f.PriceMax {
				f.PriceMax = v.Price
			}
			first = false
		}
		mat := string(Attributes(it.Product.Attributes.String)["material"])
		materials.add(l.Value(mat), l.Pick(mat), mSeen)
		for _, k := range SkinTypes(it.Product.SkinTypes.String) {
			skins.add(k, k, kSeen) // keys; the client labels them
		}
	}
	f.Sizes, f.Colors, f.Volumes, f.Materials, f.SkinTypes = sizes.list(), colors.list(), volumes.list(), materials.list(), skins.list()
	return f
}

// SplitList parses a comma-separated query value into trimmed, non-empty, unique values (at most 20).
func SplitList(s string) []string {
	out := []string{}
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" && !slices.Contains(out, p) && len(out) < 20 {
			out = append(out, p)
		}
	}
	return out
}

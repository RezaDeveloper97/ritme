package order

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/shop/catalog"
	"github.com/ritme/backend-go/internal/shop/store"
)

// Line statuses of a cart line.
const (
	LineOK          = "ok"
	LineOutOfStock  = "out_of_stock" // stock < quantity (stock may still be > 0)
	LineUnavailable = "unavailable"  // variant / product / category / seller inactive, or no seller (link-out data)
)

// Discount kinds (shop_discount_codes.kind).
const (
	DiscountPercent = "percent"
	DiscountFixed   = "fixed"
)

// Line is one cart line with its seller and derived status.
type Line struct {
	store.ListCartLinesRow
	Seller    *store.ShopSeller
	Status    string
	LineTotal uint64
}

// Group is one seller's part of the cart: one shipment at checkout.
type Group struct {
	Seller      *store.ShopSeller
	Lines       []*Line
	ItemsTotal  uint64 // purchasable lines only
	ItemCount   int
	ShippingFee uint64
}

// Cart is the priced cart. Totals count LineOK lines only; unavailable lines are listed apart.
type Cart struct {
	ID             uint64
	Groups         []*Group
	Unavailable    []*Line
	ItemsTotal     uint64
	ItemCount      int
	Discount       *store.ShopDiscountCode
	DiscountAmount uint64
	DiscountError  string // error code when the applied code no longer applies ("" when fine)
	discountUses   int64  // her non-cancelled orders with the code
}

// totals recomputes group and cart totals, fees and the discount from the lines' statuses.
func (c *Cart) totals(now time.Time) {
	c.ItemsTotal, c.ItemCount = 0, 0
	for _, g := range c.Groups {
		g.ItemsTotal, g.ItemCount = 0, 0
		for _, l := range g.Lines {
			if l.Status == LineOK {
				g.ItemsTotal += l.LineTotal
				g.ItemCount += int(l.Quantity)
			}
		}
		g.ShippingFee = shippingFee(g.Seller, g.ItemsTotal)
		c.ItemsTotal += g.ItemsTotal
		c.ItemCount += g.ItemCount
	}
	c.DiscountAmount, c.DiscountError = 0, ""
	if c.Discount != nil {
		c.DiscountAmount, c.DiscountError = evaluateDiscount(c.Discount, c.ItemsTotal, now, c.discountUses)
	}
}

// discountID is the applied code's id (NULL without one).
func (c *Cart) discountID() sql.NullInt64 {
	if c.Discount == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(c.Discount.ID), Valid: true} //nolint:gosec // G115: auto-increment id
}

// settle marks every grouped line purchasable (its stock is reserved) and recomputes the totals.
func (c *Cart) settle(now time.Time) {
	for _, g := range c.Groups {
		for _, l := range g.Lines {
			l.Status = LineOK
		}
	}
	c.totals(now)
}

// Lines are all priced lines in group order, then the unavailable ones.
func (c *Cart) Lines() []*Line {
	var out []*Line
	for _, g := range c.Groups {
		out = append(out, g.Lines...)
	}
	return append(out, c.Unavailable...)
}

// Purchasable reports whether every line can be ordered as is.
func (c *Cart) Purchasable() bool {
	if len(c.Unavailable) > 0 {
		return false
	}
	for _, g := range c.Groups {
		for _, l := range g.Lines {
			if l.Status != LineOK {
				return false
			}
		}
	}
	return true
}

// ShippingTotal is the sum of the groups' fees.
func (c *Cart) ShippingTotal() uint64 {
	var n uint64
	for _, g := range c.Groups {
		n += g.ShippingFee
	}
	return n
}

// buildCart prices rows: groups keep the order of each seller's first line.
func buildCart(cartID uint64, rows []store.ListCartLinesRow, sellers map[uint64]*store.ShopSeller) *Cart {
	c := &Cart{ID: cartID}
	byID := map[uint64]*Group{}
	for i := range rows {
		r := rows[i]
		l := &Line{ListCartLinesRow: r, Status: LineOK, LineTotal: r.Price * uint64(r.Quantity)}
		if r.SellerID.Valid {
			l.Seller = sellers[uint64(r.SellerID.Int64)] //nolint:gosec // G115: ids are positive
		}
		switch {
		case !r.VariantActive || !r.ProductActive || !r.CategoryActive || l.Seller == nil || !l.Seller.IsActive:
			l.Status = LineUnavailable
			c.Unavailable = append(c.Unavailable, l)
			continue
		case uint32(r.Quantity) > r.Stock:
			l.Status = LineOutOfStock
		}
		g, ok := byID[l.Seller.ID]
		if !ok {
			g = &Group{Seller: l.Seller}
			byID[l.Seller.ID] = g
			c.Groups = append(c.Groups, g)
		}
		g.Lines = append(g.Lines, l)
	}
	c.totals(time.Time{})
	return c
}

// shippingFee is the seller's flat fee, waived from free_shipping_over; nothing to ship costs nothing.
func shippingFee(s *store.ShopSeller, itemsTotal uint64) uint64 {
	if itemsTotal == 0 {
		return 0
	}
	if s.FreeShippingOver.Valid && itemsTotal >= uint64(s.FreeShippingOver.Int64) { //nolint:gosec // G115: money ≥ 0
		return 0
	}
	return s.ShippingFee
}

// freeShippingRemaining is how much more of this seller's items waives the fee (nil without a threshold or when met).
func freeShippingRemaining(g *Group) any {
	if !g.Seller.FreeShippingOver.Valid || g.ShippingFee == 0 {
		return nil
	}
	return uint64(g.Seller.FreeShippingOver.Int64) - g.ItemsTotal //nolint:gosec // G115: fee > 0 ⇒ total < threshold
}

// codAllowed reports whether a seller accepts pay-on-delivery for a shipment of this value (items + fee).
func codAllowed(g *Group) bool {
	return !g.Seller.CodMaxTotal.Valid || g.ItemsTotal+g.ShippingFee <= uint64(g.Seller.CodMaxTotal.Int64) //nolint:gosec // G115
}

// evaluateDiscount prices code d against an items total. It returns the amount, or an error code
// (discount_invalid | discount_exhausted | discount_used | discount_min_total) and 0. userUses counts the user's
// non-cancelled orders that used the code.
func evaluateDiscount(d *store.ShopDiscountCode, itemsTotal uint64, now time.Time, userUses int64) (uint64, string) {
	switch {
	case !d.IsActive, d.StartsAt.Valid && now.Before(d.StartsAt.Time), d.EndsAt.Valid && now.After(d.EndsAt.Time):
		return 0, "discount_invalid"
	case d.MaxUses.Valid && int64(d.UsedCount) >= int64(d.MaxUses.Int32):
		return 0, "discount_exhausted"
	case d.PerUserLimit.Valid && d.PerUserLimit.Int16 > 0 && userUses >= int64(d.PerUserLimit.Int16):
		return 0, "discount_used"
	case itemsTotal == 0 || itemsTotal < d.MinItemsTotal:
		return 0, "discount_min_total"
	}
	var amount uint64
	switch d.Kind {
	case DiscountPercent:
		amount = itemsTotal * min(d.Value, 100) / 100
	case DiscountFixed:
		amount = d.Value
	default:
		return 0, "discount_invalid"
	}
	if d.MaxDiscount.Valid && amount > uint64(d.MaxDiscount.Int64) { //nolint:gosec // G115: money ≥ 0
		amount = uint64(d.MaxDiscount.Int64) //nolint:gosec // G115
	}
	return min(amount, itemsTotal), ""
}

// variantParts are a line's non-empty size / color / volume translatable values.
func variantParts(size, color, volume string) []string {
	var parts []string
	for _, p := range []string{size, color, volume} {
		if b := bytes.TrimSpace([]byte(p)); len(b) > 0 && !bytes.Equal(b, []byte("null")) {
			parts = append(parts, p)
		}
	}
	return parts
}

// variantLabel joins the parts in one locale: «۳ تا ۶ ماه · شیری».
func variantLabel(l catalog.Locale, parts []string) string {
	var out []string
	for _, p := range parts {
		if s := l.Pick(p); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, " · ")
}

// variantLabelSnapshot is the translatable label stored on an order item: one entry per language code that any part
// carries (the language list is the data's, not the code's), each falling back to the default language.
func variantLabelSnapshot(defaultCode string, parts []string) (string, bool) {
	if len(parts) == 0 {
		return "", false
	}
	codes := map[string]bool{defaultCode: true}
	for _, p := range parts {
		var m map[string]json.RawMessage
		if json.Unmarshal([]byte(p), &m) == nil {
			for k := range m {
				codes[k] = true
			}
		}
	}
	keys := make([]string, 0, len(codes))
	for k := range codes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := map[string]string{}
	for _, k := range keys {
		if s := variantLabel(catalog.Locale{Code: k, Default: defaultCode}, parts); s != "" {
			out[k] = s
		}
	}
	if len(out) == 0 {
		return "", false
	}
	b, _ := json.Marshal(out)
	return string(b), true
}

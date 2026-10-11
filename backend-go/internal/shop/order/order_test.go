package order

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/shop/catalog"
	"github.com/ritme/backend-go/internal/shop/store"
)

var now = time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)

func TestEvaluateDiscount(t *testing.T) {
	base := store.ShopDiscountCode{Kind: DiscountPercent, Value: 10, IsActive: true, MinItemsTotal: 100000,
		MaxDiscount: sql.NullInt64{Int64: 50000, Valid: true}, MaxUses: sql.NullInt32{Int32: 5, Valid: true},
		PerUserLimit: sql.NullInt16{Int16: 1, Valid: true}}
	cases := []struct {
		name   string
		edit   func(d *store.ShopDiscountCode)
		total  uint64
		uses   int64
		amount uint64
		why    string
	}{
		{"percent", nil, 300000, 0, 30000, ""},
		{"capped", nil, 973000, 0, 50000, ""},
		{"min total", nil, 99999, 0, 0, "discount_min_total"},
		{"empty cart", func(d *store.ShopDiscountCode) { d.MinItemsTotal = 0 }, 0, 0, 0, "discount_min_total"},
		{"inactive", func(d *store.ShopDiscountCode) { d.IsActive = false }, 300000, 0, 0, "discount_invalid"},
		{"not started", func(d *store.ShopDiscountCode) { d.StartsAt = sql.NullTime{Time: now.Add(time.Hour), Valid: true} }, 300000, 0, 0, "discount_invalid"},
		{"ended", func(d *store.ShopDiscountCode) { d.EndsAt = sql.NullTime{Time: now.Add(-time.Hour), Valid: true} }, 300000, 0, 0, "discount_invalid"},
		{"exhausted", func(d *store.ShopDiscountCode) { d.UsedCount = 5 }, 300000, 0, 0, "discount_exhausted"},
		{"used by her", nil, 300000, 1, 0, "discount_used"},
		{"no per-user limit", func(d *store.ShopDiscountCode) { d.PerUserLimit = sql.NullInt16{} }, 300000, 3, 30000, ""},
		{"fixed never above total", func(d *store.ShopDiscountCode) {
			d.Kind, d.Value, d.MinItemsTotal, d.MaxDiscount = DiscountFixed, 500000, 0, sql.NullInt64{}
		}, 120000, 0, 120000, ""},
		{"unknown kind", func(d *store.ShopDiscountCode) { d.Kind = "bogo" }, 300000, 0, 0, "discount_invalid"},
	}
	for _, tc := range cases {
		d := base
		if tc.edit != nil {
			tc.edit(&d)
		}
		amount, why := evaluateDiscount(&d, tc.total, now, tc.uses)
		assert.Equal(t, tc.amount, amount, tc.name)
		assert.Equal(t, tc.why, why, tc.name)
	}
}

func TestBuildCart_GroupsFeesAndStatuses(t *testing.T) {
	a := &store.ShopSeller{ID: 1, IsActive: true, ShippingFee: 35000, FreeShippingOver: sql.NullInt64{Int64: 1000000, Valid: true},
		CodMaxTotal: sql.NullInt64{Int64: 900000, Valid: true}}
	b := &store.ShopSeller{ID: 2, IsActive: true, ShippingFee: 20000}
	off := &store.ShopSeller{ID: 3, IsActive: false}
	row := func(variant, seller uint64, price uint64, qty uint16, stock uint32) store.ListCartLinesRow {
		return store.ListCartLinesRow{VariantID: variant, Quantity: qty, Price: price, Stock: stock, VariantActive: true,
			ProductActive: true, CategoryActive: true, SellerID: sql.NullInt64{Int64: int64(seller), Valid: seller > 0}} //nolint:gosec // test
	}
	c := buildCart(9, []store.ListCartLinesRow{row(1, 1, 485000, 1, 5), row(2, 2, 98000, 2, 1), row(3, 1, 390000, 1, 3),
		row(4, 3, 1000, 1, 1), row(5, 0, 1000, 1, 1)}, map[uint64]*store.ShopSeller{1: a, 2: b, 3: off})
	require.Len(t, c.Groups, 2)
	assert.Equal(t, uint64(1), c.Groups[0].Seller.ID, "groups keep the order of each seller's first line")
	assert.Equal(t, uint64(875000), c.Groups[0].ItemsTotal)
	assert.Equal(t, uint64(35000), c.Groups[0].ShippingFee)
	assert.Equal(t, uint64(125000), freeShippingRemaining(c.Groups[0]))
	assert.Equal(t, LineOutOfStock, c.Groups[1].Lines[0].Status)
	assert.Equal(t, uint64(0), c.Groups[1].ItemsTotal)
	assert.Equal(t, uint64(0), c.Groups[1].ShippingFee, "nothing to ship costs nothing")
	assert.Len(t, c.Unavailable, 2, "inactive seller and no seller")
	assert.False(t, c.Purchasable())
	assert.False(t, codAllowed(c.Groups[0]), "875000 + 35000 > 900000")

	c.settle(now)
	assert.Equal(t, uint64(196000), c.Groups[1].ItemsTotal)
	assert.Equal(t, uint64(875000+196000), c.ItemsTotal)
	assert.Equal(t, uint64(35000+20000), c.ShippingTotal())
	assert.Equal(t, c.ItemsTotal+c.ShippingTotal(), c.Payable())
}

func TestShippingFee_FreeOver(t *testing.T) {
	s := &store.ShopSeller{ShippingFee: 35000, FreeShippingOver: sql.NullInt64{Int64: 1000000, Valid: true}}
	assert.Equal(t, uint64(35000), shippingFee(s, 999999))
	assert.Equal(t, uint64(0), shippingFee(s, 1000000))
	g := &Group{Seller: s, ItemsTotal: 1000000}
	assert.Nil(t, freeShippingRemaining(g))
}

func TestVariantLabels(t *testing.T) {
	parts := variantParts(`{"fa":"۳ تا ۶ ماه","en":"3-6 m"}`, `{"fa":"شیری"}`, "null")
	assert.Equal(t, "3-6 m · شیری", variantLabel(catalog.Locale{Code: "en", Default: "fa"}, parts), "falls back to the default language")
	snap, ok := variantLabelSnapshot("fa", parts)
	require.True(t, ok)
	var m map[string]string
	require.NoError(t, json.Unmarshal([]byte(snap), &m))
	assert.Equal(t, map[string]string{"fa": "۳ تا ۶ ماه · شیری", "en": "3-6 m · شیری"}, m)
	_, ok = variantLabelSnapshot("fa", nil)
	assert.False(t, ok)
}

func TestReturnUntil(t *testing.T) {
	d := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, d.AddDate(0, 0, 7), ReturnUntil(d, nil))
	assert.Equal(t, d.AddDate(0, 0, 14), ReturnUntil(d, &store.ShopSeller{ReturnDays: 14}))
}

func TestDetailCanCancel(t *testing.T) {
	d := &Detail{Order: store.ShopOrder{Status: StatusPlaced}, Shipments: []store.ShopOrderShipment{{Status: ShipmentAwaiting}}}
	assert.True(t, d.CanCancel())
	d.Shipments = append(d.Shipments, store.ShopOrderShipment{Status: "confirmed"})
	assert.False(t, d.CanCancel())
	d = &Detail{Order: store.ShopOrder{Status: StatusCancelled}}
	assert.False(t, d.CanCancel())
}

func TestSplitQuoted(t *testing.T) {
	msg := "Duplicate entry '12345678' for key 'shop_orders_code_unique'"
	assert.True(t, containsKey(msg, "shop_orders_code_unique"))
	assert.False(t, containsKey(msg, "shop_orders_user_id_idempotency_key_unique"))
}

// The SMS copy is discreet: the code and «pay on delivery», nothing about what was bought.
func TestOrderSMSIsDiscreet(t *testing.T) {
	for _, loc := range []string{"fa", "en"} {
		s := T("sms.order_placed", loc, "code", "52084100")
		assert.Contains(t, s, "52084100", loc)
		assert.NotContains(t, s, ":code", loc)
	}
	f := NewFakeNotifier(nil)
	require.NoError(t, f.OrderPlaced(t.Context(), Notice{OrderID: 1, Code: "x"}))
	assert.Len(t, f.Sent(), 1)
}

func flatKeys(prefix string, v any, out *[]string) {
	m, ok := v.(map[string]any)
	if !ok {
		*out = append(*out, prefix)
		return
	}
	for k, x := range m {
		flatKeys(strings.TrimPrefix(prefix+"."+k, "."), x, out)
	}
}

// Every language file carries the same keys, and every error code the service can raise has a line.
func TestLangFilesMatch(t *testing.T) {
	keys := map[string][]string{}
	require.NoError(t, fs.WalkDir(langFS, "lang", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := langFS.ReadFile(p)
		require.NoError(t, err)
		var v map[string]any
		require.NoError(t, json.Unmarshal(raw, &v), p)
		var list []string
		flatKeys("", v, &list)
		sort.Strings(list)
		keys[p] = list
		return nil
	}))
	require.Len(t, keys, 2)
	assert.Equal(t, keys["lang/en/shop_order.json"], keys["lang/fa/shop_order.json"])
	for _, code := range []string{"variant_unavailable", "out_of_stock", "quantity_limit", "cart_full", "cart_empty",
		"cart_unavailable", "discount_invalid", "discount_min_total", "discount_used", "discount_exhausted", "address_invalid",
		"address_limit", "slot_invalid", "slot_full", "cod_limit_exceeded", "not_cancellable", "not_returnable",
		"return_window_closed", "return_quantity", "busy", "not_found", "validation_failed"} {
		assert.NotEqual(t, "shop_order.errors."+code, T("errors."+code, "fa"), code)
	}
}

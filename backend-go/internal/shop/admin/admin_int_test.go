package admin_test

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/shop"
	shopadmin "github.com/ritme/backend-go/internal/shop/admin"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	marketingID uint64 = 3 // content manage, no support
	supportID   uint64 = 4 // support manage, no content
)

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	shop.NewAdmin(e.DB, nil, "http://localhost", admintest.Quiet).Routes(e.Route(), e.Kit)
	shopadmin.New(e.DB, nil, "http://localhost", admintest.Quiet).Routes(e.Route(), e.Kit)
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(3, 'Mkt', 'mkt@ritme.test', 'x', 'marketing', 1, NOW(), NOW()),
		(4, 'Sup', 'sup@ritme.test', 'x', 'support', 1, NOW(), NOW())`)
	return e
}

func id(r admintest.Resp, key string) uint64 { return uint64(r.Obj(key)["id"].(float64)) }

func s(n uint64) string { return strconv.FormatUint(n, 10) }

func TestShopAdmin_CapabilitiesAndCSRF(t *testing.T) {
	e := newEnv(t)
	assert.Equal(t, httpadmin.CapSupport, httpadmin.RouteCapability(httpadmin.Prefix+"/shop/shipments"))
	assert.Equal(t, httpadmin.CapSupport, httpadmin.RouteCapability(httpadmin.Prefix+"/shop/orders/:id"))
	assert.Equal(t, httpadmin.CapSupport, httpadmin.RouteCapability(httpadmin.Prefix+"/shop/returns/:id/approve"))
	assert.Equal(t, httpadmin.CapContent, httpadmin.RouteCapability(httpadmin.Prefix+"/shop/sellers"))
	assert.Equal(t, httpadmin.CapContent, httpadmin.RouteCapability(httpadmin.Prefix+"/shop/products"))

	assert.Equal(t, 401, e.Anonymous().Get("/shop/sellers").Status)
	assert.Equal(t, 200, e.As(marketingID).Get("/shop/sellers").Status)
	assert.Equal(t, 403, e.As(marketingID).Get("/shop/shipments").Status, "order PII needs support")
	assert.Equal(t, 403, e.As(marketingID).Get("/shop/returns").Status)
	assert.Equal(t, 200, e.As(supportID).Get("/shop/shipments").Status)
	assert.Equal(t, 403, e.As(supportID).Get("/shop/sellers").Status)
	assert.Equal(t, 200, e.As(admintest.EditorID).Get("/shop/shipments").Status)

	noToken := e.As(admintest.EditorID)
	noToken.CSRF = ""
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/shop/sellers", map[string]any{}).Status)
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/shop/shipments/1/move", map[string]any{"status": "confirmed"}).Status)
}

func TestShopAdmin_CatalogStockSlotsCodesTemplate(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	// Sellers.
	seller := map[string]any{"slug": "panbe", "name": map[string]any{"fa": "پنبه", "en": "Panbe"}, "ship_min_days": 3,
		"ship_max_days": 1, "shipping_fee": 35000, "return_days": 7, "is_active": true}
	r := c.JSON(fiber.MethodPost, "/shop/sellers", seller)
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "ship_max_days")
	seller["ship_max_days"] = 4
	r = c.JSON(fiber.MethodPost, "/shop/sellers", seller)
	require.Equal(t, 201, r.Status, r.Body)
	sellerID := id(r, "seller")
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/shop/sellers", seller).Status, "slug unique")

	// Sub-category under layette; a root with children cannot become a child.
	layette := uint64(e.Int("SELECT id FROM shop_categories WHERE slug = 'layette'"))
	beauty := uint64(e.Int("SELECT id FROM shop_categories WHERE slug = 'beauty'"))
	r = c.JSON(fiber.MethodPost, "/shop/categories", map[string]any{"slug": "clothes", "title": map[string]any{"fa": "لباس", "en": "Clothes"}, "is_active": true})
	require.Equal(t, 201, r.Status, r.Body)
	clothes := id(r, "category")
	r = c.JSON(fiber.MethodPut, "/shop/categories/"+s(clothes)+"/parent", map[string]any{"parent_id": layette})
	require.Equal(t, 200, r.Status, r.Body)
	assert.InDelta(t, float64(layette), r.Obj("category")["parent_id"], 0)
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, "/shop/categories/"+s(beauty)+"/parent", map[string]any{"parent_id": clothes}).Status)

	// Product (bloom's CRUD) + commerce fields + variants.
	r = c.JSON(fiber.MethodPost, "/shop/products", map[string]any{"category_id": clothes, "title": map[string]any{"fa": "بادی", "en": "Bodysuit"}, "is_active": true})
	require.Equal(t, 201, r.Status, r.Body)
	pid := id(r, "product")
	commerce := map[string]any{"seller_id": sellerID, "subtitle": map[string]any{"fa": "نخی"},
		"attributes": map[string]any{"material": map[string]any{"fa": "پنبه", "en": "Cotton"}, "wash": "30°"},
		"skin_types": []string{"dry", "dry", "oily"}, "badges": []any{"organic"}, "expires_on": "2027-01-31", "irc_code": "IRC-1"}
	r = c.JSON(fiber.MethodPut, "/shop/products/"+s(pid)+"/commerce", commerce)
	require.Equal(t, 200, r.Status, r.Body)
	p := r.Obj("product")
	assert.Equal(t, []any{"dry", "oily"}, p["skin_types"])
	assert.Equal(t, "2027-01-31", p["expires_on"])
	assert.InDelta(t, float64(sellerID), p["seller_id"], 0)
	bad := map[string]any{"skin_types": []string{"greasy"}, "attributes": map[string]any{"Bad Key": "x"}, "images": []string{"shop/other.jpg"}, "seller_id": 999}
	r = c.JSON(fiber.MethodPut, "/shop/products/"+s(pid)+"/commerce", bad)
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"skin_types.0", "attributes", "images", "seller_id"} {
		assert.Contains(t, r.Errors(), f)
	}

	r = c.JSON(fiber.MethodPost, "/shop/products/"+s(pid)+"/variants", map[string]any{"sku": "B-36", "size": map[string]any{"fa": "۳-۶ ماه", "en": "3-6 m"}, "price": 485000, "stock": 2, "is_active": true})
	require.Equal(t, 201, r.Status, r.Body)
	vid := id(r, "variant")
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/shop/products/"+s(pid)+"/variants", map[string]any{"sku": "B-36", "price": 1, "stock": 0}).Status, "sku unique")
	r = c.Get("/shop/stock?low=1")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 1)
	r = c.JSON(fiber.MethodPost, "/shop/variants/"+s(vid)+"/stock", map[string]any{"stock": 40})
	require.Equal(t, 200, r.Status, r.Body)
	assert.InDelta(t, 40, r.Obj("variant")["stock"], 0)
	assert.Empty(t, c.Get("/shop/stock?low=1").Items())
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'shop_variant.stock'"))

	// Slots: template over a week, only Saturdays and Mondays, two windows.
	gen := map[string]any{"seller_id": sellerID, "from": "2026-10-10", "to": "2026-10-16", "weekdays": []int{6, 1},
		"windows": []any{map[string]any{"starts_at": "09:00", "ends_at": "13:00", "capacity": 5}, map[string]any{"starts_at": "17:00", "ends_at": "21:00"}}}
	r = c.JSON(fiber.MethodPost, "/shop/slots/generate", gen)
	require.Equal(t, 201, r.Status, r.Body)
	assert.InDelta(t, 4, r.Data()["created"], 0)
	r = c.JSON(fiber.MethodPost, "/shop/slots/generate", gen)
	assert.InDelta(t, 0, r.Data()["created"], 0, "existing slots kept")
	r = c.Get("/shop/slots?seller_id=" + s(sellerID) + "&from=2026-10-10&to=2026-10-16")
	require.Len(t, r.Items(), 4)
	slot := uint64(r.Items()[0].(map[string]any)["id"].(float64))
	e.Exec("UPDATE shop_delivery_slots SET booked = 3 WHERE id = ?", slot)
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, "/shop/slots/"+s(slot), map[string]any{"capacity": 2, "is_active": true}).Status)
	assert.Equal(t, shopadmin.CodeSlotBooked, c.JSON(fiber.MethodDelete, "/shop/slots/"+s(slot), nil).Code())
	bad2 := map[string]any{"seller_id": sellerID, "from": "2026-10-16", "to": "2026-10-10", "windows": []any{map[string]any{"starts_at": "13:00", "ends_at": "09:00"}}}
	r = c.JSON(fiber.MethodPost, "/shop/slots/generate", bad2)
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "to")
	assert.Contains(t, r.Errors(), "windows")

	// Seller in use.
	assert.Equal(t, shopadmin.CodeInUse, c.JSON(fiber.MethodDelete, "/shop/sellers/"+s(sellerID), nil).Code())

	// Discount codes.
	r = c.JSON(fiber.MethodPost, "/shop/discount-codes", map[string]any{"code": "welcome10", "kind": "percent", "value": 150})
	require.Equal(t, 422, r.Status)
	r = c.JSON(fiber.MethodPost, "/shop/discount-codes", map[string]any{"code": "welcome10", "kind": "percent", "value": 10, "max_uses": 100, "is_active": true})
	require.Equal(t, 201, r.Status, r.Body)
	assert.Equal(t, "WELCOME10", r.Obj("discount_code")["code"])
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/shop/discount-codes", map[string]any{"code": "WELCOME10", "kind": "fixed", "value": 10}).Status)

	// Layette template.
	r = c.JSON(fiber.MethodPost, "/shop/layette-items", map[string]any{"group_key": "toys", "title": map[string]any{"fa": "بادی"}})
	require.Equal(t, 422, r.Status)
	r = c.JSON(fiber.MethodPost, "/shop/layette-items", map[string]any{"group_key": "clothing", "title": map[string]any{"fa": "بادی", "en": "Bodysuit"},
		"suggested_qty": 6, "is_essential": true, "category_id": clothes, "is_active": true})
	require.Equal(t, 201, r.Status, r.Body)
	item := id(r, "item")
	r = c.JSON(fiber.MethodPut, "/shop/layette-items/"+s(item), map[string]any{"group_key": "sleep", "title": map[string]any{"fa": "سرهمی"}, "is_active": false})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "sleep", r.Obj("item")["group_key"])
	assert.Equal(t, 200, c.JSON(fiber.MethodDelete, "/shop/layette-items/"+s(item), nil).Status)
}

type orderFixture struct {
	order, shipment, item, variant, product, user uint64
}

func seedOrder(e *admintest.Env) orderFixture {
	var f orderFixture
	ins := func(q string, args ...any) uint64 {
		n, err := e.Exec(q, args...).LastInsertId()
		require.NoError(e.T, err)
		return uint64(n)
	}
	f.user = ins(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120000001', NOW(), NOW())`)
	seller := ins(`INSERT INTO shop_sellers (slug, name, return_days) VALUES ('s1', '{"fa":"فروشنده","en":"Seller"}', 7)`)
	cat := uint64(e.Int("SELECT id FROM shop_categories WHERE slug = 'layette'"))
	f.product = ins(`INSERT INTO shop_products (category_id, seller_id, title, is_active) VALUES (?, ?, '{"fa":"بادی","en":"Bodysuit"}', 1)`, cat, seller)
	f.variant = ins(`INSERT INTO shop_product_variants (product_id, price, stock) VALUES (?, 100000, 3)`, f.product)
	f.order = ins(`INSERT INTO shop_orders (code, user_id, recipient_name, recipient_phone, city, address_text, items_total, payable_total, placed_at, created_at)
		VALUES ('12345678', ?, 'Sara A', '09120000001', 'Tehran', 'Street 1', 200000, 235000, NOW(), NOW())`, f.user)
	f.shipment = ins(`INSERT INTO shop_order_shipments (order_id, seller_id, items_total, shipping_fee, created_at) VALUES (?, ?, 200000, 35000, NOW())`, f.order, seller)
	f.item = ins(`INSERT INTO shop_order_items (order_id, shipment_id, product_id, variant_id, title, unit_price, quantity, line_total)
		VALUES (?, ?, ?, ?, '{"fa":"بادی","en":"Bodysuit"}', 100000, 2, 200000)`, f.order, f.shipment, f.product, f.variant)
	return f
}

func TestShopAdmin_OrderLifecycleReturnsAndReviews(t *testing.T) {
	e := newEnv(t)
	c := e.As(supportID)
	f := seedOrder(e)

	r := c.Get("/shop/shipments?status=awaiting_seller")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 1)
	row := r.Items()[0].(map[string]any)
	assert.Equal(t, "12345678", row["order_code"])
	assert.Equal(t, []any{"confirmed"}, row["allowed_statuses"])

	move := func(to string, extra ...string) admintest.Resp {
		body := map[string]any{"status": to}
		if len(extra) > 0 {
			body["tracking_code"] = extra[0]
		}
		return c.JSON(fiber.MethodPost, "/shop/shipments/"+s(f.shipment)+"/move", body)
	}
	r = move("shipped")
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, shopadmin.CodeInvalidTransition, r.Code())
	require.Equal(t, 200, move("confirmed").Status)
	r = move("shipped", "TRK-9")
	require.Equal(t, 200, r.Status, r.Body)
	o := r.Obj("order")
	assert.Equal(t, false, o["can_cancel"], "shipped: no cancel")
	assert.Equal(t, "09120000001", o["recipient_phone"])
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/shop/orders/"+s(f.order)+"/cancel", nil).Status)

	r = move("delivered")
	require.Equal(t, 200, r.Status, r.Body)
	o = r.Obj("order")
	assert.Equal(t, "delivered", o["status"], "MarkShipmentDelivered closes the order")
	ship := o["shipments"].([]any)[0].(map[string]any)
	assert.Equal(t, "TRK-9", ship["tracking_code"])
	assert.NotNil(t, ship["delivered_at"])
	assert.Len(t, o["events"], 4) // confirmed, shipped, delivered (shipment), delivered (order)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'shop_shipment.move' AND detail LIKE '%delivered%'"))
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE detail LIKE '%0912%' OR detail LIKE '%Sara%'"), "no PII in the audit log")

	// Returns: approve with restock, then it is decided.
	ret, err := e.DB.Exec(`INSERT INTO shop_returns (order_id, order_item_id, user_id, quantity, reason, status, created_at) VALUES (?, ?, ?, 1, 'damaged', 'requested', NOW())`, f.order, f.item, f.user)
	require.NoError(t, err)
	retID, _ := ret.LastInsertId()
	r = c.Get("/shop/returns?status=requested")
	require.Len(t, r.Items(), 1)
	r = c.JSON(fiber.MethodPost, "/shop/returns/"+strconv.FormatInt(retID, 10)+"/approve", map[string]any{"restock": true})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 4, e.Int("SELECT stock FROM shop_product_variants WHERE id = ?", f.variant))
	r = c.JSON(fiber.MethodPost, "/shop/returns/"+strconv.FormatInt(retID, 10)+"/reject", nil)
	assert.Equal(t, shopadmin.CodeNotPending, r.Code())

	// Admin cancel of another order while confirmed gives stock back.
	g := seedOrder2(e, f)
	require.Equal(t, 200, c.JSON(fiber.MethodPost, "/shop/shipments/"+s(g)+"/move", map[string]any{"status": "confirmed"}).Status)
	orderID := uint64(e.Int("SELECT order_id FROM shop_order_shipments WHERE id = ?", g))
	r = c.JSON(fiber.MethodPost, "/shop/orders/"+s(orderID)+"/cancel", nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "cancelled", r.Obj("order")["status"])
	assert.Equal(t, 5, e.Int("SELECT stock FROM shop_product_variants WHERE id = ?", f.variant))

	// Reviews (content capability): publishing refreshes the rating cache.
	ed := e.As(admintest.EditorID)
	rev, err := e.DB.Exec(`INSERT INTO shop_product_reviews (product_id, user_id, author_name, rating, status, created_at) VALUES (?, ?, 'S.', 4, 'pending', NOW())`, f.product, f.user)
	require.NoError(t, err)
	revID, _ := rev.LastInsertId()
	assert.Len(t, ed.Get("/shop/reviews").Items(), 1)
	assert.Equal(t, 403, c.Get("/shop/reviews").Status)
	r = ed.JSON(fiber.MethodPost, "/shop/reviews/"+strconv.FormatInt(revID, 10)+"/moderate", map[string]any{"status": "published"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 1, e.Int("SELECT rating_count FROM shop_products WHERE id = ?", f.product))
	assert.Equal(t, "4.00", e.String("SELECT rating_avg FROM shop_products WHERE id = ?", f.product))
	require.Equal(t, 200, ed.JSON(fiber.MethodDelete, "/shop/reviews/"+strconv.FormatInt(revID, 10), nil).Status)
	assert.Equal(t, 0, e.Int("SELECT rating_count FROM shop_products WHERE id = ?", f.product))
}

// seedOrder2 adds a second placed order of one unit on the same variant (stock already taken) and returns its shipment.
func seedOrder2(e *admintest.Env, f orderFixture) uint64 {
	res := e.Exec(`INSERT INTO shop_orders (code, user_id, recipient_name, recipient_phone, city, address_text, items_total, payable_total, placed_at, created_at)
		VALUES ('87654321', ?, 'Sara A', '09120000001', 'Tehran', 'Street 1', 100000, 100000, NOW(), NOW())`, f.user)
	oid, _ := res.LastInsertId()
	seller := e.Int("SELECT seller_id FROM shop_products WHERE id = ?", f.product)
	res = e.Exec(`INSERT INTO shop_order_shipments (order_id, seller_id, items_total, created_at) VALUES (?, ?, 100000, NOW())`, oid, seller)
	sid, _ := res.LastInsertId()
	e.Exec(`INSERT INTO shop_order_items (order_id, shipment_id, product_id, variant_id, title, unit_price, quantity, line_total)
		VALUES (?, ?, ?, ?, '{"fa":"بادی"}', 100000, 1, 100000)`, oid, sid, f.product, f.variant)
	return uint64(sid)
}

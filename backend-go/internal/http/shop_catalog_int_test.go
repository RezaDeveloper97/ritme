package http

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CB-SHOP-02: the internal catalog read API (/api/v1/shop/catalog/*) next to bloom's link-out routes.

func (e *companionEnv) exec(q string, args ...any) uint64 {
	e.t.Helper()
	res, err := e.db.Exec(q, args...)
	require.NoError(e.t, err)
	id, _ := res.LastInsertId()
	return uint64(id) //nolint:gosec // test ids
}

type catalogFixture struct {
	clothes, beautySub, bodysuit, sleepsuit, socks, sunscreen, linkOut, inactiveSeller uint64
}

func seedCatalog(e *companionEnv) catalogFixture {
	var f catalogFixture
	seller := e.exec(`INSERT INTO shop_sellers (slug, name, is_verified, ship_min_days, ship_max_days, shipping_fee, free_shipping_over, return_days, discreet_packaging)
		VALUES ('panbe', '{"fa":"پوشاک پنبه ریز","en":"Panbe Kids"}', 1, 2, 4, 35000, 1000000, 7, 0)`)
	hidden := e.exec(`INSERT INTO shop_sellers (slug, name, is_active) VALUES ('gone', '{"fa":"x"}', 0)`)
	f.clothes = e.exec(`INSERT INTO shop_categories (slug, title, sort_order, is_active, parent_id)
		VALUES ('baby-clothes', '{"fa":"لباس نوزاد","en":"Baby clothes"}', 1, 1, (SELECT id FROM (SELECT id FROM shop_categories WHERE slug='layette') x))`)
	e.exec(`INSERT INTO shop_categories (slug, title, sort_order, is_active, parent_id)
		VALUES ('empty-sub', '{"fa":"خالی","en":"Empty"}', 2, 1, (SELECT id FROM (SELECT id FROM shop_categories WHERE slug='layette') x))`)
	f.beautySub = e.exec(`INSERT INTO shop_categories (slug, title, sort_order, is_active, parent_id)
		VALUES ('skin-care', '{"fa":"مراقبت پوست","en":"Skin care"}', 1, 1, (SELECT id FROM (SELECT id FROM shop_categories WHERE slug='beauty') x))`)
	product := func(cat, sellerID uint64, title string, sold int, attrs, skins any) uint64 {
		return e.exec(`INSERT INTO shop_products (category_id, seller_id, title, subtitle, attributes, skin_types, badges, images, irc_code, expires_on, rating_avg, rating_count, purchase_count, is_active, created_at, updated_at)
			VALUES (?, ?, ?, '{"fa":"۳ عدد","en":"Pack of 3"}', ?, ?, '[{"fa":"پنبه ۱۰۰٪","en":"100% cotton"}]', '["shop/a.jpg","shop/b.jpg"]', 'IRC-1', '2027-05-01', 4.70, 2, ?, 1, NOW(), NOW())`,
			cat, sellerID, title, attrs, skins, sold)
	}
	variant := func(p uint64, size, color, volume any, price, stock int) uint64 {
		return e.exec(`INSERT INTO shop_product_variants (product_id, size, color, volume, price, compare_at_price, stock, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
			p, size, color, volume, price, price+50000, stock)
	}
	cotton := `{"material":{"fa":"پنبه ۱۰۰٪","en":"100% cotton"},"wash":{"fa":"۳۰ درجه","en":"30°C machine wash"}}`
	f.bodysuit = product(f.clothes, seller, `{"fa":"بادی آستین بلند","en":"Long-sleeve bodysuit"}`, 1800, cotton, nil)
	v1 := variant(f.bodysuit, `{"fa":"۰ تا ۳ ماه","en":"0-3 m"}`, `{"fa":"شیری","en":"Cream"}`, nil, 485000, 4)
	variant(f.bodysuit, `{"fa":"۳ تا ۶ ماه","en":"3-6 m"}`, `{"fa":"شیری","en":"Cream"}`, nil, 485000, 0)
	f.sleepsuit = product(f.clothes, seller, `{"fa":"سرهمی خواب","en":"Sleepsuit"}`, 98, nil, nil)
	v2 := variant(f.sleepsuit, `{"fa":"۳ تا ۶ ماه","en":"3-6 m"}`, `{"fa":"یاسی","en":"Lilac"}`, nil, 390000, 2)
	f.socks = product(f.clothes, seller, `{"fa":"جوراب","en":"Socks"}`, 130, nil, nil)
	variant(f.socks, `{"fa":"۰ تا ۳ ماه","en":"0-3 m"}`, nil, nil, 140000, 9)
	f.sunscreen = product(f.beautySub, seller, `{"fa":"ضدآفتاب","en":"Sunscreen"}`, 403, nil, `["oily","combination"]`)
	variant(f.sunscreen, nil, nil, `{"fa":"۵۰ میلی لیتر","en":"50 ml"}`, 550000, 3)
	f.inactiveSeller = product(f.clothes, hidden, `{"fa":"پنهان","en":"HiddenSeller"}`, 5000, nil, nil)
	variant(f.inactiveSeller, nil, nil, nil, 1000, 1)
	f.linkOut = shopProduct(e.t, e, "layette", "LinkOut", "https://partner.example/p", 1)

	// Reviews: two published (one verified purchase), one pending.
	buyer := e.user("Buyer", "09120008090")
	_ = buyer
	uid := e.ids["Buyer"]
	order := e.exec(`INSERT INTO shop_orders (code, user_id, recipient_name, recipient_phone, city, address_text, items_total, payable_total)
		VALUES ('100001', ?, 'B', '0912', 'Tehran', 'x', 1, 1)`, uid)
	ship := e.exec(`INSERT INTO shop_order_shipments (order_id, seller_id, items_total) VALUES (?, ?, 1)`, order, seller)
	line := e.exec(`INSERT INTO shop_order_items (order_id, shipment_id, product_id, variant_id, title, unit_price, quantity, line_total) VALUES (?, ?, ?, ?, '{"fa":"x"}', 1, 1, 1)`, order, ship, f.bodysuit, v1)
	e.exec(`INSERT INTO shop_order_items (order_id, shipment_id, product_id, variant_id, title, unit_price, quantity, line_total) VALUES (?, ?, ?, ?, '{"fa":"x"}', 1, 1, 1)`, order, ship, f.sleepsuit, v2)
	e.exec(`INSERT INTO shop_product_reviews (product_id, user_id, order_item_id, author_name, variant_label, rating, body, status, created_at)
		VALUES (?, ?, ?, 'Parnia', '{"fa":"۳ تا ۶ ماه","en":"3-6 m"}', 5, 'Soft fabric', 'published', '2026-10-01 10:00:00')`, f.bodysuit, uid, line)
	e.exec(`INSERT INTO shop_product_reviews (product_id, user_id, author_name, rating, body, status, created_at)
		VALUES (?, ?, 'Sam', 4, 'Runs large', 'published', '2026-10-02 10:00:00')`, f.bodysuit, uid)
	e.exec(`INSERT INTO shop_product_reviews (product_id, user_id, author_name, rating, body, status, created_at)
		VALUES (?, ?, 'Spam', 1, 'PENDING', 'pending', '2026-10-03 10:00:00')`, f.bodysuit, uid)
	return f
}

func names(list []any, key string) []any {
	out := []any{}
	for _, x := range list {
		out = append(out, x.(map[string]any)[key])
	}
	return out
}

func TestShopCatalog_HomeListDetailReviews(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008011")
	for _, p := range []string{"/api/v1/shop/catalog/home", "/api/v1/shop/catalog/products", "/api/v1/shop/catalog/products/1",
		"/api/v1/shop/catalog/products/1/reviews"} {
		assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, p, "", nil).Status, p)
	}
	f := seedCatalog(e)

	// bloom's link-out routes keep their shape and only list root categories.
	r := e.do(fiber.MethodGet, "/api/v1/shop/categories", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, []any{"layette", "beauty"}, names(r.data()["categories"].([]any), "slug"))
	r = e.do(fiber.MethodGet, "/api/v1/shop/products/"+itoa(f.linkOut), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "https://partner.example/p?utm_source=ritme", r.data()["product"].(map[string]any)["link_url"])

	// Home: layette by default; empty sub-categories have no section.
	r = e.do(fiber.MethodGet, "/api/v1/shop/catalog/home", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	d := r.data()
	assert.Equal(t, "layette", d["root"].(map[string]any)["slug"])
	assert.Equal(t, []any{"layette", "beauty"}, names(d["roots"].([]any), "slug"))
	subs := d["sub_categories"].([]any)
	assert.Equal(t, []any{"baby-clothes", "empty-sub"}, names(subs, "slug"))
	assert.EqualValues(t, 3, subs[0].(map[string]any)["product_count"], "inactive seller and link-out rows are not catalog products")
	assert.Equal(t, "layette", subs[0].(map[string]any)["parent"])
	sections := d["sections"].([]any)
	require.Len(t, sections, 1)
	sec := sections[0].(map[string]any)
	assert.Equal(t, "Baby clothes", sec["category"].(map[string]any)["title"])
	cards := sec["products"].([]any)
	assert.Equal(t, []any{"Long-sleeve bodysuit", "Socks", "Sleepsuit"}, names(cards, "title"), "most bought first")
	card := cards[0].(map[string]any)
	assert.EqualValues(t, 485000, card["price"])
	assert.EqualValues(t, 535000, card["compare_at_price"])
	assert.InDelta(t, 4.7, card["rating_avg"], 0.001)
	assert.Equal(t, "http://localhost/storage/shop/a.jpg", card["image_url"])
	assert.Equal(t, []any{"100% cotton"}, card["badges"])
	assert.Equal(t, "Panbe Kids", card["seller"].(map[string]any)["name"])
	assert.NotContains(t, r.Raw, "LinkOut")
	assert.NotContains(t, r.Raw, "HiddenSeller")

	r = e.do(fiber.MethodGet, "/api/v1/shop/catalog/home?root=beauty", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "Sunscreen", r.data()["sections"].([]any)[0].(map[string]any)["products"].([]any)[0].(map[string]any)["title"])
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/catalog/home?root=baby-clothes", sara, nil).Status, "a sub-category is not a home")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/catalog/home?root=nope", sara, nil).Status)

	// List: root subtree, filters, sort, paging.
	r = e.do(fiber.MethodGet, "/api/v1/shop/catalog/products?category=layette", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	d = r.data()
	assert.EqualValues(t, 3, d["meta"].(map[string]any)["total"])
	filters := d["filters"].(map[string]any)
	assert.Equal(t, []any{"۰ تا ۳ ماه", "۳ تا ۶ ماه"}, names(filters["sizes"].([]any), "value"))
	assert.Equal(t, []any{"0-3 m", "3-6 m"}, names(filters["sizes"].([]any), "label"))
	assert.Equal(t, []any{"100% cotton"}, names(filters["materials"].([]any), "label"))
	assert.EqualValues(t, 140000, filters["price"].(map[string]any)["min"])
	assert.EqualValues(t, 485000, filters["price"].(map[string]any)["max"])
	assert.Equal(t, "popular", d["sort"])

	list := func(q string) []any {
		t.Helper()
		r := e.do(fiber.MethodGet, "/api/v1/shop/catalog/products?"+q, sara, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		return names(r.data()["products"].([]any), "title")
	}
	assert.Equal(t, []any{"Socks", "Sleepsuit", "Long-sleeve bodysuit"}, list("category=baby-clothes&sort=price_asc"))
	assert.Equal(t, []any{"Long-sleeve bodysuit", "Socks"}, list("category=baby-clothes&size=%DB%B0%20%D8%AA%D8%A7%20%DB%B3%20%D9%85%D8%A7%D9%87"))
	assert.Equal(t, []any{"Sleepsuit"}, list("category=layette&color=%DB%8C%D8%A7%D8%B3%DB%8C"))
	assert.Equal(t, []any{"Long-sleeve bodysuit"}, list("category=layette&material=%D9%BE%D9%86%D8%A8%D9%87%20%DB%B1%DB%B0%DB%B0%D9%AA"))
	assert.Equal(t, []any{"Sunscreen"}, list("skin_type=oily"))
	assert.Equal(t, []any{"Sleepsuit", "Socks"}, list("category=layette&price_max=400000&sort=price_desc"))
	assert.Equal(t, []any{"Socks"}, list("q=sock"))
	assert.Equal(t, []any{"Socks"}, list("category=layette&per_page=1&page=2"))
	assert.Equal(t, []any{}, list("category=empty-sub"))
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/catalog/products?category=nope", sara, nil).Status)

	// Detail.
	r = e.do(fiber.MethodGet, "/api/v1/shop/catalog/products/"+itoa(f.bodysuit), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	p := r.data()["product"].(map[string]any)
	assert.Equal(t, "Pack of 3", p["subtitle"])
	assert.Equal(t, []any{"http://localhost/storage/shop/a.jpg", "http://localhost/storage/shop/b.jpg"}, p["images"])
	assert.Equal(t, map[string]any{"material": "100% cotton", "wash": "30°C machine wash"}, p["attributes"])
	assert.Equal(t, "IRC-1", p["irc_code"])
	assert.Equal(t, "2027-05-01", p["expires_on"])
	vs := p["variants"].([]any)
	require.Len(t, vs, 2)
	assert.Equal(t, "0-3 m · Cream", vs[0].(map[string]any)["label"])
	assert.Equal(t, false, vs[1].(map[string]any)["in_stock"])
	assert.EqualValues(t, idOf(vs[0].(map[string]any)["id"]), idOf(p["default_variant_id"]), "the in-stock variant is preselected")
	seller := p["seller"].(map[string]any)
	assert.EqualValues(t, 4, seller["ship_max_days"])
	assert.EqualValues(t, 1000000, seller["free_shipping_over"])
	rev := p["reviews"].(map[string]any)
	items := rev["items"].([]any)
	assert.Equal(t, []any{"Sam", "Parnia"}, names(items, "author_name"), "published only, newest first")
	assert.Equal(t, true, items[1].(map[string]any)["verified_purchase"])
	assert.Equal(t, "3-6 m", items[1].(map[string]any)["variant_label"])
	together := names(p["bought_together"].([]any), "title")
	require.NotEmpty(t, together)
	assert.Equal(t, "Sleepsuit", together[0], "ordered together first, then category best sellers")
	assert.NotContains(t, together, "Long-sleeve bodysuit")

	r = e.do(fiber.MethodGet, "/api/v1/shop/catalog/products/"+itoa(f.sunscreen), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, []any{"oily", "combination"}, r.data()["product"].(map[string]any)["skin_types"])
	assert.Equal(t, "50 ml", r.data()["product"].(map[string]any)["variants"].([]any)[0].(map[string]any)["label"])

	for _, id := range []uint64{f.linkOut, f.inactiveSeller, 999999} {
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/catalog/products/"+itoa(id), sara, nil).Status, id)
	}

	// Reviews list.
	r = e.do(fiber.MethodGet, "/api/v1/shop/catalog/products/"+itoa(f.bodysuit)+"/reviews?per_page=1", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, []any{"Sam"}, names(r.data()["reviews"].([]any), "author_name"))
	assert.EqualValues(t, 2, r.data()["meta"].(map[string]any)["total"])
	assert.NotContains(t, r.Raw, "PENDING")
	assert.NotContains(t, r.Raw, "user_id")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/catalog/products/"+itoa(f.linkOut)+"/reviews", sara, nil).Status)
}

func TestShopCatalog_TeenIsForbidden(t *testing.T) {
	e := newCompanionEnv(t)
	nila := e.user("Nila", "09120008012")
	e.teenMode("Nila")
	f := seedCatalog(e)
	for _, p := range []string{"/api/v1/shop/catalog/home", "/api/v1/shop/catalog/products",
		"/api/v1/shop/catalog/products/" + itoa(f.bodysuit), "/api/v1/shop/catalog/products/" + itoa(f.bodysuit) + "/reviews"} {
		r := e.do(fiber.MethodGet, p, nila, nil)
		require.Equal(t, fiber.StatusForbidden, r.Status, p)
		assert.Equal(t, "teen_commercial_blocked", r.Body["error_code"], p)
		assert.Equal(t, "The shop is not available in teen mode.", r.Body["message"], p)
	}
}

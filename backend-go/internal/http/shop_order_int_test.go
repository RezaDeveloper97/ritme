package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/shop/order"
)

// CB-SHOP-03: cart, discount code, addresses, COD checkout, orders, cancel, returns (/api/v1/shop/*).

type orderFixture struct {
	sellerA, sellerB                  uint64
	bodysuit, sleepsuit, pads         uint64 // products
	vBody, vSleep, vPads, vGone       uint64 // variants (vGone: inactive)
	slotA, slotFull, slotOther, slotB uint64
	layette                           uint64 // shop_layette_items row for the clothes category
	clothes                           uint64
}

func seedOrderShop(e *companionEnv) orderFixture {
	var f orderFixture
	f.sellerA = e.exec(`INSERT INTO shop_sellers (slug, name, is_verified, ship_min_days, ship_max_days, shipping_fee, free_shipping_over, cod_max_total, return_days)
		VALUES ('panbe', '{"fa":"پوشاک پنبه ریز","en":"Panbe Kids"}', 1, 2, 4, 35000, 1000000, 5000000, 7)`)
	f.sellerB = e.exec(`INSERT INTO shop_sellers (slug, name, is_verified, ship_min_days, ship_max_days, shipping_fee, discreet_packaging, return_days)
		VALUES ('banoo', '{"fa":"بهداشتی بانو","en":"Banoo Care"}', 1, 1, 2, 35000, 1, 7)`)
	f.clothes = e.exec(`INSERT INTO shop_categories (slug, title, sort_order, is_active, parent_id)
		VALUES ('baby-clothes', '{"fa":"لباس نوزاد","en":"Baby clothes"}', 1, 1, (SELECT id FROM (SELECT id FROM shop_categories WHERE slug='layette') x))`)
	care := e.exec(`INSERT INTO shop_categories (slug, title, sort_order, is_active, parent_id)
		VALUES ('period-care', '{"fa":"بهداشتی","en":"Period care"}', 1, 1, (SELECT id FROM (SELECT id FROM shop_categories WHERE slug='beauty') x))`)
	product := func(cat, seller uint64, title string) uint64 {
		return e.exec(`INSERT INTO shop_products (category_id, seller_id, title, image_path, is_active, created_at, updated_at) VALUES (?, ?, ?, 'shop/p.jpg', 1, NOW(), NOW())`,
			cat, seller, title)
	}
	variant := func(p uint64, size, color any, price, stock, active int) uint64 {
		return e.exec(`INSERT INTO shop_product_variants (product_id, size, color, price, stock, is_active) VALUES (?, ?, ?, ?, ?, ?)`,
			p, size, color, price, stock, active)
	}
	f.bodysuit = product(f.clothes, f.sellerA, `{"fa":"بادی آستین بلند","en":"Long-sleeve bodysuit"}`)
	f.vBody = variant(f.bodysuit, `{"fa":"۳ تا ۶ ماه","en":"3-6 m"}`, `{"fa":"شیری","en":"Cream"}`, 485000, 5, 1)
	f.vGone = variant(f.bodysuit, `{"fa":"۰ تا ۳ ماه","en":"0-3 m"}`, nil, 485000, 5, 0)
	f.sleepsuit = product(f.clothes, f.sellerA, `{"fa":"سرهمی خواب","en":"Sleepsuit"}`)
	f.vSleep = variant(f.sleepsuit, `{"fa":"۳ تا ۶ ماه","en":"3-6 m"}`, `{"fa":"یاسی","en":"Lilac"}`, 390000, 3, 1)
	f.pads = product(care, f.sellerB, `{"fa":"نوار بهداشتی روزانه","en":"Daily pads"}`)
	f.vPads = variant(f.pads, nil, nil, 98000, 10, 1)
	f.slotA = e.exec(`INSERT INTO shop_delivery_slots (seller_id, slot_date, starts_at, ends_at, capacity) VALUES (?, '2026-09-26', '17:00', '21:00', 10)`, f.sellerA)
	f.slotFull = e.exec(`INSERT INTO shop_delivery_slots (seller_id, slot_date, starts_at, ends_at, capacity, booked) VALUES (?, '2026-09-26', '09:00', '13:00', 2, 2)`, f.sellerA)
	e.exec(`INSERT INTO shop_delivery_slots (seller_id, slot_date, starts_at, ends_at) VALUES (?, '2026-09-23', '17:00', '21:00')`, f.sellerA) // today: not offered
	f.slotB = e.exec(`INSERT INTO shop_delivery_slots (seller_id, slot_date, starts_at, ends_at) VALUES (?, '2026-09-24', '09:00', '13:00')`, f.sellerB)
	f.slotOther = f.slotB
	f.layette = e.exec(`INSERT INTO shop_layette_items (group_key, title, category_id, is_active) VALUES ('clothes', '{"fa":"بادی","en":"Bodysuit"}', ?, 1)`, f.clothes)
	e.exec(`INSERT INTO shop_discount_codes (code, kind, value, max_discount, min_items_total, max_uses, per_user_limit, is_active)
		VALUES ('WELCOME10', 'percent', 10, 50000, 200000, 100, 1, 1), ('BIG', 'fixed', 30000, NULL, 5000000, NULL, NULL, 1),
		       ('OFF', 'fixed', 30000, NULL, 0, NULL, NULL, 0)`)
	return f
}

func (e *companionEnv) addToCart(token string, variant uint64, qty int) reply {
	e.t.Helper()
	return e.do(fiber.MethodPost, "/api/v1/shop/cart/items", token, map[string]any{"variant_id": variant, "quantity": qty})
}

func (e *companionEnv) address(token string) uint64 {
	e.t.Helper()
	r := e.do(fiber.MethodPost, "/api/v1/shop/addresses", token, map[string]any{"label": "Home", "recipient_name": "Sara",
		"recipient_phone": "۰۹۱۲ ۱۲۳ ۴۵۶۷", "city": "Tehran", "address_line": "Valiasr St, No. 1", "postal_code": "1234567890"})
	require.Equal(e.t, fiber.StatusCreated, r.Status, r.Raw)
	return idOf(r.data()["address"].(map[string]any)["id"])
}

func TestShopOrder_CartCheckoutPlaceCancel(t *testing.T) {
	e := newCompanionEnv(t)
	for _, rt := range [][2]string{{fiber.MethodGet, "/api/v1/shop/cart"}, {fiber.MethodPost, "/api/v1/shop/cart/items"},
		{fiber.MethodGet, "/api/v1/shop/checkout"}, {fiber.MethodPost, "/api/v1/shop/orders"}, {fiber.MethodGet, "/api/v1/shop/orders/1"},
		{fiber.MethodGet, "/api/v1/shop/addresses"}, {fiber.MethodGet, "/api/v1/shop/returns"}} {
		assert.Equal(t, fiber.StatusUnauthorized, e.do(rt[0], rt[1], "", nil).Status, rt[1])
	}
	sara := e.user("Sara", "09120008101")
	f := seedOrderShop(e)
	uid := e.ids["Sara"]
	e.exec(`INSERT INTO shop_user_checklist_items (user_id, layette_item_id, group_key, state) VALUES (?, ?, 'clothes', 'todo')`, uid, f.layette)

	// Empty cart.
	r := e.do(fiber.MethodGet, "/api/v1/shop/cart", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.EqualValues(t, 0, r.data()["cart"].(map[string]any)["items_count"])

	// Add: inactive variant, over stock, over the line limit.
	r = e.addToCart(sara, f.vGone, 1)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "variant_unavailable", r.Body["error_code"])
	r = e.addToCart(sara, f.vSleep, 4)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "out_of_stock", r.Body["error_code"])
	r = e.addToCart(sara, f.vBody, 11)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "quantity")

	require.Equal(t, fiber.StatusOK, e.addToCart(sara, f.vBody, 1).Status)
	require.Equal(t, fiber.StatusOK, e.addToCart(sara, f.vSleep, 1).Status)
	r = e.addToCart(sara, f.vPads, 1)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	cart := r.data()["cart"].(map[string]any)
	assert.EqualValues(t, 3, cart["items_count"])
	assert.EqualValues(t, 2, cart["sellers_count"])
	assert.EqualValues(t, 973000, cart["items_total"])
	groups := cart["groups"].([]any)
	g0 := groups[0].(map[string]any)
	assert.Equal(t, "Panbe Kids", g0["seller"].(map[string]any)["name"])
	assert.EqualValues(t, 875000, g0["items_total"])
	assert.EqualValues(t, 125000, g0["free_shipping_remaining"])
	line := g0["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "3-6 m · Cream", line["variant_label"])
	assert.Equal(t, "Long-sleeve bodysuit", line["title"])

	// Quantity set / remove / re-add.
	r = e.do(fiber.MethodPut, "/api/v1/shop/cart/items/"+itoa(f.vPads), sara, map[string]any{"quantity": 2})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.EqualValues(t, 1071000, r.data()["cart"].(map[string]any)["items_total"])
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, "/api/v1/shop/cart/items/"+itoa(f.vPads), sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, "/api/v1/shop/cart/items/"+itoa(f.vPads), sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPut, "/api/v1/shop/cart/items/"+itoa(f.vPads), sara, map[string]any{"quantity": 1}).Status)
	require.Equal(t, fiber.StatusOK, e.addToCart(sara, f.vPads, 1).Status)

	// Discount codes.
	for code, want := range map[string]string{"NOPE": "discount_invalid", "OFF": "discount_invalid", "BIG": "discount_invalid"} {
		r = e.do(fiber.MethodPut, "/api/v1/shop/cart/discount", sara, map[string]any{"code": code})
		require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
		assert.Equal(t, want, r.Body["error_code"], code)
	}
	r = e.do(fiber.MethodPut, "/api/v1/shop/cart/discount", sara, map[string]any{"code": "welcome10"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	disc := r.data()["cart"].(map[string]any)["discount"].(map[string]any)
	assert.Equal(t, "WELCOME10", disc["code"])
	assert.EqualValues(t, 50000, disc["amount"], "10% capped at max_discount")

	// Checkout before an address: blocked.
	r = e.do(fiber.MethodGet, "/api/v1/shop/checkout", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, []any{"address_invalid"}, r.data()["blockers"])

	// Address: digits folded, validation.
	bad := e.do(fiber.MethodPost, "/api/v1/shop/addresses", sara, map[string]any{"recipient_name": "S", "recipient_phone": "123", "city": "T", "address_line": "x"})
	require.Equal(t, fiber.StatusUnprocessableEntity, bad.Status, bad.Raw)
	assert.Contains(t, bad.Body["errors"], "recipient_phone")
	addr := e.address(sara)
	r = e.do(fiber.MethodGet, "/api/v1/shop/addresses", sara, nil)
	a0 := r.data()["addresses"].([]any)[0].(map[string]any)
	assert.Equal(t, "09121234567", a0["recipient_phone"])
	assert.Equal(t, true, a0["is_default"], "the first address is the default")

	r = e.do(fiber.MethodGet, "/api/v1/shop/checkout", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	co := r.data()
	assert.Equal(t, true, co["can_place"])
	assert.EqualValues(t, addr, co["default_address_id"])
	assert.EqualValues(t, 70000, co["shipping_total"], "two shipments, neither over its free-shipping threshold")
	assert.EqualValues(t, 973000-50000+70000, co["payable_total"])
	assert.Equal(t, []any{map[string]any{"method": "cod", "available": true}}, co["payment_methods"])
	ships := co["shipments"].([]any)
	slotsA := ships[0].(map[string]any)["slots"].([]any)
	require.Len(t, slotsA, 1, "today's slot and the full slot are not offered")
	assert.Equal(t, map[string]any{"id": float64(f.slotA), "date": "2026-09-26", "starts_at": "17:00", "ends_at": "21:00"}, slotsA[0])

	// COD only.
	body := map[string]any{"idempotency_key": "key-0000001", "address_id": addr, "payment_method": "online",
		"discreet": true, "slots": []any{map[string]any{"seller_id": f.sellerA, "slot_id": f.slotA}, map[string]any{"seller_id": f.sellerB, "slot_id": f.slotB}}}
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, body)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "payment_method")
	// A full slot.
	body["payment_method"] = "cod"
	body["slots"] = []any{map[string]any{"seller_id": f.sellerA, "slot_id": f.slotFull}}
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, body)
	require.Equal(t, fiber.StatusConflict, r.Status, r.Raw)
	assert.Equal(t, "slot_full", r.Body["error_code"])
	assert.Equal(t, 5, e.count(`SELECT stock FROM shop_product_variants WHERE id = ?`, f.vBody), "a refused order takes no stock")
	// Another seller's slot.
	body["slots"] = []any{map[string]any{"seller_id": f.sellerA, "slot_id": f.slotB}}
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, body)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "slot_invalid", r.Body["error_code"])

	body["slots"] = []any{map[string]any{"seller_id": f.sellerA, "slot_id": f.slotA}, map[string]any{"seller_id": f.sellerB, "slot_id": f.slotB}}
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, body)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	o := r.data()["order"].(map[string]any)
	orderID := idOf(o["id"])
	assert.Equal(t, false, r.data()["replayed"])
	assert.Equal(t, "placed", o["status"])
	assert.Equal(t, "cod", o["payment_method"])
	assert.Equal(t, "unpaid", o["payment_status"])
	assert.Equal(t, true, o["discreet"])
	assert.Equal(t, true, o["can_cancel"])
	assert.Len(t, o["code"], 8)
	assert.EqualValues(t, 973000, o["items_total"])
	assert.EqualValues(t, 50000, o["discount_total"])
	assert.EqualValues(t, 70000, o["shipping_total"])
	assert.EqualValues(t, 993000, o["payable_total"])
	assert.EqualValues(t, 1, o["checklist_on_the_way"])
	assert.Equal(t, "09121234567", o["recipient"].(map[string]any)["phone"])
	oships := o["shipments"].([]any)
	require.Len(t, oships, 2)
	s0 := oships[0].(map[string]any)
	assert.Equal(t, "awaiting_seller", s0["status"])
	assert.Equal(t, map[string]any{"date": "2026-09-26", "starts_at": "17:00", "ends_at": "21:00"}, s0["slot"])
	item := s0["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "3-6 m · Cream", item["variant_label"])
	assert.EqualValues(t, 485000, item["unit_price"])
	assert.Equal(t, []any{"placed"}, names(o["timeline"].([]any), "status"))

	assert.Equal(t, 4, e.count(`SELECT stock FROM shop_product_variants WHERE id = ?`, f.vBody))
	assert.Equal(t, 1, e.count(`SELECT booked FROM shop_delivery_slots WHERE id = ?`, f.slotA))
	assert.Equal(t, 1, e.count(`SELECT used_count FROM shop_discount_codes WHERE code = 'WELCOME10'`))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM shop_cart_items`))
	assert.Equal(t, "on_the_way", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ?`, uid))
	assert.Equal(t, 1, e.count(`SELECT purchase_count FROM shop_products WHERE id = ?`, f.bodysuit))

	// Price snapshot: a later price change does not rewrite the order.
	e.exec(`UPDATE shop_product_variants SET price = 1 WHERE id = ?`, f.vBody)
	// Idempotent replay: same key → the same order, nothing new.
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, body)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["replayed"])
	assert.EqualValues(t, orderID, r.data()["order"].(map[string]any)["id"])
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM shop_orders`))
	// A new key with an empty cart.
	body["idempotency_key"] = "key-0000002"
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, body)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "cart_empty", r.Body["error_code"])

	// The code was used once: per-user limit.
	require.Equal(t, fiber.StatusOK, e.addToCart(sara, f.vSleep, 1).Status)
	r = e.do(fiber.MethodPut, "/api/v1/shop/cart/discount", sara, map[string]any{"code": "WELCOME10"})
	assert.Equal(t, "discount_invalid", r.Body["error_code"], "one refusal for every reason (security audit)")

	// List and detail.
	r = e.do(fiber.MethodGet, "/api/v1/shop/orders", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	list := r.data()["orders"].([]any)
	require.Len(t, list, 1)
	assert.EqualValues(t, 3, list[0].(map[string]any)["items_count"])
	r = e.do(fiber.MethodGet, "/api/v1/shop/orders/"+itoa(orderID), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.EqualValues(t, 485000, r.data()["order"].(map[string]any)["shipments"].([]any)[0].(map[string]any)["items"].([]any)[0].(map[string]any)["unit_price"])

	// Cancel: stock, slot, code use and checklist given back; a second cancel is refused.
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders/"+itoa(orderID)+"/cancel", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	o = r.data()["order"].(map[string]any)
	assert.Equal(t, "cancelled", o["status"])
	assert.Equal(t, false, o["can_cancel"])
	assert.Equal(t, []any{"placed", "cancelled"}, names(o["timeline"].([]any), "status"))
	assert.Equal(t, 5, e.count(`SELECT stock FROM shop_product_variants WHERE id = ?`, f.vBody))
	assert.Equal(t, 0, e.count(`SELECT booked FROM shop_delivery_slots WHERE id = ?`, f.slotA))
	assert.Equal(t, 0, e.count(`SELECT used_count FROM shop_discount_codes WHERE code = 'WELCOME10'`))
	assert.Equal(t, 0, e.count(`SELECT purchase_count FROM shop_products WHERE id = ?`, f.bodysuit))
	assert.Equal(t, "todo", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ?`, uid))
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders/"+itoa(orderID)+"/cancel", sara, nil)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "not_cancellable", r.Body["error_code"])
}

func (e *companionEnv) str(q string, args ...any) string {
	e.t.Helper()
	var s string
	require.NoError(e.t, e.db.QueryRow(q, args...).Scan(&s))
	return s
}

func TestShopOrder_DeliveryAndReturns(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008111")
	f := seedOrderShop(e)
	uid := e.ids["Sara"]
	e.exec(`INSERT INTO shop_user_checklist_items (user_id, layette_item_id, group_key, state) VALUES (?, ?, 'clothes', 'in_cart')`, uid, f.layette)
	require.Equal(t, fiber.StatusOK, e.addToCart(sara, f.vBody, 2).Status)
	addr := e.address(sara)
	r := e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, map[string]any{"idempotency_key": "deliver-001", "address_id": addr, "payment_method": "cod"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	o := r.data()["order"].(map[string]any)
	orderID := idOf(o["id"])
	ship := o["shipments"].([]any)[0].(map[string]any)
	shipID := idOf(ship["id"])
	itemID := idOf(ship["items"].([]any)[0].(map[string]any)["id"])
	assert.Nil(t, ship["slot"])

	ret := func(body map[string]any, now ...string) reply {
		return e.do(fiber.MethodPost, "/api/v1/shop/orders/"+itoa(orderID)+"/returns", sara, body, now...)
	}
	r = ret(map[string]any{"order_item_id": itemID, "reason": "wrong_size"})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "not_returnable", r.Body["error_code"], "not delivered yet")

	delivered := time.Date(2026, 9, 25, 12, 0, 0, 0, time.FixedZone("Tehran", 3*3600+1800))
	svc := order.NewService(e.db)
	require.NoError(t, svc.MarkShipmentDelivered(context.Background(), shipID, delivered))
	require.NoError(t, svc.MarkShipmentDelivered(context.Background(), shipID, delivered), "idempotent")
	assert.Equal(t, "have", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ?`, uid))
	assert.Equal(t, "delivered", e.str(`SELECT status FROM shop_orders WHERE id = ?`, orderID))

	r = e.do(fiber.MethodGet, "/api/v1/shop/orders/"+itoa(orderID), sara, nil, "2026-09-26T10:00:00+03:30")
	o = r.data()["order"].(map[string]any)
	ship = o["shipments"].([]any)[0].(map[string]any)
	assert.Equal(t, "2026-10-02T12:00:00+03:30", ship["return_until"])
	assert.Equal(t, true, ship["items"].([]any)[0].(map[string]any)["returnable"])
	assert.Equal(t, false, o["can_cancel"])
	assert.Equal(t, []any{"awaiting_seller", "delivered"}, names(ship["timeline"].([]any), "status"))

	r = ret(map[string]any{"order_item_id": itemID, "reason": "bored"}, "2026-09-26T10:00:00+03:30")
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "reason")
	r = ret(map[string]any{"order_item_id": itemID, "quantity": 3, "reason": "wrong_size"}, "2026-09-26T10:00:00+03:30")
	assert.Equal(t, "return_quantity", r.Body["error_code"])
	r = ret(map[string]any{"order_item_id": itemID, "quantity": 2, "reason": "wrong_size", "note": "Too small"}, "2026-09-26T10:00:00+03:30")
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, "requested", r.data()["return"].(map[string]any)["status"])
	r = ret(map[string]any{"order_item_id": itemID, "reason": "wrong_size"}, "2026-09-26T10:00:00+03:30")
	assert.Equal(t, "return_quantity", r.Body["error_code"], "everything already returned")
	r = ret(map[string]any{"order_item_id": itemID, "reason": "damaged"}, "2026-10-03T10:00:00+03:30")
	assert.Equal(t, "return_window_closed", r.Body["error_code"])

	r = e.do(fiber.MethodGet, "/api/v1/shop/returns", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["returns"], 1)
}

func TestShopOrder_OwnerScopedAndTeen(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008121")
	mina := e.user("Mina", "09120008122")
	nila := e.user("Nila", "09120008123")
	e.teenMode("Nila")
	f := seedOrderShop(e)
	require.Equal(t, fiber.StatusOK, e.addToCart(sara, f.vBody, 1).Status)
	addr := e.address(sara)
	r := e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, map[string]any{"idempotency_key": "sara-0001", "address_id": addr, "payment_method": "cod"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	orderID := idOf(r.data()["order"].(map[string]any)["id"])
	itemID := idOf(r.data()["order"].(map[string]any)["shipments"].([]any)[0].(map[string]any)["items"].([]any)[0].(map[string]any)["id"])

	// Mina sees none of Sara's rows and cannot touch them.
	for _, rt := range []struct {
		method, path string
		body         any
	}{
		{fiber.MethodGet, "/api/v1/shop/orders/" + itoa(orderID), nil},
		{fiber.MethodPost, "/api/v1/shop/orders/" + itoa(orderID) + "/cancel", nil},
		{fiber.MethodPost, "/api/v1/shop/orders/" + itoa(orderID) + "/returns", map[string]any{"order_item_id": itemID, "reason": "damaged"}},
		{fiber.MethodPut, "/api/v1/shop/addresses/" + itoa(addr), map[string]any{"recipient_name": "X", "recipient_phone": "09120000000", "city": "X", "address_line": "X"}},
		{fiber.MethodDelete, "/api/v1/shop/addresses/" + itoa(addr), nil},
	} {
		r := e.do(rt.method, rt.path, mina, rt.body)
		assert.Equal(t, fiber.StatusNotFound, r.Status, rt.path+" "+r.Raw)
	}
	r = e.do(fiber.MethodGet, "/api/v1/shop/orders", mina, nil)
	assert.Empty(t, r.data()["orders"])
	r = e.do(fiber.MethodGet, "/api/v1/shop/addresses", mina, nil)
	assert.Empty(t, r.data()["addresses"])
	require.Equal(t, fiber.StatusOK, e.addToCart(mina, f.vPads, 1).Status)
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", mina, map[string]any{"idempotency_key": "mina-0001", "address_id": addr, "payment_method": "cod"})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "address_invalid", r.Body["error_code"], "someone else's address is not hers")
	// The same idempotency key from another user is another order.
	minaAddr := e.address(mina)
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", mina, map[string]any{"idempotency_key": "sara-0001", "address_id": minaAddr, "payment_method": "cod"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, "Sara", e.str(`SELECT recipient_name FROM shop_orders WHERE id = ?`, orderID), "Sara's order untouched")
	assert.Equal(t, "placed", e.str(`SELECT status FROM shop_orders WHERE id = ?`, orderID))

	for _, rt := range [][2]string{{fiber.MethodGet, "/api/v1/shop/cart"}, {fiber.MethodPost, "/api/v1/shop/cart/items"},
		{fiber.MethodGet, "/api/v1/shop/checkout"}, {fiber.MethodPost, "/api/v1/shop/orders"}, {fiber.MethodGet, "/api/v1/shop/orders"},
		{fiber.MethodGet, "/api/v1/shop/addresses"}} {
		r := e.do(rt[0], rt[1], nila, map[string]any{})
		require.Equal(t, fiber.StatusForbidden, r.Status, rt[1])
		assert.Equal(t, "teen_commercial_blocked", r.Body["error_code"], rt[1])
	}
}

// raw is e.do without testify (safe from goroutines): status and error_code.
func (e *companionEnv) raw(method, path, token string, body any) (int, string) {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, companionNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	if err != nil {
		return 0, err.Error()
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	code, _ := out["error_code"].(string)
	return resp.StatusCode, code
}

// TestShopOrder_NoOversellUnderConcurrency: 8 buyers race for 3 units, 6 buyers race for a 2-place delivery slot,
// and one buyer double-taps «ثبت سفارش» 5 times with one idempotency key.
func TestShopOrder_NoOversellUnderConcurrency(t *testing.T) {
	e := newCompanionEnv(t)
	f := seedOrderShop(e)
	e.exec(`UPDATE shop_product_variants SET stock = 3 WHERE id = ?`, f.vBody)
	e.exec(`UPDATE shop_product_variants SET stock = 100 WHERE id = ?`, f.vPads)
	tightSlot := e.exec(`INSERT INTO shop_delivery_slots (seller_id, slot_date, starts_at, ends_at, capacity) VALUES (?, '2026-09-27', '10:00', '14:00', 2)`, f.sellerB)

	type buyer struct {
		token string
		addr  uint64
	}
	newBuyers := func(prefix string, n int, variant uint64) []buyer {
		out := make([]buyer, n)
		for i := range out {
			group := map[string]string{"S": "1", "T": "2", "D": "3"}[prefix]
			tok := e.user(prefix+itoa(uint64(i)), "09130"+group+itoa(uint64(10000+i))) //nolint:gosec // test
			require.Equal(t, fiber.StatusOK, e.addToCart(tok, variant, 1).Status)
			out[i] = buyer{token: tok, addr: e.address(tok)}
		}
		return out
	}
	race := func(buyers []buyer, body func(b buyer) map[string]any) map[string]int {
		var mu sync.Mutex
		got := map[string]int{}
		var wg sync.WaitGroup
		start := make(chan struct{})
		for _, b := range buyers {
			wg.Add(1)
			go func(b buyer) {
				defer wg.Done()
				<-start
				status, code := e.raw(fiber.MethodPost, "/api/v1/shop/orders", b.token, body(b))
				mu.Lock()
				got[itoa(uint64(status))+" "+code]++ //nolint:gosec // test
				mu.Unlock()
			}(b)
		}
		close(start)
		wg.Wait()
		return got
	}

	stockBuyers := newBuyers("S", 8, f.vBody)
	got := race(stockBuyers, func(b buyer) map[string]any {
		return map[string]any{"idempotency_key": "stock-race-1", "address_id": b.addr, "payment_method": "cod"}
	})
	assert.Equal(t, 3, got["201 "], got)
	assert.Equal(t, 5, got["409 out_of_stock"]+got["409 busy"], got)
	assert.Equal(t, 0, e.count(`SELECT stock FROM shop_product_variants WHERE id = ?`, f.vBody), "never below zero")
	assert.Equal(t, 3, e.count(`SELECT COALESCE(SUM(quantity), 0) FROM shop_order_items WHERE variant_id = ?`, f.vBody))

	slotBuyers := newBuyers("T", 6, f.vPads)
	got = race(slotBuyers, func(b buyer) map[string]any {
		return map[string]any{"idempotency_key": "slot-race-1", "address_id": b.addr, "payment_method": "cod",
			"slots": []any{map[string]any{"seller_id": f.sellerB, "slot_id": tightSlot}}}
	})
	assert.Equal(t, 2, got["201 "], got)
	assert.Equal(t, 2, e.count(`SELECT booked FROM shop_delivery_slots WHERE id = ?`, tightSlot))
	assert.Equal(t, 2, e.count(`SELECT COUNT(*) FROM shop_order_shipments WHERE delivery_slot_id = ?`, tightSlot))
	assert.Equal(t, 100-2, e.count(`SELECT stock FROM shop_product_variants WHERE id = ?`, f.vPads), "refused orders give their stock back")

	tap := newBuyers("D", 1, f.vPads)[0]
	taps := []buyer{tap, tap, tap, tap, tap}
	got = race(taps, func(b buyer) map[string]any {
		return map[string]any{"idempotency_key": "double-tap-1", "address_id": b.addr, "payment_method": "cod"}
	})
	assert.Equal(t, 1, got["201 "], got)
	assert.Equal(t, 4, got["200 "], got)
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM shop_orders WHERE user_id = ?`, e.ids["D0"]))
}

// Security audit (directory/shop) M2: applying discount codes has a small hourly per-user budget (localized 429), and
// a user holds at most order.MaxOpenOrders placed-but-unshipped orders. L5: cancelling one order leaves the checklist
// rows another open order still carries «در راه».
func TestShopOrder_DiscountLimitAndOpenOrderCap(t *testing.T) {
	e := newCompanionEnv(t)
	f := seedOrderShop(e)
	e.exec(`UPDATE shop_product_variants SET stock = 100 WHERE id IN (?, ?, ?)`, f.vBody, f.vSleep, f.vPads)

	guess := e.user("Guess", "09120008131")
	for i := range ShopDiscountMax {
		r := e.do(fiber.MethodPut, "/api/v1/shop/cart/discount", guess, map[string]any{"code": "TRY" + itoa(uint64(i))}) //nolint:gosec // test
		require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
		assert.Equal(t, "discount_invalid", r.Body["error_code"])
	}
	r := e.do(fiber.MethodPut, "/api/v1/shop/cart/discount", guess, map[string]any{"code": "WELCOME10"})
	require.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	assert.Equal(t, "too_many_requests", r.Body["error_code"])
	assert.Equal(t, "Too many discount code tries. Try again later.", r.Body["message"])

	lina := e.user("Lina", "09120008132")
	uid := e.ids["Lina"]
	e.exec(`INSERT INTO shop_user_checklist_items (user_id, layette_item_id, group_key, state) VALUES (?, ?, 'clothes', 'todo')`, uid, f.layette)
	addr := e.address(lina)
	place := func(variant uint64, key string) reply {
		e.t.Helper()
		require.Equal(t, fiber.StatusOK, e.addToCart(lina, variant, 1).Status)
		return e.do(fiber.MethodPost, "/api/v1/shop/orders", lina, map[string]any{"idempotency_key": key, "address_id": addr, "payment_method": "cod"})
	}
	var ids []uint64
	for i, v := range []uint64{f.vBody, f.vSleep, f.vPads, f.vPads, f.vPads} { // two clothes orders, three pads orders
		r = place(v, "cap-000"+itoa(uint64(i))) //nolint:gosec // test
		require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
		ids = append(ids, idOf(r.data()["order"].(map[string]any)["id"]))
	}
	assert.Equal(t, "on_the_way", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ?`, uid))
	r = place(f.vPads, "cap-0006")
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "too_many_open_orders", r.Body["error_code"])
	assert.Equal(t, "You have too many orders waiting to ship. Wait for one to ship or cancel one first.", r.Body["message"])
	assert.Equal(t, 5, e.count(`SELECT COUNT(*) FROM shop_orders WHERE user_id = ?`, uid))

	// A shipped order no longer counts as open.
	e.exec(`UPDATE shop_order_shipments SET status = 'shipped' WHERE order_id = ?`, ids[4])
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", lina, map[string]any{"idempotency_key": "cap-0006", "address_id": addr, "payment_method": "cod"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)

	// L5: the first clothes order is cancelled, the second still carries the category → the row stays «در راه».
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/shop/orders/"+itoa(ids[0])+"/cancel", lina, nil).Status)
	assert.Equal(t, "on_the_way", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ?`, uid))
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/shop/orders/"+itoa(ids[1])+"/cancel", lina, nil).Status)
	assert.Equal(t, "todo", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ?`, uid))
}

package http

import (
	"context"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/shop/order"
)

// CB-SHOP-12 follow-ups: F-5 the user's own product review of a delivered order item, F-6 an order also creates the
// layette rows she never touched («در راه», then «دارم» on delivery, back to «باید بخرم» on cancel).

func TestShopOrder_ReviewDeliveredItem(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008131")
	mina := e.user("Mina", "09120008132")
	nila := e.user("Nila", "09120008133")
	e.teenMode("Nila")
	f := seedOrderShop(e)
	uid := e.ids["Sara"]

	// F-6: Sara never opened the layette checklist; the order still puts the matching template row «در راه».
	require.Equal(t, fiber.StatusOK, e.addToCart(sara, f.vBody, 1).Status)
	addr := e.address(sara)
	r := e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, map[string]any{"idempotency_key": "review-001", "address_id": addr, "payment_method": "cod"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	o := r.data()["order"].(map[string]any)
	orderID := idOf(o["id"])
	ship := o["shipments"].([]any)[0].(map[string]any)
	shipID := idOf(ship["id"])
	item := ship["items"].([]any)[0].(map[string]any)
	itemID := idOf(item["id"])
	assert.EqualValues(t, 1, o["checklist_on_the_way"])
	assert.Equal(t, "on_the_way", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ? AND layette_item_id = ?`, uid, f.layette))
	assert.Equal(t, false, item["reviewable"])
	assert.Nil(t, item["review_status"])

	path := "/api/v1/shop/orders/" + itoa(orderID) + "/items/" + itoa(itemID) + "/review"
	r = e.do(fiber.MethodPost, path, sara, map[string]any{"rating": 5})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "not_reviewable", r.Body["error_code"], "not delivered yet")

	delivered := time.Date(2026, 9, 25, 12, 0, 0, 0, time.FixedZone("Tehran", 3*3600+1800))
	require.NoError(t, order.NewService(e.db).MarkShipmentDelivered(context.Background(), shipID, delivered))
	assert.Equal(t, "have", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ? AND layette_item_id = ?`, uid, f.layette))

	r = e.do(fiber.MethodGet, "/api/v1/shop/orders/"+itoa(orderID), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	item = r.data()["order"].(map[string]any)["shipments"].([]any)[0].(map[string]any)["items"].([]any)[0].(map[string]any)
	assert.Equal(t, true, item["reviewable"])

	// Validation, ownership, teen.
	r = e.do(fiber.MethodPost, path, sara, map[string]any{"rating": 6})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "rating")
	r = e.do(fiber.MethodPost, path, sara, map[string]any{"body": "Soft"})
	assert.Contains(t, r.Body["errors"], "rating", "rating is required")
	r = e.do(fiber.MethodPost, path, mina, map[string]any{"rating": 5})
	assert.Equal(t, fiber.StatusNotFound, r.Status, "someone else's order: "+r.Raw)
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders/"+itoa(orderID)+"/items/999999/review", sara, map[string]any{"rating": 5})
	assert.Equal(t, fiber.StatusNotFound, r.Status, "an item not in her order: "+r.Raw)
	r = e.do(fiber.MethodPost, path, nila, map[string]any{"rating": 5})
	require.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Equal(t, "teen_commercial_blocked", r.Body["error_code"])

	// The review: pending, verified purchase, the author is her initial only.
	r = e.do(fiber.MethodPost, path, sara, map[string]any{"rating": 4, "body": "  Soft and warm  "})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	rev := r.data()["review"].(map[string]any)
	assert.Equal(t, "pending", rev["status"])
	assert.EqualValues(t, 4, rev["rating"])
	assert.Equal(t, "Soft and warm", rev["body"])
	assert.EqualValues(t, itemID, rev["order_item_id"])
	assert.EqualValues(t, f.bodysuit, rev["product_id"])
	assert.Equal(t, "S.", e.str(`SELECT author_name FROM shop_product_reviews WHERE id = ?`, idOf(rev["id"])))
	assert.NotEmpty(t, e.str(`SELECT variant_label FROM shop_product_reviews WHERE id = ?`, idOf(rev["id"])), "variant snapshot kept")

	r = e.do(fiber.MethodPost, path, sara, map[string]any{"rating": 2})
	require.Equal(t, fiber.StatusConflict, r.Status, r.Raw)
	assert.Equal(t, "already_reviewed", r.Body["error_code"])

	r = e.do(fiber.MethodGet, "/api/v1/shop/orders/"+itoa(orderID), sara, nil)
	item = r.data()["order"].(map[string]any)["shipments"].([]any)[0].(map[string]any)["items"].([]any)[0].(map[string]any)
	assert.Equal(t, false, item["reviewable"])
	assert.Equal(t, "pending", item["review_status"])

	// Pending reviews do not reach the catalog.
	r = e.do(fiber.MethodGet, "/api/v1/shop/catalog/products/"+itoa(f.bodysuit)+"/reviews", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Empty(t, r.data()["reviews"])
}

func TestShopOrder_UntouchedChecklistRowRevertsOnCancel(t *testing.T) {
	e := newCompanionEnv(t)
	mina := e.user("Mina", "09120008141")
	f := seedOrderShop(e)
	uid := e.ids["Mina"]
	require.Equal(t, fiber.StatusOK, e.addToCart(mina, f.vBody, 1).Status)
	addr := e.address(mina)
	r := e.do(fiber.MethodPost, "/api/v1/shop/orders", mina, map[string]any{"idempotency_key": "untouched-01", "address_id": addr, "payment_method": "cod"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	orderID := idOf(r.data()["order"].(map[string]any)["id"])
	assert.Equal(t, "on_the_way", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ? AND layette_item_id = ?`, uid, f.layette))

	r = e.do(fiber.MethodPost, "/api/v1/shop/orders/"+itoa(orderID)+"/cancel", mina, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "todo", e.str(`SELECT state FROM shop_user_checklist_items WHERE user_id = ? AND layette_item_id = ?`, uid, f.layette))
	assert.Equal(t, "1", e.str(`SELECT COUNT(*) FROM shop_user_checklist_items WHERE user_id = ?`, uid), "one row, no duplicate")
}

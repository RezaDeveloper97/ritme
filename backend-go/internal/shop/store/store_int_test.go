package store_test

// CB-SHOP-01: the 00080 schema and its typed store — stock and slot guards, cart upsert, checklist upsert, link-out rows
// untouched, and user rows cascading on account deletion.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/shop/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func exec(t *testing.T, db *sql.DB, q string, args ...any) uint64 {
	t.Helper()
	res, err := db.Exec(q, args...)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // auto-increment ids are positive
}

func TestShopCommerceSchema(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	q := store.New(db)
	now := sql.NullTime{Time: time.Date(2026, 10, 11, 10, 0, 0, 0, time.Local), Valid: true}

	userID := exec(t, db, "INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120000080', NOW(), NOW())")
	var layette uint64
	require.NoError(t, db.QueryRow("SELECT id FROM shop_categories WHERE slug = 'layette'").Scan(&layette))
	sub := exec(t, db, `INSERT INTO shop_categories (slug, title, parent_id, sort_order, is_active) VALUES ('baby-clothes', '{"fa":"لباس نوزاد","en":"Baby clothes"}', ?, 1, 1)`, layette)
	seller := exec(t, db, `INSERT INTO shop_sellers (slug, name, is_verified, ship_min_days, ship_max_days) VALUES ('panbe', '{"fa":"پوشاک پنبه","en":"Panbe"}', 1, 2, 4)`)
	product := exec(t, db, `INSERT INTO shop_products (category_id, title, seller_id) VALUES (?, '{"fa":"بادی","en":"Bodysuit"}', ?)`, sub, seller)
	linkOut := exec(t, db, `INSERT INTO shop_products (category_id, title, price_text, external_url) VALUES (?, '{"fa":"x","en":"x"}', '۱۰۰ هزار', 'https://example.com/p')`, layette)
	variant := exec(t, db, `INSERT INTO shop_product_variants (product_id, size, price, compare_at_price, stock) VALUES (?, '{"fa":"۳ تا ۶ ماه","en":"3-6 m"}', 485000, NULL, 2)`, product)

	subs, err := q.ListSubCategories(ctx, sql.NullInt64{Int64: int64(layette), Valid: true}) //nolint:gosec // small id
	require.NoError(t, err)
	require.Len(t, subs, 1)

	// Link-out rows keep their columns and have no variants.
	n, err := q.CountActiveVariantsByProduct(ctx, linkOut)
	require.NoError(t, err)
	assert.Zero(t, n)

	// Stock never goes negative.
	rows, err := q.DecrementVariantStock(ctx, store.DecrementVariantStockParams{Quantity: 3, UpdatedAt: now, ID: variant})
	require.NoError(t, err)
	assert.Zero(t, rows)
	rows, err = q.DecrementVariantStock(ctx, store.DecrementVariantStockParams{Quantity: 2, UpdatedAt: now, ID: variant})
	require.NoError(t, err)
	assert.EqualValues(t, 1, rows)

	// One cart per user; a variant appears once per cart.
	cartID, err := q.CreateCart(ctx, store.CreateCartParams{UserID: userID, CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)
	for _, qty := range []uint16{1, 3} {
		require.NoError(t, q.UpsertCartItem(ctx, store.UpsertCartItemParams{CartID: uint64(cartID), VariantID: variant, Quantity: qty, CreatedAt: now, UpdatedAt: now})) //nolint:gosec // positive id
	}
	items, err := q.ListCartItems(ctx, uint64(cartID)) //nolint:gosec // positive id
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.EqualValues(t, 3, items[0].Quantity)

	// Slot capacity is a hard ceiling.
	slot := exec(t, db, "INSERT INTO shop_delivery_slots (seller_id, slot_date, starts_at, ends_at, capacity) VALUES (?, '2026-10-12', '09:00', '13:00', 1)", seller)
	rows, err = q.BookDeliverySlot(ctx, store.BookDeliverySlotParams{UpdatedAt: now, ID: slot})
	require.NoError(t, err)
	assert.EqualValues(t, 1, rows)
	rows, err = q.BookDeliverySlot(ctx, store.BookDeliverySlotParams{UpdatedAt: now, ID: slot})
	require.NoError(t, err)
	assert.Zero(t, rows)

	// COD order with a per-seller shipment and a snapshot line.
	order := exec(t, db, `INSERT INTO shop_orders (code, user_id, recipient_name, recipient_phone, city, address_text, items_total, payable_total, discreet)
		VALUES ('520841', ?, 'Sara', '09120000080', 'Tehran', 'x', 485000, 485000, 1)`, userID)
	shipment := exec(t, db, "INSERT INTO shop_order_shipments (order_id, seller_id, delivery_slot_id, items_total) VALUES (?, ?, ?, 485000)", order, seller, slot)
	exec(t, db, `INSERT INTO shop_order_items (order_id, shipment_id, product_id, variant_id, title, unit_price, quantity, line_total) VALUES (?, ?, ?, ?, '{"fa":"بادی"}', 485000, 1, 485000)`, order, shipment, product, variant)
	got, err := q.GetOrderByCodeForUser(ctx, store.GetOrderByCodeForUserParams{Code: "520841", UserID: userID})
	require.NoError(t, err)
	assert.Equal(t, "cod", got.PaymentMethod)
	assert.Equal(t, "placed", got.Status)
	assert.True(t, got.Discreet)

	// Layette checklist: one row per template item per user.
	tpl := exec(t, db, `INSERT INTO shop_layette_items (group_key, title, is_essential) VALUES ('clothing', '{"fa":"بادی"}', 1)`)
	for _, state := range []string{"in_cart", "have"} {
		require.NoError(t, q.SetUserChecklistState(ctx, store.SetUserChecklistStateParams{
			UserID: userID, LayetteItemID: sql.NullInt64{Int64: int64(tpl), Valid: true}, GroupKey: "clothing", State: state, CreatedAt: now, UpdatedAt: now, //nolint:gosec // small id
		}))
	}
	list, err := q.ListUserChecklistItems(ctx, userID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "have", list[0].State)

	// Account deletion removes every user-owned shop row.
	_, err = db.Exec("DELETE FROM users WHERE id = ?", userID)
	require.NoError(t, err)
	for _, table := range []string{"shop_carts", "shop_orders", "shop_user_checklist_items"} {
		var c int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&c)) //nolint:gosec // fixed table names
		assert.Zero(t, c, table)
	}
}

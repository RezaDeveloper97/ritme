package http

import (
	"database/sql"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Buyer PII at rest (user decision 2026-10-12, migration 00091): saved addresses and the order recipient snapshot are
// sealed in pii_enc; the plaintext columns stay empty; the owner reads them decrypted; legacy plain rows still read.
func TestShopPII_EncryptedAtRest(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008191")
	f := seedOrderShop(e)
	uid := e.ids["Sara"]

	addr := e.address(sara)
	var name, phone, line string
	var label, postal, enc sql.NullString
	require.NoError(t, e.db.QueryRow(`SELECT label, recipient_name, recipient_phone, address_line, postal_code, pii_enc FROM shop_addresses WHERE id = ?`, addr).
		Scan(&label, &name, &phone, &line, &postal, &enc))
	assert.False(t, label.Valid)
	assert.Empty(t, name)
	assert.Empty(t, phone)
	assert.Empty(t, line)
	assert.False(t, postal.Valid)
	require.True(t, enc.Valid)
	for _, plain := range []string{"Sara", "0912", "1234567", "Valiasr", "Home", "1234567890"} {
		assert.NotContains(t, enc.String, plain)
	}
	assert.Equal(t, "Tehran", e.str(`SELECT city FROM shop_addresses WHERE id = ?`, addr), "city stays plain")

	r := e.do(fiber.MethodGet, "/api/v1/shop/addresses", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	a0 := r.data()["addresses"].([]any)[0].(map[string]any)
	assert.Equal(t, "Home", a0["label"])
	assert.Equal(t, "Sara", a0["recipient_name"])
	assert.Equal(t, "09121234567", a0["recipient_phone"])
	assert.Equal(t, "Valiasr St, No. 1", a0["address_line"])
	assert.Equal(t, "1234567890", a0["postal_code"])

	// Checkout lists the address decrypted; the placed order's snapshot is sealed too.
	r = e.do(fiber.MethodGet, "/api/v1/shop/checkout", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Contains(t, r.Raw, "09121234567")
	require.Equal(t, fiber.StatusOK, e.addToCart(sara, f.vBody, 1).Status)
	r = e.do(fiber.MethodPost, "/api/v1/shop/orders", sara, map[string]any{"idempotency_key": "pii-0001", "address_id": addr, "payment_method": "cod"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	o := r.data()["order"].(map[string]any)
	orderID := idOf(o["id"])
	rec := o["recipient"].(map[string]any)
	assert.Equal(t, "Sara", rec["name"])
	assert.Equal(t, "09121234567", rec["phone"])
	assert.Equal(t, "Valiasr St, No. 1", rec["address"])
	assert.Equal(t, "1234567890", rec["postal_code"])
	var oname, ophone, otext string
	var opostal, oenc sql.NullString
	require.NoError(t, e.db.QueryRow(`SELECT recipient_name, recipient_phone, address_text, postal_code, pii_enc FROM shop_orders WHERE id = ?`, orderID).
		Scan(&oname, &ophone, &otext, &opostal, &oenc))
	assert.Empty(t, oname)
	assert.Empty(t, ophone)
	assert.Empty(t, otext)
	assert.False(t, opostal.Valid)
	require.True(t, oenc.Valid)
	assert.NotContains(t, oenc.String, "0912")
	assert.NotContains(t, oenc.String, "Valiasr")
	r = e.do(fiber.MethodGet, "/api/v1/shop/orders/"+itoa(orderID), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "09121234567", r.data()["order"].(map[string]any)["recipient"].(map[string]any)["phone"])

	// Legacy plain rows (written before 00091) still read; editing a legacy address seals it and clears the plaintext.
	legacy := e.exec(`INSERT INTO shop_addresses (user_id, label, recipient_name, recipient_phone, city, address_line, postal_code, is_default, created_at, updated_at)
		VALUES (?, 'Work', 'Sara L', '09125550000', 'Karaj', 'Old Rd 5', '9998887776', 0, NOW(), NOW())`, uid)
	legacyOrder := e.exec(`INSERT INTO shop_orders (code, user_id, recipient_name, recipient_phone, city, address_text, postal_code, items_total, payable_total, placed_at, created_at, updated_at)
		VALUES ('87654321', ?, 'Sara L', '09125550000', 'Karaj', 'Old Rd 5', '9998887776', 100, 100, NOW(), NOW(), NOW())`, uid)
	r = e.do(fiber.MethodGet, "/api/v1/shop/addresses", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Contains(t, r.Raw, "09125550000")
	assert.Contains(t, r.Raw, "09121234567")
	r = e.do(fiber.MethodGet, "/api/v1/shop/orders/"+itoa(legacyOrder), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	lrec := r.data()["order"].(map[string]any)["recipient"].(map[string]any)
	assert.Equal(t, "Sara L", lrec["name"])
	assert.Equal(t, "09125550000", lrec["phone"])
	assert.Equal(t, "Old Rd 5", lrec["address"])

	r = e.do(fiber.MethodPut, "/api/v1/shop/addresses/"+itoa(legacy), sara, map[string]any{"label": "Work", "recipient_name": "Sara L",
		"recipient_phone": "09125550000", "city": "Karaj", "address_line": "Old Rd 5"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "09125550000", r.data()["address"].(map[string]any)["recipient_phone"])
	assert.Empty(t, e.str(`SELECT recipient_phone FROM shop_addresses WHERE id = ?`, legacy), "edit clears the legacy plaintext")
	assert.Equal(t, "1", e.str(`SELECT COUNT(*) FROM shop_addresses WHERE id = ? AND pii_enc IS NOT NULL`, legacy))

	// A blob moved onto another user's row does not open (bound to the owner): fail closed.
	mina := e.user("Mina", "09120008192")
	minaAddr := e.address(mina)
	e.exec(`UPDATE shop_addresses SET pii_enc = ? WHERE id = ?`, enc.String, minaAddr)
	r = e.do(fiber.MethodGet, "/api/v1/shop/addresses", mina, nil)
	assert.Equal(t, fiber.StatusServiceUnavailable, r.Status, r.Raw)
	assert.Equal(t, "storage_unavailable", r.Body["error_code"])
}

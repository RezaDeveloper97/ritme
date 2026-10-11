package http

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shopProduct(t *testing.T, e *companionEnv, cat, title, url string, active int) uint64 {
	t.Helper()
	var ext any
	if url != "" {
		ext = url
	}
	res, err := e.db.Exec(`INSERT INTO shop_products (category_id, title, price_text, badge, external_url, partner_label, sort_order, is_active, created_at, updated_at)
		VALUES ((SELECT id FROM shop_categories WHERE slug = ?), ?, '350 T', 'new', ?, 'Partner', 1, ?, NOW(), NOW())`,
		cat, `{"fa":"`+title+`","en":"`+title+`"}`, ext, active)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	return uint64(id) //nolint:gosec // test ids
}

func TestShop_PublicCatalogAndClicks(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008001")

	for _, p := range []string{"/api/v1/shop/categories", "/api/v1/shop/products", "/api/v1/shop/products/1"} {
		assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, p, "", nil).Status, p)
	}
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, "/api/v1/shop/products/1/click", "", nil).Status)

	link := shopProduct(t, e, "layette", "Swaddle", "https://partner.example/p/1?ref=a", 1)
	plain := shopProduct(t, e, "layette", "Soap", "", 1)
	hidden := shopProduct(t, e, "layette", "Hidden", "https://partner.example/h", 0)

	r := e.do(fiber.MethodGet, "/api/v1/shop/categories", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	cats := r.data()["categories"].([]any)
	require.Len(t, cats, 2, "the two seeded categories")
	first := cats[0].(map[string]any)
	assert.Equal(t, "layette", first["slug"])
	assert.Equal(t, "Layette & baby", first["title"])
	assert.EqualValues(t, 2, first["product_count"], "active products only")

	r = e.do(fiber.MethodGet, "/api/v1/shop/products?category=layette", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	prods := r.data()["products"].([]any)
	require.Len(t, prods, 2)
	p0 := prods[0].(map[string]any)
	assert.Equal(t, "Swaddle", p0["title"])
	assert.Equal(t, "https://partner.example/p/1?ref=a&utm_source=ritme", p0["link_url"])
	assert.Nil(t, prods[1].(map[string]any)["link_url"], "a product without link-out still renders")
	assert.NotContains(t, r.Raw, "Hidden")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/products?category=nope", sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/products/"+itoa(hidden), sara, nil).Status)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/shop/products/"+itoa(plain), sara, nil).Status)

	// Clicks: counted per product and day, no user id anywhere.
	for range 2 {
		r = e.do(fiber.MethodPost, "/api/v1/shop/products/"+itoa(link)+"/click", sara, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		assert.True(t, strings.Contains(r.Raw, "utm_source=ritme"))
	}
	assert.Equal(t, 2, e.count("SELECT clicks FROM shop_click_counts WHERE product_id = ?", link))
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/shop/products/"+itoa(plain)+"/click", sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/shop/products/"+itoa(hidden)+"/click", sara, nil).Status)
	assert.Equal(t, 0, e.count("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name LIKE 'shop\\_%' AND column_name LIKE '%user%'"),
		"shop tables carry no user column")
}

func TestShop_TeenHasNoShop(t *testing.T) {
	e := newCompanionEnv(t)
	nila := e.user("Nila", "09120008002")
	e.teenMode("Nila")
	id := shopProduct(t, e, "layette", "Swaddle", "https://partner.example/p/1", 1)
	r := e.do(fiber.MethodGet, "/api/v1/shop/categories", nila, nil)
	require.Equal(t, fiber.StatusOK, r.Status)
	assert.Empty(t, r.data()["categories"])
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/products", nila, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/shop/products/"+itoa(id)+"/click", nila, nil).Status)
	assert.Zero(t, e.count("SELECT COUNT(*) FROM shop_click_counts"))
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

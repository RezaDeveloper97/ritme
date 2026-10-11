package shop_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/shop"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	shop.NewAdmin(e.DB, nil, "http://localhost", admintest.Quiet).Routes(e.Route(), e.Kit)
	return e
}

func TestShopAdmin_GuardsAndCapability(t *testing.T) {
	e := newEnv(t)
	assert.Equal(t, 401, e.Anonymous().Get("/shop/categories").Status)
	assert.Equal(t, 200, e.As(admintest.EditorID).Get("/shop/categories").Status)
	assert.Equal(t, httpadmin.CapContent, httpadmin.RouteCapability(httpadmin.Prefix+"/shop/products"))

	noToken := e.As(admintest.EditorID)
	noToken.CSRF = ""
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/shop/categories", map[string]any{}).Status)
}

func TestShopAdmin_CategoriesAndProducts(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.Get("/shop/categories")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 2)

	// Validation.
	r = c.JSON(fiber.MethodPost, "/shop/categories", map[string]any{"slug": "Bad Slug"})
	assert.Equal(t, 422, r.Status)
	r = c.JSON(fiber.MethodPost, "/shop/categories", map[string]any{"slug": "layette", "title": map[string]any{"fa": "x", "en": "x"}})
	assert.Equal(t, 422, r.Status, "slug is unique")

	r = c.JSON(fiber.MethodPost, "/shop/categories", map[string]any{"slug": "mom", "title": map[string]any{"fa": "مادر", "en": "Mom"}, "is_active": true})
	require.Equal(t, 201, r.Status, r.Body)
	catID := int(r.Obj("category")["id"].(float64))

	body := map[string]any{"category_id": catID, "title": map[string]any{"fa": "پوشک", "en": "Diaper"},
		"price_text": "120", "badge": "new", "external_url": "https://shop.example/p?x=1", "partner_label": "Shop", "is_active": true}
	for _, bad := range []string{"javascript:alert(1)", "ftp://x.example/a", "not a url"} {
		b := map[string]any{}
		for k, v := range body {
			b[k] = v
		}
		b["external_url"] = bad
		assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/shop/products", b).Status, bad)
	}
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/shop/products", map[string]any{"category_id": 9999, "title": body["title"]}).Status)

	r = c.JSON(fiber.MethodPost, "/shop/products", body)
	require.Equal(t, 201, r.Status, r.Body)
	pid := int(r.Obj("product")["id"].(float64))
	assert.Equal(t, "https://shop.example/p?x=1", r.Obj("product")["external_url"])

	// A product without link-out is valid (nullable external_url).
	r = c.JSON(fiber.MethodPost, "/shop/products", map[string]any{"category_id": catID, "title": body["title"]})
	require.Equal(t, 201, r.Status, r.Body)
	assert.Nil(t, r.Obj("product")["external_url"])

	// Multipart update (the admin-web form) keeps the image untouched and rewrites the fields.
	r = c.Multipart(fiber.MethodPost, "/shop/products/"+itoa(pid), map[string]string{
		"category_id": itoa(catID), "title[fa]": "پوشک نو", "title[en]": "New diaper", "price_text": "130", "is_active": "1",
	})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "130", r.Obj("product")["price_text"])
	assert.Nil(t, r.Obj("product")["external_url"], "a full-replace update clears fields not sent")

	r = c.Get("/shop/products?category_id=" + itoa(catID))
	assert.Len(t, r.Items(), 2)
	r = c.JSON(fiber.MethodPost, "/shop/products/"+itoa(pid)+"/toggle", nil)
	assert.Equal(t, false, r.Obj("product")["is_active"])

	// A category with products cannot be deleted; an empty one can.
	r = c.JSON(fiber.MethodDelete, "/shop/categories/"+itoa(catID), nil)
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, shop.CodeInUse, r.Code())
	assert.Equal(t, 200, c.JSON(fiber.MethodDelete, "/shop/products/"+itoa(pid), nil).Status)
	assert.Equal(t, 404, c.Get("/shop/products/"+itoa(pid)).Status)
	assert.Equal(t, 404, c.Get("/shop/products/abc").Status)
	e.Exec("DELETE FROM shop_products WHERE category_id = ?", catID)
	assert.Equal(t, 200, c.JSON(fiber.MethodDelete, "/shop/categories/"+itoa(catID), nil).Status)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

package cityservices_test

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"log/slog"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/cityservices"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	cityservices.NewAdmin(e.DB, media.NewDisk(e.Storage), "http://localhost", admintest.Quiet).Routes(e.Route(), e.Kit)
	return e
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestCityAdmin_GuardsAndCapabilities(t *testing.T) {
	e := newEnv(t)
	assert.Equal(t, 401, e.Anonymous().Get("/city-services/listings").Status)
	assert.Equal(t, 401, e.Anonymous().Get("/city-service-bookings").Status)
	assert.Equal(t, 200, e.As(admintest.EditorID).Get("/city-services/listings").Status)
	assert.Equal(t, httpadmin.CapContent, httpadmin.RouteCapability(httpadmin.Prefix+"/city-services/listings"))
	assert.Equal(t, httpadmin.CapSupport, httpadmin.RouteCapability(httpadmin.Prefix+"/city-service-bookings"))
	noToken := e.As(admintest.EditorID)
	noToken.CSRF = ""
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/city-services/cities", map[string]any{}).Status)
}

func TestCityAdmin_CrudAndQueue(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	// Cities / categories.
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-services/cities", map[string]any{"slug": "Bad Slug"}).Status)
	r := c.JSON(fiber.MethodPost, "/city-services/cities", map[string]any{"slug": "tehran", "name": map[string]any{"fa": "تهران", "en": "Tehran"}, "is_active": true})
	require.Equal(t, 201, r.Status, r.Body)
	city := int(r.Obj("item")["id"].(float64))
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-services/cities", map[string]any{"slug": "tehran", "name": map[string]any{"fa": "x", "en": "x"}}).Status, "slug unique")
	r = c.Get("/city-services/categories")
	require.Len(t, r.Items(), 3, "class / pool / playhouse seeded")
	cat := int(r.Items()[0].(map[string]any)["id"].(float64))

	// Listings.
	base := map[string]any{"name": "Blue Pool", "category_id": cat, "city_id": city, "booking_mode": "request", "is_active": true}
	with := func(k string, v any) map[string]any {
		m := map[string]any{}
		for a, b := range base {
			m[a] = b
		}
		m[k] = v
		return m
	}
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-services/listings", with("booking_mode", "teleport")).Status)
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-services/listings", with("booking_mode", "link")).Status, "link needs a url")
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-services/listings", with("booking_mode", "phone")).Status, "phone needs a phone")
	for _, bad := range []string{"javascript:alert(1)", "ftp://x.example", "not a url"} {
		m := with("booking_mode", "link")
		m["booking_url"] = bad
		assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-services/listings", m).Status, bad)
	}
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-services/listings", with("city_id", 9999)).Status)
	r = c.JSON(fiber.MethodPost, "/city-services/listings", base)
	require.Equal(t, 201, r.Status, r.Body)
	lid := int(r.Obj("listing")["id"].(float64))
	assert.Equal(t, 1, len(c.Get("/city-services/listings?city_id="+itoa(city)).Items()))
	r = c.JSON(fiber.MethodPost, "/city-services/listings/"+itoa(lid)+"/toggle", nil)
	assert.Equal(t, false, r.Obj("listing")["is_active"])
	assert.Equal(t, 404, c.Get("/city-services/listings/999999").Status)
	assert.Equal(t, 404, c.JSON(fiber.MethodDelete, "/city-services/listings/999999/photos/0", nil).Status)

	// A city with listings cannot be deleted.
	r = c.JSON(fiber.MethodDelete, "/city-services/cities/"+itoa(city), nil)
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, cityservices.CodeInUse, r.Code())

	// Booking queue: status machine.
	ures := e.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120001111', NOW(), NOW())`)
	uid, _ := ures.LastInsertId()
	res := e.Exec(`INSERT INTO city_service_bookings (user_id, listing_id, contact_name, contact_phone, preferred_date, status, created_at, updated_at)
		VALUES (?, ?, 'Sara', '09120000000', '2030-01-01', 'requested', NOW(), NOW())`, uid, lid)
	bid, _ := res.LastInsertId()
	b := itoa(int(bid))
	r = c.Get("/city-service-bookings?status=requested")
	require.Len(t, r.Items(), 1)
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, "/city-service-bookings/"+b, map[string]any{"status": "completed"}).Status, "requested cannot jump to completed")
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, "/city-service-bookings/"+b, map[string]any{"status": "nope"}).Status)
	r = c.JSON(fiber.MethodPut, "/city-service-bookings/"+b, map[string]any{"status": "confirmed", "status_note": "Friday 10:00"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "confirmed", r.Obj("booking")["status"])
	assert.Equal(t, 200, c.JSON(fiber.MethodPut, "/city-service-bookings/"+b, map[string]any{"status": "completed"}).Status)
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, "/city-service-bookings/"+b, map[string]any{"status": "cancelled"}).Status, "completed is final")

	// A listing with requests cannot be deleted (deactivate instead).
	r = c.JSON(fiber.MethodDelete, "/city-services/listings/"+itoa(lid), nil)
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, cityservices.CodeInUse, r.Code())
}

func TestNotifier_LogsIdsOnly(t *testing.T) {
	var buf bytes.Buffer
	n := cityservices.NewNotifier("fake", slog.New(slog.NewTextHandler(&buf, nil)))
	require.NoError(t, n.BookingRequested(context.Background(), cityservices.BookingNotice{
		BookingID: 7, ListingID: 3, ContactName: "Sara Secret", ContactPhone: "09129999999", Note: "private note",
		PreferredDate: civildate.MustParse("2030-01-01"), ChildAgeRange: "1-3"}))
	out := buf.String()
	assert.Contains(t, out, "booking_id=7")
	for _, leak := range []string{"Sara", "0912", "private", "1-3"} {
		assert.NotContains(t, out, leak)
	}
}

func TestStatusMachine(t *testing.T) {
	assert.True(t, cityservices.CanMove("requested", "confirmed"))
	assert.True(t, cityservices.CanMove("confirmed", "completed"))
	assert.False(t, cityservices.CanMove("requested", "completed"))
	assert.False(t, cityservices.CanMove("completed", "cancelled"))
	assert.False(t, cityservices.CanMove("cancelled", "requested"))
}

func TestCityAdmin_Photos(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	cr := c.JSON(fiber.MethodPost, "/city-services/cities", map[string]any{"slug": "c", "name": map[string]any{"fa": "ش", "en": "C"}, "is_active": true})
	require.Equal(t, 201, cr.Status, cr.Body)
	r := c.JSON(fiber.MethodPost, "/city-services/listings", map[string]any{"name": "P", "booking_mode": "request", "is_active": true,
		"city_id": cr.Obj("item")["id"], "category_id": c.Get("/city-services/categories").Items()[0].(map[string]any)["id"]})
	require.Equal(t, 201, r.Status, r.Body)
	lid := itoa(int(r.Obj("listing")["id"].(float64)))

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 300, 300))))
	r = c.Multipart(fiber.MethodPost, "/city-services/listings/"+lid+"/photos", nil,
		admintest.File{Field: "photo", Name: "a.png", ContentType: "image/png", Data: buf.Bytes()})
	require.Equal(t, 201, r.Status, r.Body)
	assert.Len(t, r.Obj("listing")["photos"], 1)
	// Not an image → 422.
	r = c.Multipart(fiber.MethodPost, "/city-services/listings/"+lid+"/photos", nil,
		admintest.File{Field: "photo", Name: "a.png", ContentType: "image/png", Data: []byte("<svg/>")})
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, 422, c.Multipart(fiber.MethodPost, "/city-services/listings/"+lid+"/photos", nil).Status, "photo required")
	r = c.JSON(fiber.MethodDelete, "/city-services/listings/"+lid+"/photos/0", nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Empty(t, r.Obj("listing")["photos"])
}

package http

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedCity returns (cityID, listingIDs) with a request-mode, a link-mode and a hidden listing.
func seedCityServices(t *testing.T, e *companionEnv) (city, request, link, hidden uint64) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO city_service_cities (slug, name, sort_order, is_active, created_at, updated_at)
		VALUES ('tehran', '{"fa":"تهران","en":"Tehran"}', 1, 1, NOW(), NOW())`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	city = uint64(id) //nolint:gosec // test ids
	add := func(name, mode, url string, active int) uint64 {
		var u any
		if url != "" {
			u = url
		}
		r, err := e.db.Exec(`INSERT INTO city_service_listings (category_id, city_id, name, district, phone, price_text, booking_mode,
			booking_url, sort_order, is_active, created_at, updated_at)
			VALUES ((SELECT id FROM city_service_categories WHERE slug = 'pool'), ?, ?, 'Valiasr', '02112345678', '300', ?, ?, 1, ?, NOW(), NOW())`,
			city, name, mode, u, active)
		require.NoError(t, err)
		n, _ := r.LastInsertId()
		return uint64(n) //nolint:gosec // test ids
	}
	return city, add("Blue Pool", "request", "", 1), add("Link Pool", "link", "https://venue.example/book", 1), add("Hidden Pool", "request", "", 0)
}

func TestCityServices_PublicAndBookings(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009001")
	omid := e.user("Omid", "09120009002")

	// CB-DIR-02: the reads are public; only the bookings need a session.
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/city-services/options", "", nil).Status)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/city-services/listings", "", nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/city-services/listings/1", "", nil).Status)
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, "/api/v1/city-services/bookings", "", nil).Status)
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, "/api/v1/city-services/bookings", "", nil).Status)
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, "/api/v1/city-services/bookings/1/cancel", "", nil).Status)

	city, request, link, hidden := seedCityServices(t, e)

	r := e.do(fiber.MethodGet, "/api/v1/city-services/options", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["cities"], 1)
	assert.Len(t, r.data()["categories"], 1, "CB-DIR-02: only categories with a visible listing (pool)")

	r = e.do(fiber.MethodGet, "/api/v1/city-services/listings?city="+itoa(city)+"&category=pool", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["listings"], 2, "the hidden listing never shows")
	assert.NotContains(t, r.Raw, "Hidden Pool")
	r = e.do(fiber.MethodGet, "/api/v1/city-services/listings?category=class", sara, nil)
	assert.Empty(t, r.data()["listings"])
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodGet, "/api/v1/city-services/listings?city=abc", sara, nil).Status)

	r = e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(link), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "https://venue.example/book", r.data()["listing"].(map[string]any)["booking_url"])
	r = e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(request), sara, nil)
	assert.Nil(t, r.data()["listing"].(map[string]any)["booking_url"])
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(hidden), sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/city-services/listings/abc", sara, nil).Status)

	day := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	body := func(listing uint64) map[string]any {
		return map[string]any{"listing_id": listing, "contact_name": "Sara", "contact_phone": "۰۹۱۲ ۱۲۳-۴۵۶۷",
			"preferred_date": day, "child_age_range": "1-3", "note": "morning please"}
	}
	// Validation.
	bad := body(request)
	bad["contact_phone"] = "abc"
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/bookings", sara, bad).Status)
	bad = body(request)
	bad["preferred_date"] = "2020-01-01"
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/bookings", sara, bad).Status, "past day")
	bad = body(request)
	bad["child_age_range"] = "has asthma"
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/bookings", sara, bad).Status, "free-text child detail")
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/bookings", sara, map[string]any{}).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/bookings", sara, body(link)).Status, "link-mode takes no request")
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/bookings", sara, body(hidden)).Status)

	// Create (phone normalised, extra health-ish keys ignored).
	ok := body(request)
	ok["diagnosis"] = "x"
	ok["cycle_day"] = 12
	r = e.do(fiber.MethodPost, "/api/v1/city-services/bookings", sara, ok)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	b := r.data()["booking"].(map[string]any)
	assert.Equal(t, "requested", b["status"])
	assert.Equal(t, "09121234567", b["contact_phone"])
	bid := itoa(uint64(b["id"].(float64)))
	assert.NotContains(t, r.Raw, "diagnosis")
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/bookings", sara, body(request)).Status, "one open request per venue")

	// IDOR: Omid sees nothing of Sara's.
	r = e.do(fiber.MethodGet, "/api/v1/city-services/bookings", omid, nil)
	assert.Empty(t, r.data()["bookings"])
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/city-services/bookings/"+bid+"/cancel", omid, nil).Status)
	assert.Equal(t, 1, e.count("SELECT COUNT(*) FROM city_service_bookings WHERE status = 'requested'"))
	r = e.do(fiber.MethodGet, "/api/v1/city-services/bookings", sara, nil)
	assert.Len(t, r.data()["bookings"], 1)

	// Cancel; a second cancel is refused.
	r = e.do(fiber.MethodPost, "/api/v1/city-services/bookings/"+bid+"/cancel", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "cancelled", r.data()["booking"].(map[string]any)["status"])
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/bookings/"+bid+"/cancel", sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/city-services/bookings/abc/cancel", sara, nil).Status)

	// Privacy: no health column, no account id toward the venue payload.
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'city_service_bookings'
		AND (column_name LIKE '%health%' OR column_name LIKE '%diagnos%' OR column_name LIKE '%cycle%' OR column_name LIKE '%pregnan%')`))
}

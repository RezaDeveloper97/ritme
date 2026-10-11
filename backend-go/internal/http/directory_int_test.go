package http

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/clock"
)

// CB-DIR-02: the public directory search, place page and slots on bloom's /city-services routes.
func TestDirectory_PublicSearchPlaceSlots(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara Ahmadi", "09120009101")
	owner := e.user("Owner", "09120009102")
	_ = owner

	city := e.insExec(`INSERT INTO city_service_cities (slug, name, sort_order, is_active, created_at, updated_at)
		VALUES ('tehran', '{"fa":"تهران","en":"Tehran"}', 1, 1, NOW(), NOW())`)
	hours := `[{"day":4,"open":"09:00","close":"20:00"}]` // 2026-09-23 is a Wednesday (day 4)
	place := func(name, cat, status string, lat, lng any, ages, amen, wh string, rating any, count int) uint64 {
		var a, m, w any
		if ages != "" {
			a = ages
		}
		if amen != "" {
			m = amen
		}
		if wh != "" {
			w = wh
		}
		return e.insExec(`INSERT INTO city_service_listings (category_id, city_id, name, district, booking_mode, sort_order, is_active,
			status, owner_user_id, lat, lng, age_ranges, amenities, weekly_hours, rating_avg, rating_count, about, website_url, created_at, updated_at)
			VALUES ((SELECT id FROM city_service_categories WHERE slug = ?), ?, ?, 'Vanak', 'online', 1, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			'About text', 'javascript:alert(1)', NOW(), NOW())`,
			cat, city, name, status, e.ids["Owner"], lat, lng, a, m, w, rating, count)
	}
	pool := place("Abpari", "pool", "published", 35.7576, 51.4099, `["y1_2"]`, `["nursing_room","female_instructor"]`, hours, 4.8, 10)
	class := place("Ghoncheh", "class", "published", 35.772, 51.418, `["m0_6"]`, `["female_instructor"]`, "", nil, 0)
	far := place("Far pool", "pool", "published", 36.3, 59.6, "", "", "", 5.0, 300)
	draft := place("Draft", "pool", "draft", 35.7577, 51.41, `["y1_2"]`, "", hours, nil, 0)
	suspended := place("Suspended", "pool", "suspended", 35.7577, 51.41, "", "", "", nil, 0)

	svc := e.insExec(`INSERT INTO city_service_services (listing_id, kind, name, duration_minutes, age_min_months, age_max_months,
		capacity, price_toman, sort_order, is_active, created_at, updated_at) VALUES (?, 'session', 'Water intro', 45, 6, 18, 6, 320000, 1, 1, NOW(), NOW())`, pool)
	off := e.insExec(`INSERT INTO city_service_services (listing_id, kind, name, price_toman, sort_order, is_active, created_at, updated_at)
		VALUES (?, 'session', 'Retired', 1000, 2, 0, NOW(), NOW())`, pool)
	slot := func(listing uint64, service any, at string, capacity, booked int, status string) uint64 {
		return e.insExec(`INSERT INTO city_service_slots (listing_id, service_id, starts_at, capacity, booked_count, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`, listing, service, at, capacity, booked, status)
	}
	next := slot(pool, svc, "2026-09-23 17:00:00", 6, 2, "open")
	full := slot(pool, svc, "2026-09-24 10:30:00", 6, 6, "open")
	slot(pool, svc, "2026-09-22 17:00:00", 6, 0, "open")   // past
	slot(pool, svc, "2026-09-25 17:00:00", 6, 0, "closed") // closed
	slot(pool, off, "2026-09-25 18:00:00", 6, 0, "open")   // inactive service
	slot(draft, nil, "2026-09-24 09:00:00", 6, 0, "open")

	// A completed booking with a published review, and a hidden one.
	bk := func(status string) uint64 {
		return e.insExec(`INSERT INTO city_service_bookings (user_id, listing_id, contact_name, contact_phone, preferred_date, status, created_at, updated_at)
			VALUES (?, ?, 'Sara', '09121234567', '2026-09-01', ?, NOW(), NOW())`, e.ids["Sara Ahmadi"], pool, status)
	}
	e.insExec(`INSERT INTO city_service_reviews (listing_id, booking_id, user_id, rating, score_cleanliness, body, status, created_at, updated_at)
		VALUES (?, ?, ?, 5, 4, 'Warm water', 'published', NOW(), NOW())`, pool, bk("completed"), e.ids["Sara Ahmadi"])
	e.insExec(`INSERT INTO city_service_reviews (listing_id, booking_id, user_id, rating, body, status, created_at, updated_at)
		VALUES (?, ?, ?, 1, 'HIDDEN-REVIEW', 'hidden', NOW(), NOW())`, pool, bk("completed"), e.ids["Sara Ahmadi"])
	// A public place photo, and a private file id that must never resolve.
	photo := e.insExec(`INSERT INTO files (user_id, purpose, visibility, mime, size_bytes, sha256, path, created_at, updated_at)
		VALUES (NULL, 'place_photo', 'public', 'image/webp', 10, REPEAT('a', 64), 'place_photo/0/p1.webp.bin', NOW(), NOW())`)
	secret := e.insExec(`INSERT INTO files (user_id, purpose, visibility, mime, size_bytes, sha256, path, created_at, updated_at)
		VALUES (?, 'place_licence', 'private', 'application/pdf', 10, REPEAT('b', 64), 'place_licence/1/licence.pdf.bin', NOW(), NOW())`, e.ids["Owner"])
	_, err := e.db.Exec(`UPDATE city_service_listings SET photo_file_ids = JSON_ARRAY(?, ?) WHERE id = ?`, secret, photo, pool)
	require.NoError(t, err)

	names := func(r reply) []string {
		var out []string
		for _, x := range r.data()["listings"].([]any) {
			out = append(out, x.(map[string]any)["name"].(string))
		}
		return out
	}
	list := func(q string) reply {
		r := e.do(fiber.MethodGet, "/api/v1/city-services/listings"+q, "", nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		return r
	}

	// Anonymous browse: only published places; draft / suspended never show anywhere.
	r := list("")
	assert.ElementsMatch(t, []string{"Abpari", "Ghoncheh", "Far pool"}, names(r))
	assert.NotContains(t, r.Raw, "Draft")
	assert.NotContains(t, r.Raw, "Suspended")
	assert.NotContains(t, r.Raw, "owner")
	assert.Equal(t, "rank", r.data()["sort"])

	// Distance + radius + sort.
	r = list("?lat=35.7576&lng=51.4099&radius_km=10&sort=distance")
	assert.Equal(t, []string{"Abpari", "Ghoncheh"}, names(r))
	first := r.data()["listings"].([]any)[0].(map[string]any)
	assert.InDelta(t, 0, first["distance_km"], 0.01)
	assert.Equal(t, true, first["open_now"])
	assert.Equal(t, false, first["verified"])
	assert.InDelta(t, 320000, first["price_from_toman"], 0)
	assert.Equal(t, "2026-09-23T17:00:00+03:30", first["next_slot_at"])
	assert.Equal(t, "http://localhost/api/v1/files/public/"+itoa(photo)+"/p1.webp", first["photo_url"], "private file ids never resolve")

	// Near beats a better-rated far place in the default rank once a position is given.
	r = list("?lat=35.7576&lng=51.4099")
	assert.Equal(t, "Abpari", names(r)[0])
	r = list("?sort=rating")
	assert.Equal(t, "Far pool", names(r)[0])

	// Age: 14 months fits Abpari (y1_2), not Ghoncheh (m0_6); Far pool has no age data and stays.
	r = list("?age_months=14")
	assert.ElementsMatch(t, []string{"Abpari", "Far pool"}, names(r))
	// Ghoncheh fits by a service even though its range does not.
	e.insExec(`INSERT INTO city_service_services (listing_id, kind, name, age_min_months, age_max_months, sort_order, is_active, created_at, updated_at)
		VALUES (?, 'session', 'Toddlers', 12, 36, 1, 1, NOW(), NOW())`, class)
	r = list("?age_months=14")
	assert.ElementsMatch(t, []string{"Abpari", "Ghoncheh", "Far pool"}, names(r))

	// Open now, amenities, q, bbox.
	assert.Equal(t, []string{"Abpari"}, names(list("?open_now=1")))
	assert.ElementsMatch(t, []string{"Abpari"}, names(list("?amenities=nursing_room,female_instructor")))
	assert.ElementsMatch(t, []string{"Ghoncheh"}, names(list("?q=ghon")))
	assert.Empty(t, names(list("?q=%25")), "LIKE wildcards are escaped")
	assert.ElementsMatch(t, []string{"Abpari", "Ghoncheh"}, names(list("?bbox=51.40,35.75,51.42,35.78")))
	assert.Equal(t, []string{"Abpari"}, names(list("?category=pool&city="+itoa(city)+"&bbox=51.40,35.75,51.42,35.78")))

	for _, bad := range []string{"?age_months=-1", "?age_months=999", "?open_now=maybe", "?amenities=sauna", "?lat=35.7",
		"?lat=95&lng=51", "?radius_km=5", "?lat=35.7&lng=51.4&radius_km=500", "?bbox=1,2,3", "?sort=paid", "?q=" + longQ()} {
		assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodGet, "/api/v1/city-services/listings"+bad, "", nil).Status, bad)
	}

	// Options: categories with a published listing only (pool, class), amenities + age ranges from the catalog.
	r = e.do(fiber.MethodGet, "/api/v1/city-services/options", "", nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["categories"], 2)
	assert.Len(t, r.data()["amenities"], 7)
	assert.Len(t, r.data()["place_age_ranges"], 6)

	// The place page.
	r = e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(pool)+"?lat=35.77&lng=51.42", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	l := r.data()["listing"].(map[string]any)
	assert.Len(t, l["services"], 1, "inactive services hidden")
	slots := l["next_slots"].([]any)
	require.Len(t, slots, 2, "past, closed and inactive-service slots hidden")
	assert.Equal(t, true, slots[1].(map[string]any)["is_full"])
	assert.Nil(t, l["website_url"], "only http(s) links")
	assert.NotNil(t, l["distance_km"])
	rev := l["reviews"].(map[string]any)
	assert.InDelta(t, 1, rev["rating_count"], 0)
	assert.InDelta(t, 5, rev["rating_avg"], 0)
	latest := rev["latest"].([]any)
	require.Len(t, latest, 1)
	assert.Equal(t, "Sara", latest[0].(map[string]any)["author_first_name"], "first name only")
	assert.NotContains(t, r.Raw, "HIDDEN-REVIEW")
	assert.NotContains(t, r.Raw, "Ahmadi")
	assert.NotContains(t, r.Raw, "09121234567")
	assert.NotContains(t, r.Raw, "licence")
	assert.NotContains(t, r.Raw, "owner")
	for _, id := range []uint64{draft, suspended} {
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(id), "", nil).Status)
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(id)+"/slots", "", nil).Status)
	}
	_ = far

	// Slots.
	r = e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(pool)+"/slots", "", nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	got := r.data()["slots"].([]any)
	require.Len(t, got, 2)
	assert.InDelta(t, next, got[0].(map[string]any)["id"], 0)
	assert.InDelta(t, full, got[1].(map[string]any)["id"], 0)
	r = e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(pool)+"/slots?from=2026-09-24&days=1&service_id="+itoa(svc), "", nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["slots"], 1)
	for _, bad := range []string{"?service_id=" + itoa(off), "?days=0", "?days=99", "?from=nope", "?service_id=abc"} {
		assert.Equal(t, fiber.StatusUnprocessableEntity,
			e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(pool)+"/slots"+bad, "", nil).Status, bad)
	}

	// Cache-Control: public for anonymous reads, private with a session; the throttle headers are present.
	for token, want := range map[string]string{"": "public, max-age=60", sara: "private, max-age=60"} {
		req := httptest.NewRequest(fiber.MethodGet, "/api/v1/city-services/listings", nil)
		req.Header.Set(clock.Header, companionNow)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, want, resp.Header.Get("Cache-Control"))
		assert.Equal(t, "120", resp.Header.Get("X-RateLimit-Limit"))
	}
}

func longQ() string {
	s := ""
	for range 81 {
		s += "a"
	}
	return s
}

package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cityservices"
	"github.com/ritme/backend-go/internal/platform/clock"
)

var bookingCodeRe = regexp.MustCompile(`^[2-9A-HJ-NP-Z]{8}$`)

// dirBookingFixture is an online pool (cancel free until 24 h before) with two age-bounded services and slots, plus a
// request-mode class and a draft place. The clock is companionNow = 2026-09-23 10:00 Tehran.
type dirBookingFixture struct {
	pool, class, draft           uint64
	baby, toddler                uint64 // services: 6–18 and 18–48 months
	s1, s2, sToddler, full, past uint64
	small                        uint64 // capacity 2 (concurrency)
	soon                         uint64 // today 20:00, inside the 24 h window
	saraChild, omidChild, oldKid uint64
	sara, omid                   string
	saraUser, omidUser           uint64
	bookURL, bookingsURL         string
}

func seedDirBookings(t *testing.T, e *companionEnv) dirBookingFixture {
	t.Helper()
	f := dirBookingFixture{bookURL: "/api/v1/city-services/bookings", bookingsURL: "/api/v1/city-services/bookings"}
	f.sara = e.user("Sara Ahmadi", "09120009201")
	f.omid = e.user("Omid", "09120009202")
	f.saraUser, f.omidUser = e.ids["Sara Ahmadi"], e.ids["Omid"]
	city := e.insExec(`INSERT INTO city_service_cities (slug, name, sort_order, is_active, created_at, updated_at)
		VALUES ('tehran', '{"fa":"تهران","en":"Tehran"}', 1, 1, NOW(), NOW())`)
	place := func(name, cat, mode, status string) uint64 {
		return e.insExec(`INSERT INTO city_service_listings (category_id, city_id, name, district, booking_mode, sort_order, is_active,
			status, lat, lng, cancel_free_hours, cancel_policy, created_at, updated_at)
			VALUES ((SELECT id FROM city_service_categories WHERE slug = ?), ?, ?, 'Vanak', ?, 1, 1, ?, 35.7576, 51.4099, 24,
			'Free until 24 hours before', NOW(), NOW())`, cat, city, name, mode, status)
	}
	f.pool = place("Abpari", "pool", "online", "published")
	f.class = place("Ghoncheh", "class", "request", "published")
	f.draft = place("Draft", "pool", "online", "draft")
	svc := func(name string, minM, maxM int) uint64 {
		return e.insExec(`INSERT INTO city_service_services (listing_id, kind, name, duration_minutes, age_min_months, age_max_months,
			capacity, price_toman, sort_order, is_active, created_at, updated_at) VALUES (?, 'session', ?, 45, ?, ?, 6, 320000, 1, 1, NOW(), NOW())`,
			f.pool, name, minM, maxM)
	}
	f.baby, f.toddler = svc("Water intro", 6, 18), svc("Swim", 18, 48)
	slot := func(listing, service uint64, at string, capacity, booked int) uint64 {
		return e.insExec(`INSERT INTO city_service_slots (listing_id, service_id, starts_at, ends_at, capacity, booked_count, status, created_at, updated_at)
			VALUES (?, ?, ?, DATE_ADD(?, INTERVAL 45 MINUTE), ?, ?, 'open', NOW(), NOW())`, listing, service, at, at, capacity, booked)
	}
	f.s1 = slot(f.pool, f.baby, "2026-09-24 17:00:00", 6, 0)
	f.s2 = slot(f.pool, f.baby, "2026-09-26 17:00:00", 6, 0)
	f.sToddler = slot(f.pool, f.toddler, "2026-09-24 18:00:00", 6, 0)
	f.full = slot(f.pool, f.baby, "2026-09-25 10:30:00", 6, 6)
	f.past = slot(f.pool, f.baby, "2026-09-22 17:00:00", 6, 0)
	f.small = slot(f.pool, f.baby, "2026-09-27 17:00:00", 2, 0)
	f.soon = slot(f.pool, f.baby, "2026-09-23 20:00:00", 6, 0)
	child := func(owner uint64, name, birth string) uint64 {
		return e.insExec(`INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`, owner, name, birth)
	}
	f.saraChild = child(f.saraUser, "Arian", "2025-07-20") // 14 months on 2026-09-24
	f.oldKid = child(f.saraUser, "Nika", "2022-01-01")     // 4 years: too old for Water intro
	f.omidChild = child(f.omidUser, "Kian", "2025-07-01")  // not Sara's
	return f
}

// rawDo is a goroutine-safe request (no require inside): the status, or -1 on a transport error.
func (e *companionEnv) rawDo(method, path, token string, body any) int {
	raw, err := json.Marshal(body)
	if err != nil {
		return -1
	}
	req := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, companionNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		return -1
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func (e *companionEnv) seats(slot uint64) int {
	return e.count(`SELECT booked_count FROM city_service_slots WHERE id = ?`, slot)
}

// CB-DIR-03: slot booking, child ownership + age fit, upcoming / past, reschedule, cancel window, review, report, IDOR.
func TestDirectoryBookings_Flow(t *testing.T) {
	e := newCompanionEnv(t)
	f := seedDirBookings(t, e)
	field := func(r reply) map[string]any { m, _ := r.Body["errors"].(map[string]any); return m }

	for _, path := range []string{f.bookingsURL + "/1", f.bookingsURL + "/1/reschedule", f.bookingsURL + "/1/review",
		"/api/v1/city-services/listings/" + itoa(f.pool) + "/report"} {
		method := fiber.MethodPost
		if path == f.bookingsURL+"/1" {
			method = fiber.MethodGet
		}
		assert.Equal(t, fiber.StatusUnauthorized, e.do(method, path, "", nil).Status, path)
	}

	book := func(token string, body map[string]any) reply { return e.do(fiber.MethodPost, f.bookURL, token, body) }
	base := func(slot uint64) map[string]any {
		return map[string]any{"listing_id": f.pool, "slot_id": slot, "child_id": f.saraChild, "note": "first swim"}
	}

	// Refusals.
	r := book(f.sara, map[string]any{"listing_id": f.pool, "preferred_date": "2026-09-24"})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, field(r), "slot_id", "an online place needs a slot")
	for name, tc := range map[string]struct {
		body  map[string]any
		field string
	}{
		"foreign child":    {map[string]any{"listing_id": f.pool, "slot_id": f.s1, "child_id": f.omidChild}, "child_id"},
		"missing child":    {map[string]any{"listing_id": f.pool, "slot_id": f.s1, "child_id": 999999}, "child_id"},
		"age mismatch":     {map[string]any{"listing_id": f.pool, "slot_id": f.s1, "child_id": f.oldKid}, "child_id"},
		"service mismatch": {map[string]any{"listing_id": f.pool, "slot_id": f.sToddler, "service_id": f.baby}, "service_id"},
		"full slot":        {base(f.full), "slot_id"},
		"past slot":        {base(f.past), "slot_id"},
		"other place slot": {map[string]any{"listing_id": f.class, "slot_id": f.s1, "preferred_date": "2026-09-24"}, "slot_id"},
		"draft place":      {map[string]any{"listing_id": f.draft, "slot_id": f.s1}, "listing_id"},
		"bad slot id":      {map[string]any{"listing_id": f.pool, "slot_id": "abc"}, "slot_id"},
	} {
		r := book(f.sara, tc.body)
		require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, name+": "+r.Raw)
		assert.Contains(t, field(r), tc.field, name)
	}
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM city_service_bookings`))

	// Create: contact defaults to the account, the slot fixes the day, the service price is snapshotted.
	r = book(f.sara, base(f.s1))
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	b := r.data()["booking"].(map[string]any)
	assert.Equal(t, "requested", b["status"], "never auto-confirmed")
	assert.Regexp(t, bookingCodeRe, b["code"])
	assert.Equal(t, "2026-09-24", b["preferred_date"])
	assert.Equal(t, "Sara Ahmadi", b["contact_name"])
	assert.Equal(t, "09120009201", b["contact_phone"])
	assert.InDelta(t, 320000, b["price_toman"], 0)
	assert.Equal(t, map[string]any{"id": float64(f.saraChild), "name": "Arian", "age_months": float64(14)}, b["child"])
	assert.Equal(t, "2026-09-24T17:00:00+03:30", b["slot"].(map[string]any)["starts_at"])
	assert.Equal(t, "2026-09-23T17:00:00+03:30", b["cancel_free_until"])
	assert.Equal(t, true, b["is_upcoming"])
	assert.Equal(t, true, b["can_cancel"])
	assert.Equal(t, true, b["can_reschedule"])
	assert.Equal(t, false, b["can_review"])
	assert.NotContains(t, r.Raw, "2025-07-20", "no birth date leaves the API")
	bid := itoa(uint64(b["id"].(float64)))
	assert.Equal(t, 1, e.seats(f.s1))
	r = book(f.sara, base(f.s1))
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, field(r), "slot_id", "one seat per mother and slot")
	assert.Equal(t, 1, e.seats(f.s1))

	// IDOR: Omid sees and touches nothing of Sara's.
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, f.bookingsURL+"/"+bid, f.omid, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/cancel", f.omid, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/reschedule", f.omid, map[string]any{"slot_id": f.s2}).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/review", f.omid, map[string]any{"rating": 5}).Status)
	assert.Empty(t, e.do(fiber.MethodGet, f.bookingsURL, f.omid, nil).data()["bookings"])
	assert.Equal(t, 1, e.seats(f.s1))
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, f.bookingsURL+"/abc", f.sara, nil).Status)

	// One booking; upcoming / past + counts.
	r = e.do(fiber.MethodGet, f.bookingsURL+"/"+bid, f.sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, b["code"], r.data()["booking"].(map[string]any)["code"])
	r = e.do(fiber.MethodGet, f.bookingsURL+"?scope=upcoming", f.sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["bookings"], 1)
	assert.Equal(t, map[string]any{"upcoming": float64(1), "past": float64(0)}, r.data()["counts"])
	assert.Empty(t, e.do(fiber.MethodGet, f.bookingsURL+"?scope=past", f.sara, nil).data()["bookings"])
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodGet, f.bookingsURL+"?scope=all", f.sara, nil).Status)

	// Reschedule: the new seat is taken, the old one given back, back to requested.
	_, err := e.db.Exec(`UPDATE city_service_bookings SET status = 'confirmed', confirmed_at = NOW() WHERE id = ?`, bid)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/reschedule", f.sara, map[string]any{"slot_id": f.full}).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/reschedule", f.sara, map[string]any{"slot_id": f.sToddler}).Status, "another service")
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/reschedule", f.sara, map[string]any{}).Status)
	r = e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/reschedule", f.sara, map[string]any{"slot_id": f.s2})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	b = r.data()["booking"].(map[string]any)
	assert.Equal(t, "requested", b["status"])
	assert.Equal(t, "2026-09-26", b["preferred_date"])
	assert.Nil(t, b["confirmed_at"])
	assert.Equal(t, 0, e.seats(f.s1))
	assert.Equal(t, 1, e.seats(f.s2))

	// Cancel window: confirmed → only until 24 h before (2026-09-25 17:00); a requested one until the start.
	_, err = e.db.Exec(`UPDATE city_service_bookings SET status = 'confirmed' WHERE id = ?`, bid)
	require.NoError(t, err)
	r = e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/cancel", f.sara, nil, "2026-09-25T18:00:00+03:30")
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Raw, "free cancellation window")
	assert.Equal(t, false, e.do(fiber.MethodGet, f.bookingsURL+"/"+bid, f.sara, nil, "2026-09-25T18:00:00+03:30").data()["booking"].(map[string]any)["can_cancel"])
	r = e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/cancel", f.sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	b = r.data()["booking"].(map[string]any)
	assert.Equal(t, "cancelled", b["status"])
	assert.Equal(t, "user", b["cancelled_by"])
	assert.Equal(t, 0, e.seats(f.s2), "the seat is back")
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/cancel", f.sara, nil).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/reschedule", f.sara, map[string]any{"slot_id": f.s1}).Status)
	r = e.do(fiber.MethodGet, f.bookingsURL+"?scope=past", f.sara, nil)
	assert.Len(t, r.data()["bookings"], 1)
	r = book(f.sara, base(f.soon)) // 10 h ahead, still requested → cancellable until the start
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	soon := itoa(uint64(r.data()["booking"].(map[string]any)["id"].(float64)))
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, f.bookingsURL+"/"+soon+"/cancel", f.sara, nil, "2026-09-23T19:00:00+03:30").Status)

	// Review: only after completed, one per booking, pending (not public) until moderated.
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/review", f.sara, map[string]any{"rating": 5}).Status)
	done := e.insExec(`INSERT INTO city_service_bookings (user_id, listing_id, contact_name, contact_phone, preferred_date, status,
		code, service_id, completed_at, created_at, updated_at) VALUES (?, ?, 'Sara', '09120009201', '2026-09-10', 'completed',
		'DONE2345', ?, NOW(), NOW(), NOW())`, f.saraUser, f.pool, f.baby)
	rv := f.bookingsURL + "/" + itoa(done) + "/review"
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, rv, f.sara, map[string]any{"rating": 6}).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, rv, f.sara, map[string]any{"rating": 4, "score_instructor": 0}).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, rv, f.omid, map[string]any{"rating": 1}).Status)
	r = e.do(fiber.MethodPost, rv, f.sara, map[string]any{"rating": 5, "score_cleanliness": 4, "body": "Warm water", "user_id": f.omidUser})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, "pending", r.data()["review"].(map[string]any)["status"])
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM city_service_reviews WHERE user_id = ? AND status = 'pending'`, f.saraUser))
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, rv, f.sara, map[string]any{"rating": 4}).Status, "one per booking")
	r = e.do(fiber.MethodGet, "/api/v1/city-services/listings/"+itoa(f.pool), "", nil)
	assert.NotContains(t, r.Raw, "Warm water", "pending reviews are not public")
	r = e.do(fiber.MethodGet, f.bookingsURL+"/"+itoa(done), f.sara, nil)
	b = r.data()["booking"].(map[string]any)
	assert.Equal(t, false, b["can_review"])
	assert.Equal(t, "pending", b["review"].(map[string]any)["status"])

	// Report a place.
	rp := "/api/v1/city-services/listings/" + itoa(f.pool) + "/report"
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, rp, f.sara, map[string]any{"reason": "spam"}).Status)
	r = e.do(fiber.MethodPost, rp, f.sara, map[string]any{"reason": "closed", "note": "Closed since summer"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, rp, f.sara, map[string]any{"reason": "other"}).Status, "one open report")
	assert.Equal(t, fiber.StatusCreated, e.do(fiber.MethodPost, rp, f.omid, map[string]any{"reason": "wrong_location"}).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/city-services/listings/"+itoa(f.draft)+"/report", f.sara, map[string]any{"reason": "closed"}).Status)

	// Admin reject gives the seat back and stamps the history.
	r = book(f.omid, map[string]any{"listing_id": f.pool, "slot_id": f.s1, "child_id": f.omidChild})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	oid := uint64(r.data()["booking"].(map[string]any)["id"].(float64))
	assert.Equal(t, 1, e.seats(f.s1))
	ok, err := cityservices.NewStore(e.db).MoveBooking(context.Background(), oid, "requested", "rejected", sql.NullString{}, time.Now())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, 0, e.seats(f.s1))

	// The request-mode class still takes bloom's day request (no slot), and the day rules still apply.
	r = book(f.sara, map[string]any{"listing_id": f.class, "preferred_date": "2026-09-30", "child_id": f.saraChild})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Nil(t, r.data()["booking"].(map[string]any)["slot"])
}

// CB-DIR-03 acceptance: N mothers race for a slot with 2 seats — exactly 2 requests win, booked_count never overflows.
func TestDirectoryBookings_NoOverbooking(t *testing.T) {
	e := newCompanionEnv(t)
	f := seedDirBookings(t, e)
	const racers = 8
	tokens := make([]string, racers)
	for i := range tokens {
		tokens[i] = e.user("Racer "+itoa(uint64(i)), "0912100"+itoa(uint64(1000+i)))
	}
	statuses := make([]int, racers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range tokens {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			statuses[i] = e.rawDo(fiber.MethodPost, f.bookURL, tokens[i], map[string]any{"listing_id": f.pool, "slot_id": f.small})
		}(i)
	}
	close(start)
	wg.Wait()
	won := 0
	for _, s := range statuses {
		switch s {
		case fiber.StatusCreated:
			won++
		case fiber.StatusUnprocessableEntity:
		default:
			t.Fatalf("unexpected status %d (%v)", s, statuses)
		}
	}
	assert.Equal(t, 2, won, statuses)
	assert.Equal(t, 2, e.seats(f.small))
	assert.Equal(t, 2, e.count(`SELECT COUNT(*) FROM city_service_bookings WHERE slot_id = ?`, f.small))
}

// Security audit (directory/shop) L1: one mother double-tapping «رزرو» in parallel gets exactly one booking — for one
// slot (one seat per mother and slot) and for a day request (one open request per venue) — and her open cap holds.
func TestDirectoryBookings_SameUserRace(t *testing.T) {
	e := newCompanionEnv(t)
	f := seedDirBookings(t, e)
	race := func(n int, body map[string]any) (won int) {
		statuses := make([]int, n)
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := range n {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				statuses[i] = e.rawDo(fiber.MethodPost, f.bookURL, f.sara, body)
			}(i)
		}
		close(start)
		wg.Wait()
		for _, s := range statuses {
			switch s {
			case fiber.StatusCreated:
				won++
			case fiber.StatusUnprocessableEntity:
			default:
				t.Fatalf("unexpected status %d (%v)", s, statuses)
			}
		}
		return won
	}
	assert.Equal(t, 1, race(6, map[string]any{"listing_id": f.pool, "slot_id": f.s1}))
	assert.Equal(t, 1, e.seats(f.s1), "the losers took no seat")
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM city_service_bookings WHERE user_id = ? AND slot_id = ?`, f.saraUser, f.s1))
	assert.Equal(t, 1, race(6, map[string]any{"listing_id": f.class, "preferred_date": "2026-09-30"}))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM city_service_bookings WHERE user_id = ? AND listing_id = ?`, f.saraUser, f.class))

	// The open cap (MaxOpenBookings) under a parallel burst over distinct slots: never above it.
	slots := make([]uint64, cityservices.MaxOpenBookings+4)
	for i := range slots {
		slots[i] = e.insExec(`INSERT INTO city_service_slots (listing_id, service_id, starts_at, ends_at, capacity, booked_count, status,
			created_at, updated_at) VALUES (?, NULL, DATE_ADD('2026-10-01 09:00:00', INTERVAL ? DAY), DATE_ADD('2026-10-01 10:00:00', INTERVAL ? DAY),
			6, 0, 'open', NOW(), NOW())`, f.pool, i, i)
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, s := range slots {
		wg.Add(1)
		go func(s uint64) {
			defer wg.Done()
			<-start
			e.rawDo(fiber.MethodPost, f.bookURL, f.sara, map[string]any{"listing_id": f.pool, "slot_id": s})
		}(s)
	}
	close(start)
	wg.Wait()
	assert.Equal(t, cityservices.MaxOpenBookings, e.count(`SELECT COUNT(*) FROM city_service_bookings WHERE user_id = ?
		AND status IN ('requested','confirmed')`, f.saraUser))
}

// Security audit (directory/shop) L2: a reschedule keeps the service (a service-less slot is another service) and
// re-checks the child's age on the new date.
func TestDirectoryBookings_RescheduleGuards(t *testing.T) {
	e := newCompanionEnv(t)
	f := seedDirBookings(t, e)
	r := e.do(fiber.MethodPost, f.bookURL, f.sara, map[string]any{"listing_id": f.pool, "slot_id": f.s1, "child_id": f.saraChild})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	bid := itoa(uint64(r.data()["booking"].(map[string]any)["id"].(float64)))
	noService := e.insExec(`INSERT INTO city_service_slots (listing_id, service_id, starts_at, ends_at, capacity, booked_count, status,
		created_at, updated_at) VALUES (?, NULL, '2026-09-28 17:00:00', '2026-09-28 18:00:00', 6, 0, 'open', NOW(), NOW())`, f.pool)
	tooOld := e.insExec(`INSERT INTO city_service_slots (listing_id, service_id, starts_at, ends_at, capacity, booked_count, status,
		created_at, updated_at) VALUES (?, ?, '2027-03-01 17:00:00', '2027-03-01 18:00:00', 6, 0, 'open', NOW(), NOW())`, f.pool, f.baby)
	resched := func(slot uint64) reply {
		return e.do(fiber.MethodPost, f.bookingsURL+"/"+bid+"/reschedule", f.sara, map[string]any{"slot_id": slot})
	}
	r = resched(noService)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "slot_id", "a service booking cannot move to a service-less slot")
	r = resched(tooOld) // Arian is 19 months on 2027-03-01: Water intro is 6–18
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "slot_id")
	assert.Equal(t, 1, e.seats(f.s1), "nothing moved")
	assert.Equal(t, 0, e.seats(noService)+e.seats(tooOld))
	require.Equal(t, fiber.StatusOK, resched(f.s2).Status, "same service, child fits")
}

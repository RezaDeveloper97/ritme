package cityservices_test

// CB-DIR-05 integration: application review → published place → services / slots → booking confirmed (discreet
// notice) → review moderated (rating recomputed), plus the capability, CSRF, IDOR and capacity guards.

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/cityservices"
	"github.com/ritme/backend-go/internal/cityservices/join"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

type captureNotifier struct {
	mu   sync.Mutex
	sent []cityservices.StatusNotice
}

func (n *captureNotifier) BookingStatusChanged(_ context.Context, s cityservices.StatusNotice) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, s)
	return nil
}

type dirEnv struct {
	*admintest.Env
	files  *files.Service
	notify *captureNotifier
}

const (
	marketingID uint64 = 3 // content manage, no support
	supportID   uint64 = 4 // support manage, no content
)

func newDirEnv(t *testing.T) *dirEnv {
	t.Helper()
	e := admintest.New(t)
	vault, err := files.NewVault(e.Storage, files.DevKey("dir05-test"), nil, false)
	require.NoError(t, err)
	fs := files.NewService(files.Options{DB: e.DB, Vault: vault, BaseURL: "http://localhost"})
	n := &captureNotifier{}
	cityservices.NewAdmin(e.DB, media.NewDisk(e.Storage), "http://localhost", admintest.Quiet).
		WithFiles(fs).WithStatusNotifier(n).Routes(e.Route(), e.Kit)
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(3, 'Mkt', 'mkt@ritme.test', 'x', 'marketing', 1, NOW(), NOW()),
		(4, 'Sup', 'sup@ritme.test', 'x', 'support', 1, NOW(), NOW())`)
	return &dirEnv{Env: e, files: fs, notify: n}
}

func pngBytes(t *testing.T, seed uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 320, 240))
	for x := range 320 {
		img.Set(x, 0, color.RGBA{R: seed, G: uint8(x), B: 9, A: 255}) //nolint:gosec // G115: test pattern
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	return b.Bytes()
}

// seedApplication stores a submitted application of a new user with nPhotos photos and a licence; returns ids.
func (e *dirEnv) seedApplication(t *testing.T, cityID int64, nPhotos int, mode string) (appID, userID uint64, photoIDs []uint64) {
	t.Helper()
	res := e.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Owner Person', NULL, NOW(), NOW())`)
	uid, _ := res.LastInsertId()
	userID = uint64(uid) //nolint:gosec // G115: test id
	now := time.Now()
	for i := range nPhotos {
		f, err := e.files.Put(t.Context(), userID, files.PurposePlacePhoto, pngBytes(t, uint8(i+1)), now) //nolint:gosec // G115: small
		require.NoError(t, err)
		photoIDs = append(photoIDs, f.ID)
	}
	lic, err := e.files.Put(t.Context(), userID, files.PurposePlaceLicence, pngBytes(t, 200), now)
	require.NoError(t, err)
	pj := "["
	for i, id := range photoIDs {
		if i > 0 {
			pj += ","
		}
		pj += itoa(int(id)) //nolint:gosec // G115: test id
	}
	pj += "]"
	res = e.Exec(`INSERT INTO city_service_applications (user_id, tracking_code, status, step, business_name, category_slugs,
		city_id, district, contact_name, contact_phone, owner_confirmed, lat, lng, address, about, age_ranges, amenities,
		weekly_hours, photo_file_ids, services, booking_mode, documents, terms_accepted_at, submitted_at, created_at, updated_at)
		VALUES (?, ?, 'submitted', 4, 'Little Swimmers', '["pool"]', ?, 'Vanak', 'Owner Person', '09121112233', 1,
		'35.700000', '51.400000', 'Street 1', 'Warm pool', '["m0_6"]', '["parking"]',
		'[{"day":0,"open":"09:00","close":"18:00"}]', ?,
		'[{"kind":"session","name":"Baby swim","duration_minutes":45,"age_min_months":3,"age_max_months":24,"capacity":6,"price_toman":500000}]',
		?, ?, NOW(), NOW(), NOW(), NOW())`,
		userID, "TRK-"+itoa(int(uid)), cityID, pj, mode, `{"licence":`+itoa(int(lic.ID))+`}`) //nolint:gosec // G115: test ids
	id, _ := res.LastInsertId()
	return uint64(id), userID, photoIDs //nolint:gosec // G115: test id
}

func (e *dirEnv) seedCity(t *testing.T) int64 {
	t.Helper()
	res := e.Exec(`INSERT INTO city_service_cities (slug, name, sort_order, is_active, created_at, updated_at)
		VALUES (?, '{"fa":"تهران","en":"Tehran"}', 0, 1, NOW(), NOW())`, "c"+itoa(int(time.Now().UnixNano()%1e9)))
	id, _ := res.LastInsertId()
	return id
}

func TestDirAdmin_CapabilitiesCSRFAndPII(t *testing.T) {
	e := newDirEnv(t)
	city := e.seedCity(t)
	app, _, _ := e.seedApplication(t, city, 1, "request")
	a := itoa(int(app)) //nolint:gosec // G115: test id

	assert.Equal(t, httpadmin.CapContent, httpadmin.RouteCapability(httpadmin.Prefix+"/city-service-applications/1"))
	assert.Equal(t, httpadmin.CapContent, httpadmin.RouteCapability(httpadmin.Prefix+"/city-services/reviews"))
	assert.Equal(t, 401, e.Anonymous().Get("/city-service-applications").Status)
	assert.Equal(t, 403, e.As(supportID).Get("/city-service-applications").Status, "support lacks content")
	assert.Equal(t, 403, e.As(supportID).Get("/city-services/reviews").Status)

	// Contact details need «support» too: the marketing role (content only) gets them hidden.
	r := e.As(marketingID).Get("/city-service-applications/" + a)
	require.Equal(t, 200, r.Status, r.Body)
	app1 := r.Obj("application")
	assert.Nil(t, app1["contact_phone"])
	assert.Equal(t, true, app1["contact_hidden"])
	// Security audit M1: so are the private documents — not listed, and no link (403 permission, not 404).
	assert.Empty(t, app1["documents"])
	r = e.As(marketingID).Get("/city-service-applications/" + a + "/documents/licence")
	assert.Equal(t, 403, r.Status, r.Body)
	assert.NotContains(t, r.Body, "signature=")
	r = e.As(admintest.EditorID).Get("/city-service-applications/" + a)
	assert.Equal(t, "09121112233", r.Obj("application")["contact_phone"])
	assert.Len(t, r.Obj("application")["photos"], 1)
	assert.Equal(t, true, r.Obj("application")["documents"].([]any)[0].(map[string]any)["present"])
	assert.Equal(t, 200, e.As(admintest.EditorID).Get("/city-service-applications/"+a+"/documents/licence").Status)

	noToken := e.As(admintest.EditorID)
	noToken.CSRF = ""
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/approve", map[string]any{}).Status)
	assert.Equal(t, "submitted", e.String(`SELECT status FROM city_service_applications WHERE id = ?`, app))
	assert.Equal(t, 404, e.As(admintest.EditorID).Get("/city-service-applications/999999").Status)
}

func TestDirAdmin_ApplicationToReview_EndToEnd(t *testing.T) {
	e := newDirEnv(t)
	c := e.As(admintest.EditorID)
	city := e.seedCity(t)
	app, owner, photos := e.seedApplication(t, city, 2, "online")
	a := itoa(int(app)) //nolint:gosec // G115: test id

	// Queue + licence via a signed link only.
	r := c.Get("/city-service-applications?status=submitted")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 1)
	r = c.Get("/city-service-applications/" + a + "/documents/licence")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Contains(t, r.Data()["url"], "/download?")
	assert.Contains(t, r.Data()["url"], "signature=")
	assert.Equal(t, 404, c.Get("/city-service-applications/"+a+"/documents/health_cert").Status)

	// needs_info needs a note; then back to submitted (the applicant resubmits).
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/review", map[string]any{"status": "needs_info"}).Status)
	r = c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/review", map[string]any{"status": "needs_info", "note": "Clearer licence"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, false, r.Obj("application")["can_approve"])
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/approve", map[string]any{}).Status)
	e.Exec(`UPDATE city_service_applications SET status = 'submitted' WHERE id = ?`, app)
	r = c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/review", map[string]any{"status": "in_review"})
	require.Equal(t, 200, r.Status, r.Body)

	// Approve → a published, verified place with the application's photos, hours and services.
	r = c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/approve", map[string]any{"note": "Welcome"})
	require.Equal(t, 200, r.Status, r.Body)
	lid := int(r.Obj("application")["listing_id"].(float64))
	assert.Equal(t, "approved", r.Obj("application")["status"])
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/approve", map[string]any{}).Status, "approved is final")
	l := itoa(lid)
	r = c.Get("/city-services/listings/" + l)
	require.Equal(t, 200, r.Status, r.Body)
	place := r.Obj("listing")
	assert.Equal(t, "published", place["status"])
	assert.Equal(t, true, place["verified"])
	assert.Equal(t, "online", place["booking_mode"])
	assert.Len(t, place["photo_file_ids"], len(photos))
	assert.Len(t, place["file_photo_urls"], len(photos))
	assert.Len(t, place["weekly_hours"], 1)
	assert.Equal(t, int(owner), e.Int(`SELECT owner_user_id FROM city_service_listings WHERE id = ?`, lid)) //nolint:gosec // G115: test id
	r = c.Get("/city-services/listings/" + l + "/services")
	require.Len(t, r.Items(), 1)
	svc := int(r.Items()[0].(map[string]any)["id"].(float64))

	// Directory fields: status / verified / catalog codes validated.
	dir := map[string]any{"status": "suspended", "verified": false, "age_ranges": []string{"nope"}}
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, "/city-services/listings/"+l+"/directory", dir).Status)
	dir["age_ranges"] = []string{"m0_6"}
	dir["website_url"] = "javascript:alert(1)"
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, "/city-services/listings/"+l+"/directory", dir).Status)
	dir["status"], dir["verified"], dir["website_url"] = "published", true, "https://swim.example"
	dir["cancel_free_hours"] = 24
	r = c.JSON(fiber.MethodPut, "/city-services/listings/"+l+"/directory", dir)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, float64(24), r.Obj("listing")["cancel_free_hours"])

	// Slots: single + weekly generator (idempotent), capacity guard, IDOR.
	tomorrow := time.Now().In(civildate.Tehran).AddDate(0, 0, 1)
	r = c.JSON(fiber.MethodPost, "/city-services/listings/"+l+"/slots", map[string]any{
		"service_id": svc, "starts_at": tomorrow.Format("2006-01-02") + " 08:00", "capacity": 2})
	require.Equal(t, 201, r.Status, r.Body)
	gen := map[string]any{"from": tomorrow.Format("2006-01-02"), "to": tomorrow.AddDate(0, 0, 6).Format("2006-01-02"),
		"weekdays": []int{0, 1, 2, 3, 4, 5, 6}, "times": []string{"10:00", "12:00"}, "capacity": 3, "service_id": svc}
	r = c.JSON(fiber.MethodPost, "/city-services/listings/"+l+"/slots/generate", gen)
	require.Equal(t, 201, r.Status, r.Body)
	assert.Equal(t, float64(14), r.Data()["created"])
	r = c.JSON(fiber.MethodPost, "/city-services/listings/"+l+"/slots/generate", gen)
	assert.Equal(t, float64(0), r.Data()["created"], "same starts are skipped")
	gen["to"] = tomorrow.AddDate(0, 0, 200).Format("2006-01-02")
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-services/listings/"+l+"/slots/generate", gen).Status)
	r = c.Get("/city-services/listings/" + l + "/slots?from=" + tomorrow.Format("2006-01-02"))
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 15)
	slot := int(r.Items()[0].(map[string]any)["id"].(float64))
	assert.Equal(t, "08:00", strings.Split(r.Items()[0].(map[string]any)["starts_at"].(string), "T")[1][:5])
	s := itoa(slot)

	// Another place cannot reach this place's slot or service (IDOR → 404).
	other := c.JSON(fiber.MethodPost, "/city-services/listings", map[string]any{"name": "Other", "booking_mode": "request",
		"city_id": city, "category_id": place["category_id"], "is_active": true})
	require.Equal(t, 201, other.Status, other.Body)
	o := itoa(int(other.Obj("listing")["id"].(float64)))
	assert.Equal(t, 404, c.JSON(fiber.MethodPut, "/city-services/listings/"+o+"/slots/"+s, map[string]any{"capacity": 1, "status": "open"}).Status)
	assert.Equal(t, 404, c.JSON(fiber.MethodDelete, "/city-services/listings/"+o+"/services/"+itoa(svc), nil).Status)

	// A mother books the 08:00 slot (two seats taken): capacity can't drop below 2, the slot can't be deleted.
	ures := e.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Mom', '09120001111', NOW(), NOW())`)
	uid, _ := ures.LastInsertId()
	cres := e.Exec(`INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, 'Nika', ?, NOW(), NOW())`,
		uid, tomorrow.AddDate(0, -6, 0).Format("2006-01-02"))
	child, _ := cres.LastInsertId()
	bres := e.Exec(`INSERT INTO city_service_bookings (user_id, listing_id, contact_name, contact_phone, preferred_date, status,
		code, service_id, slot_id, child_id, price_toman, created_at, updated_at)
		VALUES (?, ?, 'Sara Mom', '09120001111', ?, 'requested', 'ABCD2345', ?, ?, ?, 500000, NOW(), NOW())`,
		uid, lid, tomorrow.Format("2006-01-02"), svc, slot, child)
	bid, _ := bres.LastInsertId()
	e.Exec(`UPDATE city_service_slots SET booked_count = 2 WHERE id = ?`, slot)
	r = c.JSON(fiber.MethodPut, "/city-services/listings/"+l+"/slots/"+s, map[string]any{"capacity": 1, "status": "open"})
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, cityservices.CodeBelowBooked, r.Code())
	assert.Equal(t, 200, c.JSON(fiber.MethodPut, "/city-services/listings/"+l+"/slots/"+s, map[string]any{"capacity": 2, "status": "closed"}).Status)
	assert.Equal(t, cityservices.CodeInUse, c.JSON(fiber.MethodDelete, "/city-services/listings/"+l+"/slots/"+s, nil).Code())
	assert.Equal(t, cityservices.CodeInUse, c.JSON(fiber.MethodDelete, "/city-services/listings/"+l+"/services/"+itoa(svc), nil).Code())

	// Booking queue: the new fields, confirm → discreet notice (ids + status only), complete.
	b := itoa(int(bid))
	r = c.Get("/city-service-bookings/" + b)
	require.Equal(t, 200, r.Status, r.Body)
	bk := r.Obj("booking")
	assert.Equal(t, "ABCD2345", bk["code"])
	assert.Equal(t, "Nika", bk["child"].(map[string]any)["name"])
	assert.NotNil(t, bk["slot"])
	assert.Equal(t, "Baby swim", bk["service"].(map[string]any)["name"])
	assert.Equal(t, 403, e.As(marketingID).Get("/city-service-bookings/"+b).Status, "the queue (with the child's name) is support-only")
	require.Equal(t, 200, c.JSON(fiber.MethodPut, "/city-service-bookings/"+b, map[string]any{"status": "confirmed"}).Status)
	require.Len(t, e.notify.sent, 1)
	assert.Equal(t, cityservices.StatusNotice{BookingID: uint64(bid), UserID: uint64(uid), Status: "confirmed"}, e.notify.sent[0]) //nolint:gosec // G115: test ids
	require.Equal(t, 200, c.JSON(fiber.MethodPut, "/city-service-bookings/"+b, map[string]any{"status": "completed"}).Status)
	assert.Len(t, e.notify.sent, 1, "completion is not notified")

	// Review moderation recomputes the rating.
	rres := e.Exec(`INSERT INTO city_service_reviews (listing_id, booking_id, user_id, rating, body, status, created_at, updated_at)
		VALUES (?, ?, ?, 4, 'Lovely', 'pending', NOW(), NOW())`, lid, bid, uid)
	rid, _ := rres.LastInsertId()
	r = c.Get("/city-services/reviews?status=pending")
	require.Len(t, r.Items(), 1)
	assert.Equal(t, "Sara", r.Items()[0].(map[string]any)["author"], "first name only")
	rv := "/city-services/reviews/" + itoa(int(rid))
	require.Equal(t, 200, c.JSON(fiber.MethodPut, rv, map[string]any{"status": "published"}).Status)
	assert.Equal(t, 1, e.Int(`SELECT rating_count FROM city_service_listings WHERE id = ?`, lid))
	assert.Equal(t, "4.00", e.String(`SELECT rating_avg FROM city_service_listings WHERE id = ?`, lid))
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, rv, map[string]any{"status": "published"}).Status)
	require.Equal(t, 200, c.JSON(fiber.MethodPut, rv, map[string]any{"status": "hidden"}).Status)
	assert.Equal(t, 0, e.Int(`SELECT rating_count FROM city_service_listings WHERE id = ?`, lid))
	assert.Empty(t, e.String(`SELECT rating_avg FROM city_service_listings WHERE id = ?`, lid))

	// Reports queue.
	pres := e.Exec(`INSERT INTO city_service_reports (listing_id, user_id, reason, note, status, created_at, updated_at)
		VALUES (?, ?, 'closed', 'Closed on Fridays', 'open', NOW(), NOW())`, lid, uid)
	pid, _ := pres.LastInsertId()
	r = c.Get("/city-services/reports?status=open")
	require.Len(t, r.Items(), 1)
	assert.NotContains(t, r.Items()[0].(map[string]any), "user_id")
	rp := "/city-services/reports/" + itoa(int(pid))
	require.Equal(t, 200, c.JSON(fiber.MethodPut, rp, map[string]any{"status": "resolved"}).Status)
	assert.Equal(t, 422, c.JSON(fiber.MethodPut, rp, map[string]any{"status": "dismissed"}).Status)
}

func TestDirAdmin_RejectDeletesPhotos(t *testing.T) {
	e := newDirEnv(t)
	c := e.As(admintest.EditorID)
	app, owner, photos := e.seedApplication(t, e.seedCity(t), 2, "phone")
	a := itoa(int(app)) //nolint:gosec // G115: test id

	// Phone mode needs the venue's public phone to approve (never copied from the applicant's contact).
	r := c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/approve", map[string]any{})
	assert.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "phone")

	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/review", map[string]any{"status": "rejected"}).Status, "note required")
	r = c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/review", map[string]any{"status": "rejected", "note": "Not a venue for children"})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "rejected", r.Obj("application")["status"])
	assert.Empty(t, r.Obj("application")["photos"])
	for _, id := range photos {
		_, err := e.files.Get(t.Context(), owner, id)
		require.ErrorIs(t, err, files.ErrNotFound)
	}
	assert.Equal(t, 422, c.JSON(fiber.MethodPost, "/city-service-applications/"+a+"/review", map[string]any{"status": "in_review"}).Status, "rejected is final")
	// The licence (private) stays for the record.
	r = c.Get("/city-service-applications/" + a)
	docs := r.Obj("application")["documents"].([]any)
	assert.Equal(t, true, docs[0].(map[string]any)["present"])
}

func TestStatusNotifier_FakeLogsIdsOnly(t *testing.T) {
	var buf bytes.Buffer
	n := cityservices.FakeStatusNotifier{Logger: slog.New(slog.NewTextHandler(&buf, nil))}
	require.NoError(t, n.BookingStatusChanged(context.Background(), cityservices.StatusNotice{BookingID: 9, UserID: 5, Status: "confirmed"}))
	assert.Contains(t, buf.String(), "booking_id=9")
	assert.Contains(t, buf.String(), "status=confirmed")
}

func TestApplicationReviewMoves(t *testing.T) {
	assert.True(t, join.CanReview(join.StatusSubmitted, join.StatusInReview))
	assert.True(t, join.CanReview(join.StatusNeedsInfo, join.StatusRejected))
	assert.False(t, join.CanReview(join.StatusRejected, join.StatusInReview))
	assert.False(t, join.CanReview(join.StatusSubmitted, join.StatusApproved), "approval goes through Approve")
	assert.True(t, join.CanApprove(join.StatusInReview))
	assert.False(t, join.CanApprove(join.StatusNeedsInfo))
	assert.Equal(t, []string{}, join.ReviewMoves(join.StatusApproved))
}

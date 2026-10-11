package http

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/clock"
)

// joinUpload sends one multipart `file` to path (CB-DIR-04 photo / document routes).
func (e *companionEnv) joinUpload(method, path, token, name string, data []byte) reply {
	e.t.Helper()
	// Blobs land under STORAGE_PATH (the committed key fixtures directory): remove them afterwards.
	e.t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(keysDir, "app")) })
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", name)
	require.NoError(e.t, err)
	_, err = part.Write(data)
	require.NoError(e.t, err)
	require.NoError(e.t, w.Close())
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, companionNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	out := reply{Status: resp.StatusCode, Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func joinPNG(t *testing.T, shade uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	img.Set(1, 1, color.RGBA{R: shade, G: 10, B: 10, A: 255})
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	return b.Bytes()
}

var joinPDF = []byte("%PDF-1.4 licence\n%%EOF")

func TestCityServicesJoin_Flow(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009101")
	omid := e.user("Omid", "09120009102")
	const p = "/api/v1/city-services/applications"

	for _, path := range []string{p, p + "/1", p + "/1/status"} {
		assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, path, "", nil).Status, path)
	}
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, p, "", nil).Status)
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, p+"/1/submit", "", nil).Status)

	city, _, _, _ := seedCityServices(t, e)

	// Open a draft; a second POST returns the same draft.
	r := e.do(fiber.MethodPost, p, sara, nil)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	app := r.data()["application"].(map[string]any)
	assert.Equal(t, "draft", app["status"])
	assert.Equal(t, true, app["editable"])
	assert.Nil(t, app["tracking_code"])
	id := itoa(idOf(app["id"]))
	r = e.do(fiber.MethodPost, p, sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, id, itoa(idOf(r.data()["application"].(map[string]any)["id"])))

	// Submit an empty draft → every missing field, nothing stored.
	r = e.do(fiber.MethodPost, p+"/"+id+"/submit", sara, nil)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "application_incomplete", r.Body["error_code"])
	errs := r.Body["errors"].(map[string]any)
	for _, f := range []string{"business_name", "photos", "services", "documents.licence", "terms"} {
		assert.Contains(t, errs, f)
	}

	// Step 1 — validation (unknown category, foreign city, bad phone), then a valid save (Persian digits folded).
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/1", sara, map[string]any{"business_name": "Ab Pari",
		"category_slugs": []string{"pool", "nope", "pool"}, "city_id": 99999, "contact_phone": "12345"})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	errs = r.Body["errors"].(map[string]any)
	assert.Contains(t, errs, "category_slugs.1")
	assert.Contains(t, errs, "category_slugs.2")
	assert.Contains(t, errs, "city_id")
	assert.Contains(t, errs, "contact_phone")
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/1", sara, map[string]any{"business_name": "Ab Pari",
		"category_slugs": []string{"pool", "class"}, "city_id": city, "district": "Vanak", "contact_name": "Sara",
		"contact_phone": "۰۹۱۲ ۱۲۳ ۴۵۶۷", "owner_confirmed": true, "diagnosis": "ignored"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	app = r.data()["application"].(map[string]any)
	assert.Equal(t, "09121234567", app["contact_phone"])
	assert.EqualValues(t, 1, app["step"])
	assert.NotContains(t, r.Raw, "diagnosis")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPut, p+"/"+id+"/steps/5", sara, map[string]any{}).Status)

	// Step 2 — catalog codes, hours order, coordinates.
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/2", sara, map[string]any{"lat": 35.7, "age_ranges": []string{"bogus"},
		"weekly_hours": []map[string]any{{"day": 0, "open": "20:00", "close": "09:00"}}})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	errs = r.Body["errors"].(map[string]any)
	assert.Contains(t, errs, "lat")
	assert.Contains(t, errs, "age_ranges.0")
	assert.Contains(t, errs, "weekly_hours.0.close")
	// A JSON object where a list belongs is a 422, never a 500 (it used to nil-deref) nor a `.*` rule bypass.
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/2", sara, map[string]any{"weekly_hours": map[string]any{"a": map[string]any{"open": "09:00"}}})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"].(map[string]any), "weekly_hours")
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/3", sara, map[string]any{"services": map[string]any{"x": map[string]any{"kind": "nope"}}})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"].(map[string]any), "services")
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/2", sara, map[string]any{"lat": 35.757, "lng": 51.41, "address": "Vanak sq.",
		"about": "Warm pool.\nFemale coaches.", "age_ranges": []string{"m6_12", "y1_2"}, "amenities": []string{"nursing_room"},
		"weekly_hours": []map[string]any{{"day": 5, "open": "09:00", "close": "14:00"}, {"day": 0, "open": "09:00", "close": "20:00"}}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	app = r.data()["application"].(map[string]any)
	assert.InDelta(t, 35.757, app["lat"], 1e-9)
	assert.EqualValues(t, 0, app["weekly_hours"].([]any)[0].(map[string]any)["day"], "sorted by day")

	// Step 3 — services.
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/3", sara, map[string]any{"services": []map[string]any{
		{"kind": "session", "name": "Water intro", "duration_minutes": 45, "age_min_months": 18, "age_max_months": 6}}})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"].(map[string]any), "services.0.age_max_months")
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/3", sara, map[string]any{"services": []map[string]any{
		{"kind": "session", "name": "Water intro", "duration_minutes": 45, "age_min_months": 6, "age_max_months": 18, "capacity": 6, "price_toman": 320000},
		{"kind": "package", "name": "8 sessions", "package_sessions": 8, "package_validity_days": 60, "price_toman": 2400000}}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["application"].(map[string]any)["services"], 2)

	// Step 4.
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/4", sara, map[string]any{"booking_mode": "teleport"})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/4", sara, map[string]any{"booking_mode": "online", "terms_accepted": true})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotNil(t, r.data()["application"].(map[string]any)["terms_accepted_at"])

	// Photos (public, re-encoded) and private documents.
	var photoIDs []string
	for i := range 4 {
		r = e.joinUpload(fiber.MethodPost, p+"/"+id+"/photos", sara, "room.png", joinPNG(t, uint8(40+i)))
		require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
		ph := r.data()["photo"].(map[string]any)
		assert.Equal(t, "image/webp", ph["mime"])
		assert.Contains(t, ph["url"], "/api/v1/files/public/")
		photoIDs = append(photoIDs, itoa(idOf(ph["id"])))
	}
	r = e.joinUpload(fiber.MethodPost, p+"/"+id+"/photos", sara, "fake.png", []byte("not an image"))
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	r = e.joinUpload(fiber.MethodPut, p+"/"+id+"/documents/passport", sara, "x.pdf", joinPDF)
	assert.Equal(t, fiber.StatusNotFound, r.Status, "unknown slot")
	r = e.joinUpload(fiber.MethodPut, p+"/"+id+"/documents/licence", sara, "licence.pdf", joinPDF)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	doc := r.data()["document"].(map[string]any)
	assert.Contains(t, doc["url"], "/download?expires=", "private: a signed, expiring link — never a public URL")
	assert.NotNil(t, doc["url_expires_at"])
	firstLicence := idOf(doc["id"])
	r = e.joinUpload(fiber.MethodPut, p+"/"+id+"/documents/licence", sara, "licence2.pdf", joinPDF)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, 0, e.count("SELECT COUNT(*) FROM files WHERE id = ?", firstLicence), "a replaced document's file is deleted")
	assert.Equal(t, 1, e.count("SELECT COUNT(*) FROM files WHERE purpose = 'place_licence' AND visibility = 'private'"))

	// Pool → the health certificate is still missing.
	r = e.do(fiber.MethodPost, p+"/"+id+"/submit", sara, nil)
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, []string{"documents.health_cert"}, keys(r.Body["errors"].(map[string]any)))
	r = e.joinUpload(fiber.MethodPut, p+"/"+id+"/documents/health_cert", sara, "health.png", joinPNG(t, 90))
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)

	// IDOR: Omid sees, edits, uploads to and submits nothing of Sara's (uniform 404), and cannot read her files.
	for _, req := range []struct{ m, path string }{{fiber.MethodGet, p + "/" + id}, {fiber.MethodGet, p + "/" + id + "/status"},
		{fiber.MethodPut, p + "/" + id + "/steps/1"}, {fiber.MethodPost, p + "/" + id + "/submit"},
		{fiber.MethodDelete, p + "/" + id}, {fiber.MethodDelete, p + "/" + id + "/photos/" + photoIDs[0]},
		{fiber.MethodDelete, p + "/" + id + "/documents/licence"}} {
		assert.Equal(t, fiber.StatusNotFound, e.do(req.m, req.path, omid, map[string]any{}).Status, req.m+" "+req.path)
	}
	assert.Equal(t, fiber.StatusNotFound, e.joinUpload(fiber.MethodPost, p+"/"+id+"/photos", omid, "x.png", joinPNG(t, 1)).Status)
	assert.Equal(t, fiber.StatusNotFound, e.joinUpload(fiber.MethodPut, p+"/"+id+"/documents/licence", omid, "x.pdf", joinPDF).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/files/"+itoa(idOf(doc["id"])), omid, nil).Status)
	r = e.do(fiber.MethodGet, p, omid, nil)
	assert.Empty(t, r.data()["applications"])

	// Submit → tracking code, status timeline; the draft is now locked (photos still open).
	r = e.do(fiber.MethodPost, p+"/"+id+"/submit", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	app = r.data()["application"].(map[string]any)
	assert.Equal(t, "submitted", app["status"])
	code, _ := app["tracking_code"].(string)
	assert.Regexp(t, `^[2-9A-HJ-NP-Z]{4}-[2-9A-HJ-NP-Z]{4}-[2-9A-HJ-NP-Z]{4}$`, code)
	assert.Contains(t, r.Body["message"], code)
	assert.Equal(t, false, app["editable"])
	assert.Equal(t, true, app["can_add_photos"])
	r = e.do(fiber.MethodGet, p+"/"+id+"/status", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	tl := r.data()["timeline"].([]any)
	assert.Equal(t, "done", tl[0].(map[string]any)["state"])
	assert.Equal(t, "current", tl[1].(map[string]any)["state"])
	assert.Equal(t, fiber.StatusConflict, e.do(fiber.MethodPut, p+"/"+id+"/steps/1", sara, map[string]any{}).Status)
	assert.Equal(t, fiber.StatusConflict, e.do(fiber.MethodPost, p+"/"+id+"/submit", sara, nil).Status)
	assert.Equal(t, fiber.StatusConflict, e.do(fiber.MethodDelete, p+"/"+id, sara, nil).Status)
	assert.Equal(t, fiber.StatusConflict, e.do(fiber.MethodDelete, p+"/"+id+"/documents/licence", sara, nil).Status)
	assert.Equal(t, fiber.StatusConflict, e.joinUpload(fiber.MethodPut, p+"/"+id+"/documents/licence", sara, "x.pdf", joinPDF).Status)
	r = e.joinUpload(fiber.MethodPost, p+"/"+id+"/photos", sara, "more.png", joinPNG(t, 200))
	assert.Equal(t, fiber.StatusCreated, r.Status, "JoinDone: more photos may be added while the team reviews")

	// needs_info → editable again; a resubmission keeps the tracking code.
	_, err := e.db.Exec("UPDATE city_service_applications SET status = 'needs_info', review_note = 'Add a photo' WHERE id = ?", id)
	require.NoError(t, err)
	r = e.do(fiber.MethodPut, p+"/"+id+"/steps/4", sara, map[string]any{"booking_mode": "request", "terms_accepted": true})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	r = e.do(fiber.MethodPost, p+"/"+id+"/submit", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, code, r.data()["application"].(map[string]any)["tracking_code"])
	assert.Nil(t, r.data()["application"].(map[string]any)["review_note"])

	// A new draft; removing a photo / document; deleting the draft drops its files.
	r = e.do(fiber.MethodPost, p, sara, nil)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	id2 := itoa(idOf(r.data()["application"].(map[string]any)["id"]))
	r = e.joinUpload(fiber.MethodPost, p+"/"+id2+"/photos", sara, "a.png", joinPNG(t, 7))
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	ph2 := itoa(idOf(r.data()["photo"].(map[string]any)["id"]))
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, p+"/"+id2+"/photos/"+photoIDs[0], sara, nil).Status, "not this application's photo")
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, p+"/"+id2+"/photos/"+ph2, sara, nil).Status)
	assert.Equal(t, 0, e.count("SELECT COUNT(*) FROM files WHERE id = ?", ph2))
	r = e.joinUpload(fiber.MethodPut, p+"/"+id2+"/documents/instructor_cert", sara, "c.pdf", joinPDF)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	cert := idOf(r.data()["document"].(map[string]any)["id"])
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, p+"/"+id2, sara, nil).Status)
	assert.Equal(t, 0, e.count("SELECT COUNT(*) FROM files WHERE id = ?", cert))
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, p+"/"+id2, sara, nil).Status)

	r = e.do(fiber.MethodGet, p, sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Len(t, r.data()["applications"], 1)

	// Privacy: no health column on the application table.
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'city_service_applications'
		AND (column_name LIKE '%health%' OR column_name LIKE '%diagnos%' OR column_name LIKE '%cycle%' OR column_name LIKE '%pregnan%')`))
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestCityServicesJoin_SubmitThrottle(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009103")
	r := e.do(fiber.MethodPost, "/api/v1/city-services/applications", sara, nil)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	id := itoa(idOf(r.data()["application"].(map[string]any)["id"]))
	for range JoinSubmitMax {
		assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/city-services/applications/"+id+"/submit", sara, nil).Status)
	}
	r = e.do(fiber.MethodPost, "/api/v1/city-services/applications/"+id+"/submit", sara, nil)
	assert.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	assert.True(t, strings.Contains(r.Raw, "too_many_requests"))
}

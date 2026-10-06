package labs_test

// B-N6-06 end to end against MariaDB with the fake AI provider: the AI gate (401, Plus 402, consent 403), the upload
// (PDF and photo), encryption at rest, the job states (sync and the async worker, lease recovery), verify / edit /
// add, the interpretation (AI summary, red flags, questions), feedback, trends, deletion (lab, page, account) and
// the IDOR matrix.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/labs/files"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffe06"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db       *sql.DB
	app      *fiber.App
	iss      *passport.Issuer
	svc      *labs.Service
	runner   *labs.Runner
	plus     *plus.Service
	consents *consent.Service
	storage  string
}

func setup(t *testing.T, sync bool) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	languages := i18n.NewRegistry(i18nstore.New(db), nil, quiet)
	locale := i18n.Middleware(languages)

	plusSvc := plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet)
	gate := plus.NewGate(plusSvc, clock.Real{})
	fake := ai.NewFake()
	client := ai.NewClientWith(ai.Options{Provider: config.AIProviderFake, Chatter: fake, Extractor: fake})
	consents := consent.NewService(db)
	g := access.NewGuard(access.Options{Client: client, Consents: consents, Gate: gate, Plus: plusSvc, Logger: quiet})

	storage := t.TempDir()
	box, err := files.New(storage, nil, nil, false)
	require.NoError(t, err)
	svc := labs.NewService(labs.Options{DB: db, Files: box, AI: client, Consents: consents, Plus: plusSvc,
		Catalog: catalog.NewReader(catalogstore.New(db), nil, 0, quiet), Logger: quiet})
	runner := labs.NewRunner(svc, labs.RunnerOptions{Sync: sync, Poll: 50 * time.Millisecond, StoragePath: storage})
	h := labs.NewHandlers(svc, languages, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet), BodyLimit: 25 << 20})
	app.Use(clock.Middleware(clock.Real{}, true))
	const p = "/api/v1/labs"
	upload := append([]any{guard}, g.Chain(ai.FeatureLabAnalysis)...)
	app.Get(p, locale, guard, h.Index)
	app.Post(p, locale, append(upload, h.Upload)...)
	app.Post(p+"/manual", locale, guard, h.StoreManual)
	app.Get(p+"/trends", locale, guard, h.Trends)
	app.Get(p+"/markers", locale, guard, h.Catalog)
	app.Get(p+"/:id", locale, guard, h.Show)
	app.Put(p+"/:id", locale, guard, h.Update)
	app.Delete(p+"/:id", locale, guard, h.Destroy)
	app.Get(p+"/:id/status", locale, guard, h.Status)
	app.Post(p+"/:id/verify", locale, guard, h.Verify)
	app.Post(p+"/:id/feedback", locale, guard, h.Feedback)
	app.Post(p+"/:id/markers", locale, guard, h.StoreMarker)
	app.Get(p+"/:id/markers/:mid", locale, guard, h.ShowMarker)
	app.Put(p+"/:id/markers/:mid", locale, guard, h.UpdateMarker)
	app.Delete(p+"/:id/markers/:mid", locale, guard, h.DestroyMarker)
	app.Get(p+"/:id/files/:fid", locale, guard, h.File)
	app.Delete(p+"/:id/files/:fid", locale, guard, h.DestroyFile)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), svc: svc, runner: runner,
		plus: plusSvc, consents: consents, storage: storage}
}

// user creates a user; plus = trial started, consent = ai_lab_analysis accepted.
func (e *env) user(t *testing.T, mobile string, trial, consented bool) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Test', ?, NOW(), NOW())`, mobile)
	require.NoError(t, err)
	id64, err := res.LastInsertId()
	require.NoError(t, err)
	id := uint64(id64) //nolint:gosec // test ids
	_, err = e.db.Exec(`INSERT INTO user_profiles (user_id, birthday, last_period_start, cycle_duration, period_duration, user_goal, created_at, updated_at)
		VALUES (?, '1995-03-01', DATE_SUB(CURDATE(), INTERVAL 20 DAY), 28, 5, 'ttc', NOW(), NOW())`, id)
	require.NoError(t, err)
	now := time.Now()
	if trial {
		require.NoError(t, e.plus.StartTrial(context.Background(), id, now))
	}
	if consented {
		require.NoError(t, e.consents.Accept(context.Background(), id, consent.AILabAnalysis, 1, now))
	}
	tok, err := e.iss.Issue(context.Background(), id, now)
	require.NoError(t, err)
	return id, tok.AccessToken
}

type resp struct {
	status int
	body   map[string]any
	raw    []byte
	header http.Header
}

func (e *env) do(t *testing.T, method, path, token, contentType string, body io.Reader) resp {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "fa")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	raw, _ := io.ReadAll(res.Body)
	out := resp{status: res.StatusCode, raw: raw, header: res.Header}
	_ = json.Unmarshal(raw, &out.body)
	return out
}

func (e *env) json(t *testing.T, method, path, token string, body any) resp {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	return e.do(t, method, path, token, "application/json", r)
}

type part struct {
	field, name string
	data        []byte
}

func multipartBody(t *testing.T, fields map[string]string, parts ...part) (string, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	for _, p := range parts {
		fw, err := w.CreateFormFile(p.field, p.name)
		require.NoError(t, err)
		_, _ = fw.Write(p.data)
	}
	require.NoError(t, w.Close())
	return w.FormDataContentType(), &buf
}

func pdf(key string) []byte { return []byte("%PDF-1.4\n% RITME-FAKE:" + key + "\n%%EOF") }

func (e *env) upload(t *testing.T, token string, fields map[string]string, parts ...part) resp {
	t.Helper()
	if fields == nil {
		fields = map[string]string{"category": "blood"}
	}
	ct, body := multipartBody(t, fields, parts...)
	return e.do(t, http.MethodPost, "/api/v1/labs", token, ct, body)
}

func data(r resp) map[string]any { m, _ := r.body["data"].(map[string]any); return m }

func markers(r resp) []any { l, _ := data(r)["markers"].([]any); return l }

func labID(r resp) uint64 { return uint64(data(r)["id"].(float64)) }

func usage(t *testing.T, e *env, userID uint64) int {
	t.Helper()
	var n sql.NullInt64
	require.NoError(t, e.db.QueryRow(`SELECT SUM(used) FROM plus_usage_counters WHERE user_id = ? AND feature = 'plus.lab_ai'`, userID).Scan(&n))
	return int(n.Int64)
}

func TestGateAndUploadPipeline(t *testing.T) {
	e := setup(t, true)
	free, freeTok := e.user(t, "09900006001", false, false)
	noConsent, ncTok := e.user(t, "09900006002", true, false)
	a, aTok := e.user(t, "09900006003", true, true)
	_ = free

	assert.Equal(t, http.StatusUnauthorized, e.json(t, http.MethodGet, "/api/v1/labs", "", nil).status)
	r := e.upload(t, freeTok, nil, part{"files", "sheet.pdf", pdf("lab_panel")})
	require.Equal(t, http.StatusPaymentRequired, r.status, string(r.raw))
	assert.Equal(t, "plus_required", r.body["error_code"])

	r = e.upload(t, ncTok, nil, part{"files", "sheet.pdf", pdf("lab_panel")})
	require.Equal(t, http.StatusForbidden, r.status, string(r.raw))
	assert.Equal(t, "consent_required", r.body["error_code"])
	assert.Equal(t, 0, usage(t, e, noConsent), "a refused request keeps the quota")

	// Validation: wrong category, not a document → 422, the reserved use is given back.
	r = e.upload(t, aTok, map[string]string{"category": "x"}, part{"files", "sheet.pdf", pdf("lab_panel")})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, string(r.raw))
	r = e.upload(t, aTok, nil, part{"files", "notes.txt", []byte("hello")})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, string(r.raw))
	assert.Equal(t, 0, usage(t, e, a))

	r = e.upload(t, aTok, map[string]string{"category": "blood", "fasting": "1", "taken_on": "2026-09-01"},
		part{"files", "sheet.pdf", pdf("lab_panel")})
	require.Equal(t, http.StatusAccepted, r.status, string(r.raw))
	assert.Equal(t, labs.StatusNeedsReview, data(r)["status"])
	assert.Equal(t, "review", data(r)["stage"])
	assert.Equal(t, "2026-09-01", data(r)["taken_on"], "the user's date wins over the sheet's")
	assert.Equal(t, "آزمایشگاه نمونه", data(r)["lab_name"])
	require.Len(t, markers(r), 5)
	first := markers(r)[0].(map[string]any)
	assert.Equal(t, "hemoglobin", first["code"])
	assert.Equal(t, "هموگلوبین", first["name"])
	assert.Equal(t, labs.StateBorderlineLow, first["state"])
	assert.Equal(t, float64(1), data(r)["low_confidence_count"], "FBS read at 0.62")
	assert.Equal(t, 1, usage(t, e, a))
	id := labID(r)

	// The sheet is encrypted at rest and comes back intact to its owner only.
	fs := data(r)["files"].([]any)
	require.Len(t, fs, 1)
	fileURL := fs[0].(map[string]any)["url"].(string)
	var path string
	require.NoError(t, e.db.QueryRow(`SELECT path FROM lab_files WHERE lab_id = ?`, id).Scan(&path))
	raw, err := os.ReadFile(filepath.Join(e.storage, path))
	require.NoError(t, err)
	assert.False(t, bytes.Contains(raw, []byte("RITME-FAKE")), "plaintext must not reach the disk")
	got := e.do(t, http.MethodGet, fileURL, aTok, "", nil)
	require.Equal(t, http.StatusOK, got.status)
	assert.Equal(t, pdf("lab_panel"), got.raw)
	assert.Equal(t, "application/pdf", got.header.Get("Content-Type"))
	assert.Contains(t, got.header.Get("Content-Disposition"), "attachment")
	assert.Equal(t, "nosniff", got.header.Get("X-Content-Type-Options"))
	assert.Equal(t, "private, no-store", got.header.Get("Cache-Control"))

	// IDOR: another user sees a uniform 404 everywhere and an empty list.
	_, bTok := e.user(t, "09900006004", true, true)
	mid := uint64(first["id"].(float64))
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, fmt.Sprintf("/api/v1/labs/%d", id)},
		{http.MethodGet, fmt.Sprintf("/api/v1/labs/%d/status", id)},
		{http.MethodPut, fmt.Sprintf("/api/v1/labs/%d", id)},
		{http.MethodDelete, fmt.Sprintf("/api/v1/labs/%d", id)},
		{http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/verify", id)},
		{http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/feedback", id)},
		{http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/markers", id)},
		{http.MethodGet, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, mid)},
		{http.MethodPut, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, mid)},
		{http.MethodDelete, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, mid)},
		{http.MethodGet, fileURL},
		{http.MethodDelete, fileURL},
	} {
		r := e.json(t, c.method, c.path, bTok, map[string]any{"category": "blood", "name": "x", "value": 1, "helpful": true})
		assert.Equal(t, http.StatusNotFound, r.status, c.method+" "+c.path)
		assert.Equal(t, "lab_not_found", r.body["error_code"], c.method+" "+c.path)
	}
	list := e.json(t, http.MethodGet, "/api/v1/labs", bTok, nil)
	assert.Empty(t, data(list)["labs"])

	// Verify / edit / add: fix FBS, add the missed B12, drop TSH, then confirm.
	var fbsID uint64
	for _, m := range markers(r) {
		if m.(map[string]any)["code"] == "glucose_fasting" {
			fbsID = uint64(m.(map[string]any)["id"].(float64))
		}
	}
	up := e.json(t, http.MethodPut, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, fbsID), aTok,
		map[string]any{"name": "FBS", "value": "۹۵", "unit": "mg/dL", "ref_low": 70, "ref_high": 100})
	require.Equal(t, http.StatusOK, up.status, string(up.raw))
	bad := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/markers", id), aTok, map[string]any{"name": "B12"})
	require.Equal(t, http.StatusUnprocessableEntity, bad.status)
	bad = e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/markers", id), aTok, map[string]any{"name": "B12", "value": 300, "ref_low": 900, "ref_high": 200})
	require.Equal(t, http.StatusUnprocessableEntity, bad.status)
	add := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/markers", id), aTok,
		map[string]any{"name": "Vitamin B12", "value": 310, "unit": "pg/mL", "ref_low": 200, "ref_high": 900})
	require.Equal(t, http.StatusCreated, add.status, string(add.raw))
	require.Len(t, markers(add), 6)
	last := markers(add)[5].(map[string]any)
	assert.Equal(t, "b12", last["code"])
	assert.Equal(t, labs.MarkerManual, last["source"])
	for _, m := range markers(add) {
		mm := m.(map[string]any)
		if mm["code"] == "glucose_fasting" {
			assert.Equal(t, labs.MarkerEdited, mm["source"])
			assert.Equal(t, float64(95), mm["value"])
		}
		if mm["code"] == "tsh" {
			del := e.json(t, http.MethodDelete, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, uint64(mm["id"].(float64))), aTok, nil)
			require.Equal(t, http.StatusOK, del.status)
			require.Len(t, markers(del), 5)
		}
	}

	v := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/verify", id), aTok, nil)
	require.Equal(t, http.StatusAccepted, v.status, string(v.raw))
	assert.Equal(t, labs.StatusReady, data(v)["status"])
	in := data(v)["interpretation"].(map[string]any)
	assert.Equal(t, labs.SummaryAI, in["source"])
	assert.Contains(t, in["summary"], "پزشک", "the fake's non-diagnostic reply")
	assert.NotEmpty(t, in["doctor_questions"])
	assert.Contains(t, in["disclaimer"], "تشخیص یا درمان نیست")
	assert.Equal(t, float64(labs.MaxInterpretations-1), data(v)["ai_interpretations_left"])
	counts := data(v)["counts"].(map[string]any)
	assert.Equal(t, float64(5), counts["total"])
	assert.Equal(t, float64(3), counts["attention"]) // hemoglobin (borderline), ferritin, vitamin D

	// Marker detail: about, factors, context notes (5-day periods, TTC), trend sentence.
	var ferritin uint64
	for _, m := range markers(v) {
		if m.(map[string]any)["code"] == "ferritin" {
			ferritin = uint64(m.(map[string]any)["id"].(float64))
		}
	}
	md := e.json(t, http.MethodGet, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, ferritin), aTok, nil)
	require.Equal(t, http.StatusOK, md.status, string(md.raw))
	assert.Equal(t, "ذخیره آهن بدن", data(md)["about"].(map[string]any)["subtitle"])
	assert.NotEmpty(t, data(md)["factors"])
	notes := fmt.Sprint(data(md)["context_notes"])
	assert.Contains(t, notes, "۵", "period length note")
	assert.Contains(t, notes, "اقدام به بارداری")
	assert.Equal(t, float64(1), data(md)["trend"].(map[string]any)["count"])

	// An edit after ready sends the upload back to review; verifying again uses one more interpretation.
	re := e.json(t, http.MethodPut, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, fbsID), aTok, map[string]any{"name": "FBS", "value": 130, "unit": "mg/dL"})
	require.Equal(t, http.StatusOK, re.status)
	assert.Equal(t, labs.StatusNeedsReview, data(re)["status"])
	assert.Equal(t, true, data(re)["interpretation"].(map[string]any)["stale"])
	v2 := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/verify", id), aTok, nil)
	require.Equal(t, http.StatusAccepted, v2.status)
	assert.Equal(t, float64(labs.MaxInterpretations-2), data(v2)["ai_interpretations_left"])
	flags := data(v2)["interpretation"].(map[string]any)["red_flags"].([]any)
	require.Len(t, flags, 1, "fasting glucose ≥ 126 is a «soon» red flag")
	assert.Equal(t, labs.SeveritySoon, flags[0].(map[string]any)["severity"])

	fb := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/feedback", id), aTok, map[string]any{"helpful": false, "note": "unclear"})
	require.Equal(t, http.StatusOK, fb.status)
	show := e.json(t, http.MethodGet, fmt.Sprintf("/api/v1/labs/%d", id), aTok, nil)
	assert.Equal(t, false, data(show)["feedback"].(map[string]any)["helpful"])

	// Lists, catalog, status.
	list = e.json(t, http.MethodGet, "/api/v1/labs", aTok, nil)
	require.Len(t, data(list)["labs"], 1)
	cat := e.json(t, http.MethodGet, "/api/v1/labs/markers", aTok, nil)
	assert.Len(t, data(cat)["markers"], 26)
	st := e.json(t, http.MethodGet, fmt.Sprintf("/api/v1/labs/%d/status", id), aTok, nil)
	assert.Equal(t, "done", data(st)["stage"])

	// Deleting a page keeps the values; deleting the lab removes its files.
	dp := e.json(t, http.MethodDelete, fileURL, aTok, nil)
	require.Equal(t, http.StatusOK, dp.status, string(dp.raw))
	assert.Empty(t, data(dp)["files"])
	assert.Len(t, markers(dp), 5)
	_, err = os.Stat(filepath.Join(e.storage, path))
	assert.True(t, os.IsNotExist(err))
	dl := e.json(t, http.MethodDelete, fmt.Sprintf("/api/v1/labs/%d", id), aTok, nil)
	require.Equal(t, http.StatusOK, dl.status)
	assert.Equal(t, http.StatusNotFound, e.json(t, http.MethodGet, fmt.Sprintf("/api/v1/labs/%d", id), aTok, nil).status)
}

func TestFailedExtractionRefundsAndPhoto(t *testing.T) {
	e := setup(t, true)
	a, aTok := e.user(t, "09900006011", true, true)

	r := e.upload(t, aTok, nil, part{"files", "sheet.pdf", pdf("error")})
	require.Equal(t, http.StatusAccepted, r.status, string(r.raw))
	assert.Equal(t, labs.StatusFailed, data(r)["status"])
	assert.Equal(t, labs.CodeAIFailed, data(r)["error_code"])
	assert.NotEmpty(t, data(r)["error_message"])
	assert.Equal(t, 0, usage(t, e, a), "a failed extraction gives the Plus use back")
	edit := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/markers", labID(r)), aTok, map[string]any{"name": "x", "value": 1})
	assert.Equal(t, http.StatusConflict, edit.status)
	vf := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/verify", labID(r)), aTok, nil)
	assert.Equal(t, http.StatusConflict, vf.status)

	r = e.upload(t, aTok, nil, part{"files", "sheet.pdf", pdf("blank")})
	assert.Equal(t, labs.CodeNothingRead, data(r)["error_code"])
	assert.Equal(t, 0, usage(t, e, a))

	// A photo is re-encoded (WebP, metadata dropped) and read like any page; two pages merge.
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for x := range 64 {
		img.Set(x, 10, color.RGBA{R: 200, A: 255})
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	r = e.upload(t, aTok, map[string]string{"category": "thyroid"}, part{"files", "p1.png", buf.Bytes()}, part{"files", "p2.pdf", pdf("lab_panel")})
	require.Equal(t, http.StatusAccepted, r.status, string(r.raw))
	assert.Equal(t, labs.StatusNeedsReview, data(r)["status"])
	assert.Len(t, markers(r), 5, "same markers on both pages are merged")
	fs := data(r)["files"].([]any)
	require.Len(t, fs, 2)
	assert.Equal(t, "image/webp", fs[0].(map[string]any)["mime"])
	got := e.do(t, http.MethodGet, fs[0].(map[string]any)["url"].(string), aTok, "", nil)
	assert.Equal(t, "image/webp", got.header.Get("Content-Type"))
	assert.Equal(t, "image/webp", ai.SniffImage(got.raw))
	assert.Equal(t, 1, usage(t, e, a))

	tooMany := []part{}
	for range labs.MaxFiles + 1 {
		tooMany = append(tooMany, part{"files", "p.pdf", pdf("lab_panel")})
	}
	r = e.upload(t, aTok, nil, tooMany...)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
}

func TestManualLabsTrendsAndAccountDeletion(t *testing.T) {
	e := setup(t, true)
	a, aTok := e.user(t, "09900006021", false, false) // manual labs need neither Plus nor consent

	for _, c := range []struct {
		date  string
		value float64
	}{{"2025-02-01", 32}, {"2026-03-01", 18}} {
		r := e.json(t, http.MethodPost, "/api/v1/labs/manual", aTok, map[string]any{
			"category": "blood", "taken_on": c.date,
			"markers": []any{
				map[string]any{"name": "Ferritin", "value": c.value, "unit": "ng/mL", "ref_low": 15, "ref_high": 150},
				map[string]any{"name": "TSH", "value": 2, "unit": "mIU/L"},
			},
		})
		require.Equal(t, http.StatusCreated, r.status, string(r.raw))
		assert.Equal(t, labs.StatusReady, data(r)["status"])
		in := data(r)["interpretation"].(map[string]any)
		assert.Equal(t, labs.SummaryRules, in["source"])
		assert.Equal(t, float64(0), data(r)["ai_interpretations_left"])
		tsh := markers(r)[1].(map[string]any)
		assert.Equal(t, "typical", tsh["reference"].(map[string]any)["source"], "no sheet range: the catalog's (units match)")
	}
	bad := e.json(t, http.MethodPost, "/api/v1/labs/manual", aTok, map[string]any{"category": "blood", "markers": []any{}})
	assert.Equal(t, http.StatusUnprocessableEntity, bad.status)
	bad = e.json(t, http.MethodPost, "/api/v1/labs/manual", aTok, map[string]any{"category": "blood", "taken_on": "2999-01-01",
		"markers": []any{map[string]any{"name": "TSH", "value": 2}}})
	assert.Equal(t, http.StatusUnprocessableEntity, bad.status)

	// A manual lab is not verified through the AI path.
	list := e.json(t, http.MethodGet, "/api/v1/labs", aTok, nil)
	firstID := uint64(data(list)["labs"].([]any)[0].(map[string]any)["id"].(float64))
	assert.Equal(t, http.StatusConflict, e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/verify", firstID), aTok, nil).status)
	upd := e.json(t, http.MethodPut, fmt.Sprintf("/api/v1/labs/%d", firstID), aTok, map[string]any{"category": "blood", "title": "چکاپ سالانه"})
	require.Equal(t, http.StatusOK, upd.status)
	assert.Equal(t, "چکاپ سالانه", data(upd)["display_title"])

	tr := e.json(t, http.MethodGet, "/api/v1/labs/trends", aTok, nil)
	require.Equal(t, http.StatusOK, tr.status, string(tr.raw))
	assert.Equal(t, float64(2), data(tr)["labs_count"])
	assert.Equal(t, "2025-02-01", data(tr)["first_date"])
	ms := data(tr)["markers"].([]any)
	require.Len(t, ms, 2)
	fer := ms[0].(map[string]any)
	assert.Equal(t, "ferritin", fer["code"])
	trend := fer["trend"].(map[string]any)
	assert.Equal(t, labs.TrendFalling, trend["direction"])
	assert.Contains(t, trend["sentence"], "کاهش")

	ready, hub, err := e.svc.HubLabs(context.Background(), a, "fa", "fa")
	require.NoError(t, err)
	assert.True(t, ready)
	b, _ := json.Marshal(hub)
	assert.Contains(t, string(b), `"values":[32,18]`)

	// Account deletion removes the user's lab directory (the rows cascade).
	rel, err := files.New(e.storage, nil, nil, false)
	require.NoError(t, err)
	_, err = rel.Put(a, []byte("sheet"))
	require.NoError(t, err)
	svc := &profile.Service{DB: e.db, Q: profilestore.New(e.db), StoragePath: e.storage, Logger: quiet}
	require.NoError(t, svc.DeleteAccount(context.Background(), a))
	_, err = os.Stat(filepath.Join(e.storage, files.Dir, fmt.Sprint(a)))
	assert.True(t, os.IsNotExist(err))
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM lab_reports WHERE user_id = ?`, a).Scan(&n))
	assert.Zero(t, n)
}

func TestAsyncWorkerAndLeaseRecovery(t *testing.T) {
	e := setup(t, false)
	_, aTok := e.user(t, "09900006031", true, true)
	require.NoError(t, e.runner.Start())
	t.Cleanup(e.runner.Shutdown)

	r := e.upload(t, aTok, nil, part{"files", "sheet.pdf", pdf("lab_panel")})
	require.Equal(t, http.StatusAccepted, r.status, string(r.raw))
	assert.Contains(t, []any{labs.StatusQueued, labs.StatusExtracting, labs.StatusNeedsReview}, data(r)["status"])
	id := labID(r)
	waitStatus := func(want string) map[string]any {
		var last map[string]any
		require.Eventually(t, func() bool {
			last = data(e.json(t, http.MethodGet, fmt.Sprintf("/api/v1/labs/%d/status", id), aTok, nil))
			return last["status"] == want
		}, 10*time.Second, 50*time.Millisecond)
		return last
	}
	st := waitStatus(labs.StatusNeedsReview)
	assert.Equal(t, float64(5), st["marker_count"])
	var status string
	require.NoError(t, e.db.QueryRow(`SELECT status FROM lab_jobs WHERE lab_id = ? AND kind = 'extract'`, id).Scan(&status))
	assert.Equal(t, "done", status)

	require.Equal(t, http.StatusAccepted, e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/verify", id), aTok, nil).status)
	waitStatus(labs.StatusReady)

	// A job left running by a crashed worker (expired lease) is picked up again.
	_, err := e.db.Exec(`UPDATE lab_reports SET status = 'interpreting' WHERE id = ?`, id)
	require.NoError(t, err)
	var userID uint64
	require.NoError(t, e.db.QueryRow(`SELECT user_id FROM lab_reports WHERE id = ?`, id).Scan(&userID))
	_, err = e.db.Exec(`INSERT INTO lab_jobs (lab_id, user_id, kind, locale, status, attempts, available_at, locked_until, created_at, updated_at)
		VALUES (?, ?, 'interpret', 'fa', 'running', 1, DATE_SUB(NOW(), INTERVAL 1 HOUR), DATE_SUB(NOW(), INTERVAL 1 MINUTE), NOW(), NOW())`, id, userID)
	require.NoError(t, err)
	waitStatus(labs.StatusReady)
	var running int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM lab_jobs WHERE lab_id = ? AND status IN ('pending', 'running')`, id).Scan(&running))
	assert.Zero(t, running)

	// The sweep removes the files of users that no longer exist.
	box, _ := files.New(e.storage, nil, nil, false)
	_, err = box.Put(999999, []byte("orphan"))
	require.NoError(t, err)
	e.runner.Sweep(context.Background())
	_, err = os.Stat(filepath.Join(e.storage, files.Dir, "999999"))
	assert.True(t, os.IsNotExist(err))
	_ = strings.TrimSpace
}

// B-N6-06b: caps from the append-only usage log, the interpretation limit, SQL-side busy checks, the PDF header.
func TestCapsAndBusyChecks(t *testing.T) {
	e := setup(t, true)
	a, aTok := e.user(t, "09900006041", true, true)
	logCalls := func(n int, at string) {
		for range n {
			_, err := e.db.Exec(`INSERT INTO ai_usage_logs (user_id, feature, op, provider, model, ok, created_at)
				VALUES (?, 'lab_analysis', 'extract', 'fake', 'x', 1, `+at+`)`, a)
			require.NoError(t, err)
		}
	}

	// A PDF header far from offset 0 (a polyglot) is refused.
	r := e.upload(t, aTok, nil, part{"files", "x.pdf", []byte("<html>polyglot %PDF-1.4 RITME-FAKE:lab_panel")})
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, string(r.raw))

	r = e.upload(t, aTok, nil, part{"files", "sheet.pdf", pdf("lab_panel")})
	require.Equal(t, http.StatusAccepted, r.status, string(r.raw))
	id := labID(r)

	// Verify up to the interpretation limit, then 429.
	for i := range labs.MaxInterpretations {
		v := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/verify", id), aTok, nil)
		require.Equal(t, http.StatusAccepted, v.status, "verify %d: %s", i+1, v.raw)
	}
	v := e.json(t, http.MethodPost, fmt.Sprintf("/api/v1/labs/%d/verify", id), aTok, nil)
	require.Equal(t, http.StatusTooManyRequests, v.status, string(v.raw))
	assert.Equal(t, "lab_interpret_limit", v.body["error_code"])

	// Marker edits are refused in SQL while the lab is processing.
	mid := uint64(markers(r)[0].(map[string]any)["id"].(float64))
	_, err := e.db.Exec(`UPDATE lab_reports SET status = 'interpreting' WHERE id = ?`, id)
	require.NoError(t, err)
	up := e.json(t, http.MethodPut, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, mid), aTok, map[string]any{"name": "Hb", "value": 12})
	assert.Equal(t, http.StatusConflict, up.status)
	del := e.json(t, http.MethodDelete, fmt.Sprintf("/api/v1/labs/%d/markers/%d", id, mid), aTok, nil)
	assert.Equal(t, http.StatusConflict, del.status)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM lab_markers WHERE id = ?`, mid).Scan(&n))
	assert.Equal(t, 1, n)

	// The daily page cap counts the usage log: deleting labs does not reset it.
	require.Equal(t, http.StatusOK, e.json(t, http.MethodDelete, fmt.Sprintf("/api/v1/labs/%d", id), aTok, nil).status)
	before := usage(t, e, a)
	logCalls(labs.PagesPerDay-1, "NOW()")
	r = e.upload(t, aTok, nil, part{"files", "a.pdf", pdf("lab_panel")}, part{"files", "b.pdf", pdf("lab_panel")})
	require.Equal(t, http.StatusTooManyRequests, r.status, string(r.raw))
	assert.Equal(t, "lab_daily_limit", r.body["error_code"])
	assert.Equal(t, before, usage(t, e, a), "the refused upload keeps no use")
}

func TestBlankRefundsAreCapped(t *testing.T) {
	now := time.Now().In(time.FixedZone("IRST", 3*3600+1800))
	if now.Day() < 3 {
		t.Skip("needs two days of the month behind the daily window")
	}
	e := setup(t, true)
	a, aTok := e.user(t, "09900006051", true, true)
	for range labs.RefundableExtractCalls + 1 {
		_, err := e.db.Exec(`INSERT INTO ai_usage_logs (user_id, feature, op, provider, model, ok, created_at)
			VALUES (?, 'lab_analysis', 'extract', 'fake', 'x', 1, DATE_SUB(NOW(), INTERVAL 30 HOUR))`, a)
		require.NoError(t, err)
	}
	r := e.upload(t, aTok, nil, part{"files", "sheet.pdf", pdf("blank")})
	require.Equal(t, http.StatusAccepted, r.status, string(r.raw))
	assert.Equal(t, labs.CodeNothingRead, data(r)["error_code"])
	assert.Equal(t, 1, usage(t, e, a), "beyond the monthly allowance a blank sheet keeps its use")
	r = e.upload(t, aTok, nil, part{"files", "sheet.pdf", pdf("error")})
	assert.Equal(t, labs.CodeAIFailed, data(r)["error_code"])
	assert.Equal(t, 1, usage(t, e, a), "a provider failure is always refunded")
}

func TestSweepFinishesStuckJobsAndLabs(t *testing.T) {
	e := setup(t, false)
	a, _ := e.user(t, "09900006061", true, true)
	now := time.Now()
	insertLab := func(status string, quota bool) uint64 {
		q := "NULL"
		if quota {
			q = "NOW()"
			_, err := e.plus.Consume(context.Background(), a, plus.LabAI, now.In(time.FixedZone("IRST", 3*3600+1800)).Truncate(time.Second))
			require.NoError(t, err)
		}
		res, err := e.db.Exec(`INSERT INTO lab_reports (user_id, source, category, status, quota_at, interpret_count, created_at, updated_at)
			VALUES (?, 'upload', 'blood', ?, `+q+`, 1, DATE_SUB(NOW(), INTERVAL 2 HOUR), DATE_SUB(NOW(), INTERVAL 2 HOUR))`, a, status)
		require.NoError(t, err)
		id, _ := res.LastInsertId()
		return uint64(id) //nolint:gosec // test ids
	}
	status := func(id uint64) (string, sql.NullString) {
		var s string
		var code sql.NullString
		require.NoError(t, e.db.QueryRow(`SELECT status, error_code FROM lab_reports WHERE id = ?`, id).Scan(&s, &code))
		return s, code
	}

	// An extraction whose last attempt's worker died: failed, refunded.
	stuck := insertLab(labs.StatusExtracting, true)
	_, err := e.db.Exec(`INSERT INTO lab_jobs (lab_id, user_id, kind, locale, status, attempts, available_at, locked_until, created_at, updated_at)
		VALUES (?, ?, 'extract', 'fa', 'running', ?, NOW(), DATE_SUB(NOW(), INTERVAL 1 MINUTE), NOW(), NOW())`, stuck, a, labs.MaxAttempts)
	require.NoError(t, err)
	// A queued lab without any job, and an interpreting one without a job.
	orphan := insertLab(labs.StatusQueued, true)
	interp := insertLab(labs.StatusInterpreting, false)
	_, err = e.db.Exec(`INSERT INTO lab_markers (lab_id, user_id, name, value, source, created_at, updated_at) VALUES (?, ?, 'TSH', 2.1, 'extracted', NOW(), NOW())`, interp, a)
	require.NoError(t, err)
	require.Equal(t, 2, usage(t, e, a))

	e.runner.SweepJobs(context.Background())
	s, code := status(stuck)
	assert.Equal(t, labs.StatusFailed, s)
	assert.Equal(t, labs.CodeInternal, code.String)
	s, _ = status(orphan)
	assert.Equal(t, labs.StatusFailed, s)
	s, _ = status(interp)
	assert.Equal(t, labs.StatusReady, s, "an interpretation without a job gets the rules summary")
	assert.Equal(t, 0, usage(t, e, a), "both reserved uses refunded")
	var jobStatus string
	require.NoError(t, e.db.QueryRow(`SELECT status FROM lab_jobs WHERE lab_id = ?`, stuck).Scan(&jobStatus))
	assert.Equal(t, "failed", jobStatus)

	// The file sweep never removes a directory that still has lab_files rows.
	box, _ := files.New(e.storage, nil, nil, false)
	rel, err := box.Put(888888, []byte("x"))
	require.NoError(t, err)
	conn, err := e.db.Conn(context.Background()) // one connection: FOREIGN_KEY_CHECKS is per session
	require.NoError(t, err)
	_, err = conn.ExecContext(context.Background(), `SET FOREIGN_KEY_CHECKS = 0`)
	require.NoError(t, err)
	_, err = conn.ExecContext(context.Background(), `INSERT INTO lab_files (lab_id, user_id, page, mime, size_bytes, path, created_at) VALUES (?, 888888, 1, 'application/pdf', 1, ?, NOW())`, stuck, rel)
	require.NoError(t, err)
	_, _ = conn.ExecContext(context.Background(), `SET FOREIGN_KEY_CHECKS = 1`)
	require.NoError(t, conn.Close())
	e.runner.Sweep(context.Background())
	_, err = os.Stat(filepath.Join(e.storage, rel))
	assert.NoError(t, err, "kept: rows still point at it")
}

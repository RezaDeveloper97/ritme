package extract_test

// CB-REC-02 end to end against MariaDB with the fake AI provider: the AI gate (401, Plus 402 for free users who then
// fill the fields in by hand, consent 403), extraction per kind (sync and the async worker, lease recovery, exhausted
// jobs), refunds on failure, review with corrections, the pregnancy dating offer (never silent; explicit confirm,
// dismiss) and user isolation.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/healthrecord/extract"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffe07"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db       *sql.DB
	app      *fiber.App
	iss      *passport.Issuer
	files    *files.Service
	runner   *extract.Runner
	plus     *plus.Service
	consents *consent.Service
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
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))

	plusSvc := plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet)
	gate := plus.NewGate(plusSvc, clock.Real{})
	fake := ai.NewFake()
	client := ai.NewClientWith(ai.Options{Provider: config.AIProviderFake, Extractor: fake})
	consents := consent.NewService(db)
	g := access.NewGuard(access.Options{Client: client, Consents: consents, Gate: gate, Plus: plusSvc, Logger: quiet})

	vault, err := files.NewVault(t.TempDir(), files.DevKey("ritme-file-dev-key"), nil, false)
	require.NoError(t, err)
	fs := files.NewService(files.Options{DB: db, Vault: vault, BaseURL: "https://api.test"})
	docs := healthrecord.NewDocuments(db, fs, nil).OnDeletePending(extract.RefundDeleted(plusSvc, clock.Real{}, quiet))
	svc := extract.NewService(extract.Options{DB: db, Docs: docs, Files: fs, AI: client, Consents: consents, Plus: plusSvc,
		Pregnancy: pregstore.New(db), Logger: quiet})
	runner := extract.NewRunner(svc, extract.RunnerOptions{Sync: sync, Poll: 50 * time.Millisecond})
	t.Cleanup(runner.Shutdown)
	h := extract.NewHandlers(svc, clock.Real{})
	dh := healthrecord.NewDocumentHandlers(docs, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	const p = "/api/v1/health-record/documents"
	app.Post(p, locale, guard, dh.StoreDocument)
	app.Get(p+"/:id", locale, guard, dh.ShowDocument)
	app.Put(p+"/:id", locale, guard, dh.UpdateDocument)
	app.Delete(p+"/:id", locale, guard, dh.DestroyDocument)
	start := append([]any{guard}, g.Chain(ai.FeatureDocExtract)...)
	app.Post(p+"/:id/extract", locale, append(start, h.Extract)...)
	app.Post(p+"/:id/review", locale, guard, h.Review)
	app.Get(p+"/:id/dating", locale, guard, h.Dating)
	app.Post(p+"/:id/dating", locale, guard, h.ApplyDating)
	app.Delete(p+"/:id/dating", locale, guard, h.DismissDating)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), files: fs, runner: runner,
		plus: plusSvc, consents: consents}
}

// user creates a user; trial = Plus trial started, consented = ai_documents accepted.
func (e *env) user(t *testing.T, mobile string, trial, consented bool) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Test', ?, NOW(), NOW())`, mobile)
	require.NoError(t, err)
	id64, err := res.LastInsertId()
	require.NoError(t, err)
	id := uint64(id64) //nolint:gosec // test ids
	now := time.Now()
	if trial {
		require.NoError(t, e.plus.StartTrial(context.Background(), id, now))
	}
	if consented {
		require.NoError(t, e.consents.Accept(context.Background(), id, consent.AIDocuments, 1, now))
	}
	tok, err := e.iss.Issue(context.Background(), id, now)
	require.NoError(t, err)
	return id, tok.AccessToken
}

func (e *env) pregnant(t *testing.T, uid uint64, lmp string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, is_locked, age_source, confidence_level,
		lmp_date, uncertainty_days, onboarding_completed, created_at, updated_at)
		VALUES (?, 1, 0, 0, 'lmp', 'medium', ?, 3, 1, NOW(), NOW())`, uid, lmp)
	require.NoError(t, err)
}

type resp struct {
	status int
	body   map[string]any
	raw    string
}

func (r resp) data() map[string]any { m, _ := r.body["data"].(map[string]any); return m }

func (e *env) do(t *testing.T, method, path, token string, body any) resp {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	raw, _ := io.ReadAll(res.Body)
	out := resp{status: res.StatusCode, raw: string(raw)}
	_ = json.Unmarshal(raw, &out.body)
	return out
}

func pdf(key string) []byte { return []byte("%PDF-1.4\n% RITME-FAKE:" + key + "\n%%EOF") }

// document creates a document of kind with one file carrying the fake fixture key (no file when key is "").
func (e *env) document(t *testing.T, uid uint64, token, kind, key string) string {
	t.Helper()
	body := map[string]any{"kind": kind}
	if key != "" {
		f, err := e.files.Put(context.Background(), uid, files.PurposeRecordDocument, pdf(key), time.Now())
		require.NoError(t, err)
		body["file_ids"] = []uint64{f.ID}
	}
	r := e.do(t, http.MethodPost, docsPath, token, body)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	return fmt.Sprint(uint64(r.data()["id"].(float64)))
}

const docsPath = "/api/v1/health-record/documents"

func (e *env) usage(t *testing.T, uid uint64) int {
	t.Helper()
	var n sql.NullInt64
	require.NoError(t, e.db.QueryRow(`SELECT SUM(used) FROM plus_usage_counters WHERE user_id = ? AND feature = 'plus.doc_ai'`, uid).Scan(&n))
	return int(n.Int64)
}

func extracted(r resp) map[string]any { m, _ := r.data()["extracted"].(map[string]any); return m }

func TestExtract_Unauthenticated(t *testing.T) {
	e := setup(t, true)
	for _, c := range []struct{ method, path string }{
		{http.MethodPost, docsPath + "/1/extract"}, {http.MethodPost, docsPath + "/1/review"},
		{http.MethodGet, docsPath + "/1/dating"}, {http.MethodPost, docsPath + "/1/dating"}, {http.MethodDelete, docsPath + "/1/dating"},
	} {
		r := e.do(t, c.method, c.path, "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.Equal(t, "unauthenticated", r.body["error_code"])
	}
}

// Free users get 402 before anything is read and fill the document in by hand; without the consent → 403.
func TestExtract_FreeAndConsentGates(t *testing.T) {
	e := setup(t, true)
	free, freeTok := e.user(t, "09120000001", false, true)
	id := e.document(t, free, freeTok, healthrecord.KindVisit, "visit")
	r := e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", freeTok, nil)
	require.Equal(t, http.StatusPaymentRequired, r.status, r.raw)
	assert.Equal(t, "plus.doc_ai", r.body["feature"])
	r = e.do(t, http.MethodPut, docsPath+"/"+id, freeTok, map[string]any{"date": "2026-09-15", "centre": "Clinic", "doctor": "Dr. A"})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, healthrecord.ReviewManual, r.data()["review_state"])
	assert.Nil(t, r.data()["extracted"])
	assert.Equal(t, 0, e.usage(t, free))

	uid, tok := e.user(t, "09120000002", true, false)
	id = e.document(t, uid, tok, healthrecord.KindVisit, "visit")
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil)
	require.Equal(t, http.StatusForbidden, r.status, r.raw)
	assert.Equal(t, "consent_required", r.body["error_code"])
	assert.Equal(t, "ai_documents", r.body["consent"])
	assert.Equal(t, 0, e.usage(t, uid))
}

func TestExtract_PerKindAndReview(t *testing.T) {
	e := setup(t, true)
	uid, tok := e.user(t, "09120000003", true, true)
	cases := []struct {
		kind, key string
		want      map[string]any
	}{
		{healthrecord.KindImaging, "imaging", map[string]any{"kind": "ultrasound", "ga_weeks": 12.0, "edd": "2027-04-05"}},
		{healthrecord.KindVisit, "visit", map[string]any{"specialty": "زنان و زایمان", "next_visit": "2026-12-15"}},
		{healthrecord.KindPrescription, "prescription", map[string]any{"doctor": "دکتر نمونه"}},
		{healthrecord.KindHospital, "hospital", map[string]any{"ended_on": "2026-08-04", "procedures": "لاپاراسکوپی"}},
		{healthrecord.KindOther, "other_document", map[string]any{"title": "گواهی پزشکی"}},
	}
	for i, c := range cases {
		id := e.document(t, uid, tok, c.kind, c.key)
		r := e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil)
		require.Equal(t, http.StatusAccepted, r.status, r.raw)
		assert.Equal(t, healthrecord.ReviewNeedsReview, r.data()["review_state"], c.kind)
		ex := extracted(r)
		require.NotNil(t, ex, c.kind)
		assert.NotContains(t, ex, "job", "the job bookkeeping is never shown")
		assert.Equal(t, extract.StatusDone, ex["status"])
		fields, _ := ex["fields"].(map[string]any)
		for k, v := range c.want {
			f, _ := fields[k].(map[string]any)
			assert.Equal(t, v, f["value"], c.kind+" "+k)
			assert.NotZero(t, f["confidence"])
		}
		if c.kind == healthrecord.KindPrescription {
			items, _ := ex["items"].([]any)
			assert.Len(t, items, 2)
		}
		assert.Equal(t, i+1, e.usage(t, uid), c.kind)
	}

	// Review a visit with a correction and a cleared value; the columns follow.
	id := e.document(t, uid, tok, healthrecord.KindVisit, "visit")
	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil).status)
	r := e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, map[string]any{"fields": map[string]any{"ga_weeks": 3}})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw) // not a visit field
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, map[string]any{"fields": map[string]any{"date": "2030-01-01"}})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, map[string]any{"items": []any{}})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw) // a visit has no items
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, map[string]any{"fields": map[string]any{"doctor": "Dr. B", "diagnosis": nil}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	doc, _ := r.data()["document"].(map[string]any)
	assert.Equal(t, healthrecord.ReviewConfirmed, doc["review_state"])
	assert.Equal(t, "Dr. B", doc["doctor"])
	assert.Equal(t, "2026-09-15", doc["date"])
	assert.Equal(t, "مطب نمونه", doc["centre"])
	reviewed, _ := doc["extracted"].(map[string]any)["reviewed"].(map[string]any)
	assert.Equal(t, "Dr. B", reviewed["doctor"])
	assert.NotContains(t, reviewed, "diagnosis")
	dating, _ := r.data()["dating"].(map[string]any)
	assert.Equal(t, "unavailable", dating["state"])
	assert.Equal(t, extract.ReasonNotImaging, dating["reason"])

	// Confirmed: no second extraction, no second review.
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil)
	assert.Equal(t, http.StatusConflict, r.status, r.raw)
	assert.Equal(t, extract.CodeDocumentConfirmed, r.body["error_code"])
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, nil)
	assert.Equal(t, http.StatusConflict, r.status, r.raw)
	assert.Equal(t, extract.CodeNotInReview, r.body["error_code"])
	assert.Equal(t, 6, e.usage(t, uid), "rejected requests gave their reserved use back")

	// Prescription rows replaced on review.
	id = e.document(t, uid, tok, healthrecord.KindPrescription, "prescription")
	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil).status)
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, map[string]any{"items": []any{map[string]any{"medicine": "Folic acid", "dose": "1"}}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	rows, _ := r.data()["document"].(map[string]any)["extracted"].(map[string]any)["reviewed_items"].([]any)
	require.Len(t, rows, 1)
	assert.Equal(t, map[string]any{"medicine": "Folic acid", "dose": "1"}, rows[0])
}

// A document without files is 422 and a failed extraction (provider error, nothing read) refunds the reserved use.
func TestExtract_FailuresRefund(t *testing.T) {
	e := setup(t, true)
	uid, tok := e.user(t, "09120000004", true, true)
	id := e.document(t, uid, tok, healthrecord.KindVisit, "")
	r := e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "file_ids")
	assert.Equal(t, 0, e.usage(t, uid))

	for key, code := range map[string]string{"error": extract.CodeAIFailed, "blank": extract.CodeNothingRead} {
		id := e.document(t, uid, tok, healthrecord.KindVisit, key)
		r := e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil)
		require.Equal(t, http.StatusAccepted, r.status, r.raw)
		assert.Equal(t, healthrecord.ReviewFailed, r.data()["review_state"], key)
		assert.Equal(t, code, extracted(r)["error_code"], key)
		assert.Equal(t, 0, e.usage(t, uid), key+": refunded")
		// A failed document can be sent again (and filled in by hand meanwhile).
		r = e.do(t, http.MethodPut, docsPath+"/"+id, tok, map[string]any{"centre": "Clinic"})
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
}

func TestExtract_UserIsolation(t *testing.T) {
	e := setup(t, true)
	a, aTok := e.user(t, "09120000005", true, true)
	b, bTok := e.user(t, "09120000006", true, true)
	e.pregnant(t, b, "2026-06-25")
	id := e.document(t, a, aTok, healthrecord.KindImaging, "imaging")

	r := e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", bTok, nil)
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	assert.Equal(t, healthrecord.ErrorCodeDocumentNotFound, r.body["error_code"])
	assert.Equal(t, 0, e.usage(t, b), "the 404 gave B's reserved use back")

	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", aTok, nil).status)
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/review", nil}, {http.MethodGet, "/dating", nil},
		{http.MethodPost, "/dating", map[string]any{"confirm": true}}, {http.MethodDelete, "/dating", nil},
	} {
		r := e.do(t, c.method, docsPath+"/"+id+c.path, bTok, c.body)
		assert.Equal(t, http.StatusNotFound, r.status, c.path+" "+r.raw)
	}
	r = e.do(t, http.MethodGet, docsPath+"/"+id, aTok, nil)
	assert.Equal(t, healthrecord.ReviewNeedsReview, r.data()["review_state"], "B changed nothing")
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pregnancy_profiles WHERE user_id = ? AND age_source = 'lmp'`, b).Scan(&n))
	assert.Equal(t, 1, n)
}

func profile(t *testing.T, e *env, uid uint64) (src, us string, weeks, days sql.NullInt32, due string) {
	t.Helper()
	var usDate, dueDate sql.NullString
	require.NoError(t, e.db.QueryRow(`SELECT age_source, DATE_FORMAT(ultrasound_date, '%Y-%m-%d'), ultrasound_weeks, ultrasound_days, DATE_FORMAT(estimated_due_date, '%Y-%m-%d')
		FROM pregnancy_profiles WHERE user_id = ?`, uid).Scan(&src, &usDate, &weeks, &days, &dueDate))
	return src, usDate.String, weeks, days, dueDate.String
}

// The dating offer: nothing changes until the user confirms; then the pregnancy is dated from the scan and the
// document is linked to it. Dismissing leaves the pregnancy alone.
func TestExtract_PregnancyDating(t *testing.T) {
	e := setup(t, true)
	uid, tok := e.user(t, "09120000007", true, true)
	e.pregnant(t, uid, "2026-06-25")
	id := e.document(t, uid, tok, healthrecord.KindImaging, "imaging")
	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil).status)

	r := e.do(t, http.MethodGet, docsPath+"/"+id+"/dating", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "unavailable", r.data()["state"])
	assert.Equal(t, extract.ReasonNotConfirmed, r.data()["reason"])
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/dating", tok, map[string]any{"confirm": true})
	require.Equal(t, http.StatusConflict, r.status, r.raw)
	assert.Equal(t, extract.CodeDatingUnavailable, r.body["error_code"])

	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/review", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	offer, _ := r.data()["dating"].(map[string]any)
	assert.Equal(t, extract.OfferOffered, offer["state"])
	assert.Equal(t, map[string]any{"ga_weeks": 12.0, "ga_days": 3.0, "due_date": "2027-03-30",
		"weeks_today": offer["proposed"].(map[string]any)["weeks_today"], "days_today": offer["proposed"].(map[string]any)["days_today"]},
		offer["proposed"])
	src, _, _, _, _ := profile(t, e, uid)
	assert.Equal(t, "lmp", src, "confirming the document never re-dates the pregnancy by itself")

	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/dating", tok, map[string]any{})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/dating", tok, map[string]any{"confirm": true})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, extract.OfferApplied, r.data()["state"])
	assert.Equal(t, map[string]any{"source": "ultrasound", "due_date": "2027-03-30"}, r.data()["current"])
	src, us, w, d, due := profile(t, e, uid)
	assert.Equal(t, "ultrasound", src)
	assert.Equal(t, "2026-09-18", us)
	assert.Equal(t, int32(12), w.Int32)
	assert.Equal(t, int32(3), d.Int32)
	assert.Equal(t, "2027-03-30", due)

	r = e.do(t, http.MethodGet, docsPath+"/"+id, tok, nil)
	used, _ := r.data()["where_used"].([]any)
	require.Len(t, used, 1)
	assert.Equal(t, "pregnancy", used[0].(map[string]any)["type"])
	assert.Equal(t, "applied", used[0].(map[string]any)["state"])
	assert.Equal(t, http.StatusConflict, e.do(t, http.MethodPost, docsPath+"/"+id+"/dating", tok, map[string]any{"confirm": true}).status)

	// A second scan with the same dating: nothing to offer. A different one: dismissed, pregnancy untouched.
	id2 := e.document(t, uid, tok, healthrecord.KindImaging, "imaging")
	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id2+"/extract", tok, nil).status)
	r = e.do(t, http.MethodPost, docsPath+"/"+id2+"/review", tok, nil)
	assert.Equal(t, extract.ReasonSameDating, r.data()["dating"].(map[string]any)["reason"])
	id3 := e.document(t, uid, tok, healthrecord.KindImaging, "imaging_edd")
	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id3+"/extract", tok, nil).status)
	r = e.do(t, http.MethodPost, docsPath+"/"+id3+"/review", tok, map[string]any{"fields": map[string]any{"edd": "2027-04-02"}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, extract.OfferOffered, r.data()["dating"].(map[string]any)["state"])
	r = e.do(t, http.MethodDelete, docsPath+"/"+id3+"/dating", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, extract.OfferDismissed, r.data()["state"])
	_, _, _, _, due = profile(t, e, uid)
	assert.Equal(t, "2027-03-30", due)
	assert.Equal(t, extract.OfferDismissed, e.do(t, http.MethodGet, docsPath+"/"+id3+"/dating", tok, nil).data()["state"])
}

func waitState(t *testing.T, e *env, tok, id, want string) resp {
	t.Helper()
	var r resp
	require.Eventually(t, func() bool {
		r = e.do(t, http.MethodGet, docsPath+"/"+id, tok, nil)
		return r.data()["review_state"] == want
	}, 10*time.Second, 50*time.Millisecond, "want %s", want)
	return r
}

// The async workers: a queued job is picked up, a dead worker's job is reclaimed after its lease, a job that used every
// attempt fails and refunds, a consent withdrawn before the job runs fails it (refund).
func TestExtract_AsyncRunner(t *testing.T) {
	e := setup(t, false)
	uid, tok := e.user(t, "09120000008", true, true)
	id := e.document(t, uid, tok, healthrecord.KindHospital, "hospital")
	r := e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil)
	require.Equal(t, http.StatusAccepted, r.status, r.raw)
	assert.Equal(t, healthrecord.ReviewPending, r.data()["review_state"])
	assert.Equal(t, extract.StatusPending, extracted(r)["status"])
	r = e.do(t, http.MethodPost, docsPath+"/"+id+"/extract", tok, nil)
	assert.Equal(t, http.StatusConflict, r.status, r.raw)

	assert.Equal(t, 1, e.usage(t, uid), "the 409 gave its reserved use back")

	// Consent withdrawn while queued (another user).
	other, otherTok := e.user(t, "09120000009", true, true)
	id2 := e.document(t, other, otherTok, healthrecord.KindVisit, "visit")
	require.Equal(t, http.StatusAccepted, e.do(t, http.MethodPost, docsPath+"/"+id2+"/extract", otherTok, nil).status)
	require.NoError(t, e.consents.Withdraw(context.Background(), other, consent.AIDocuments, time.Now()))
	assert.Equal(t, 1, e.usage(t, other))

	// A dead worker's job (lease expired) and an exhausted one.
	past := time.Now().Add(-time.Hour).Unix()
	job := func(attempts int) string {
		return fmt.Sprintf(`{"schema":"visit","status":"pending","error_code":null,"fields":{},"items":[],"reviewed":null,"reviewed_items":null,
			"requested_at":"2026-09-23T10:00:00+03:30","extracted_at":null,"reviewed_at":null,"dating":null,
			"job":{"state":"running","attempts":%d,"available_at":%d,"locked_until":%d,"locale":"fa","reserved_at":""}}`, attempts, past, past)
	}
	dead := e.document(t, uid, tok, healthrecord.KindVisit, "visit")
	_, err := e.db.Exec(`UPDATE record_documents SET review_state = 'pending', extracted = ? WHERE id = ?`, job(1), dead)
	require.NoError(t, err)
	exhausted := e.document(t, uid, tok, healthrecord.KindVisit, "visit")
	_, err = e.db.Exec(`UPDATE record_documents SET review_state = 'pending', extracted = ? WHERE id = ?`, job(extract.MaxAttempts), exhausted)
	require.NoError(t, err)

	require.NoError(t, e.runner.Start())
	r = waitState(t, e, tok, id, healthrecord.ReviewNeedsReview)
	assert.Equal(t, "بیمارستان نمونه", extracted(r)["fields"].(map[string]any)["centre"].(map[string]any)["value"])
	r = waitState(t, e, otherTok, id2, healthrecord.ReviewFailed)
	assert.Equal(t, extract.CodeConsent, extracted(r)["error_code"])
	assert.Equal(t, 0, e.usage(t, other), "refunded")
	waitState(t, e, tok, dead, healthrecord.ReviewNeedsReview)
	r = waitState(t, e, tok, exhausted, healthrecord.ReviewFailed)
	assert.Equal(t, extract.CodeInternal, extracted(r)["error_code"])
	assert.Equal(t, 1, e.usage(t, uid), "only the successful extraction keeps its use")
}

package plus_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30"
	plan1m   = 1 // seeded: 990,000 rials
	plan3m   = 2 // seeded: 2,370,000 rials, highlighted
	plan6m   = 3 // seeded: 3,900,000 rials
)

var (
	quiet  = slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg    = config.Plus{VATRateBps: 1000, TrialDays: 7, InvoiceTTL: 30 * time.Minute, CallbackURL: "http://web.test/plus/return"}
	tNow   = mustTime(now)
	tehran = civildate.Tehran
)

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t.In(civildate.Tehran)
}

// tamperGateway is the fake with a provider that misbehaves: it reports a different settled amount and/or a
// fixed bank reference for every payment.
type tamperGateway struct {
	plus.FakeGateway
	amountDelta int64
	refID       string
	fail        bool
}

func (g tamperGateway) Verify(ctx context.Context, req plus.VerifyRequest) (plus.VerifyResult, error) {
	if g.fail {
		return plus.VerifyResult{}, errors.New("provider timeout")
	}
	res, err := g.FakeGateway.Verify(ctx, req)
	if res.Paid {
		res.AmountRials = uint64(int64(res.AmountRials) + g.amountDelta) //nolint:gosec // test
		if g.refID != "" {
			res.RefID = g.refID
		}
	}
	return res, err
}

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
	svc *plus.Service
}

func setup(t *testing.T, gw plus.Gateway) *env {
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
	svc := plus.NewService(db, cfg, gw, quiet)
	h := plus.NewHandlers(svc, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/plus/plans", locale, h.Plans)
	app.Get("/api/v1/plus/status", locale, guard, h.Status)
	app.Get("/api/v1/plus/usage", locale, guard, h.Usage)
	app.Get("/api/v1/plus/history", locale, guard, h.History)
	app.Post("/api/v1/plus/trial/start", locale, guard, h.StartTrial)
	app.Post("/api/v1/plus/checkout", locale, guard, h.Checkout)
	app.Post("/api/v1/plus/verify", locale, guard, h.Verify)
	app.Post("/api/v1/plus/cancel", locale, guard, h.Cancel)
	app.Post("/api/v1/plus/restore", locale, guard, h.Restore)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), svc: svc}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (r response) obj(path ...string) map[string]any {
	m := r.data()
	for _, p := range path {
		m, _ = m[p].(map[string]any)
	}
	return m
}

func (e *env) doAt(t *testing.T, at, method, path, token, lang, body string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	req.Header.Set(clock.Header, at)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func (e *env) do(t *testing.T, method, path, token, lang, body string) response {
	t.Helper()
	return e.doAt(t, now, method, path, token, lang, body)
}

func (e *env) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

func (e *env) discount(t *testing.T, code, kind string, value int, maxRedemptions, perUser any, expiresAt any) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO plus_discount_codes (code, kind, value, max_redemptions, per_user_limit, expires_at, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, code, kind, value, maxRedemptions, perUser, expiresAt)
	require.NoError(t, err)
}

// checkout creates a paid-for-later invoice and returns its reference and authority.
func (e *env) checkout(t *testing.T, at, token, body string) (string, string, response) {
	t.Helper()
	r := e.doAt(t, at, http.MethodPost, "/api/v1/plus/checkout", token, "en", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	ref, _ := r.obj("invoice")["reference"].(string)
	auth, _ := r.obj("payment")["authority"].(string)
	return ref, auth, r
}

func verifyBody(ref, authority, status string) string {
	b, _ := json.Marshal(map[string]string{"reference": ref, "authority": authority, "status": status})
	return string(b)
}

func entitlement(t *testing.T, list any, key string) map[string]any {
	t.Helper()
	items, _ := list.([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["key"] == key {
			return m
		}
	}
	t.Fatalf("no entitlement %s in %v", key, list)
	return nil
}

func TestUnauthenticated(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/plus/status"},
		{http.MethodGet, "/api/v1/plus/usage"},
		{http.MethodGet, "/api/v1/plus/history"},
		{http.MethodPost, "/api/v1/plus/trial/start"},
		{http.MethodPost, "/api/v1/plus/checkout"},
		{http.MethodPost, "/api/v1/plus/verify"},
		{http.MethodPost, "/api/v1/plus/cancel"},
		{http.MethodPost, "/api/v1/plus/restore"},
	} {
		r := e.do(t, c.method, c.path, "", "", `{}`)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw, c.path)
	}
}

func TestPlans_PublicWithDerivedFigures(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	r := e.do(t, http.MethodGet, "/api/v1/plus/plans", "", "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "IRR", r.data()["currency"])
	assert.EqualValues(t, 1000, r.data()["vat_rate_bps"])
	assert.EqualValues(t, 7, r.data()["trial_days"])
	plans, _ := r.data()["plans"].([]any)
	require.Len(t, plans, 3)
	p3 := plans[1].(map[string]any)
	assert.Equal(t, "plus_3m", p3["code"])
	assert.Equal(t, "۳ ماهه", p3["title"])
	assert.Equal(t, "محبوب\u200cترین", p3["badge"])
	assert.EqualValues(t, 2370000, p3["price"])
	assert.EqualValues(t, 790000, p3["monthly_price"])
	assert.EqualValues(t, 20, p3["savings_percent"])
	assert.Equal(t, true, p3["is_highlighted"])
	assert.Nil(t, plans[0].(map[string]any)["badge"])
	assert.EqualValues(t, 34, plans[2].(map[string]any)["savings_percent"])

	// Inactive plans disappear; English titles.
	_, err := e.db.Exec(`UPDATE plus_plans SET is_active = 0 WHERE id = ?`, plan6m)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, "/api/v1/plus/plans", "", "en", "")
	plans, _ = r.data()["plans"].([]any)
	require.Len(t, plans, 2)
	assert.Equal(t, "Most popular", plans[1].(map[string]any)["badge"])
}

func TestTrial_StartOnceAndExpire(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	_, tok := e.user(t, "09120000801")

	r := e.do(t, http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "free", r.data()["tier"])
	assert.Equal(t, false, r.data()["is_plus"])
	assert.Equal(t, true, r.data()["trial_available"])
	assert.Nil(t, r.data()["trial"])
	assert.Equal(t, false, entitlement(t, r.data()["entitlements"], "plus.deep_analysis")["allowed"])

	r = e.do(t, http.MethodPost, "/api/v1/plus/trial/start", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Your free Ritme Plus trial has started", r.body["message"])
	assert.Equal(t, "trial", r.data()["tier"])
	assert.Equal(t, true, r.data()["is_plus"])
	assert.Equal(t, false, r.data()["trial_available"])
	assert.Equal(t, map[string]any{"started_at": "2026-09-23T10:00:00+03:30", "ends_at": "2026-09-30T10:00:00+03:30", "is_active": true, "days_left": float64(7)}, r.obj("trial"))
	lab := entitlement(t, r.data()["entitlements"], "plus.lab_ai")
	assert.Equal(t, map[string]any{"key": "plus.lab_ai", "allowed": true, "unlimited": false, "limit": float64(10), "used": float64(0), "remaining": float64(10)}, lab)

	r = e.do(t, http.MethodPost, "/api/v1/plus/trial/start", tok, "fa", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.JSONEq(t, `{"success":false,"message":"از دوره رایگان قبلاً استفاده کرده\u200cای.","error_code":"trial_used"}`, r.raw)

	// Six days later: one day left; at the end instant it is over.
	r = e.doAt(t, "2026-09-29T12:00:00+03:30", http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	assert.Equal(t, "trial", r.data()["tier"])
	assert.EqualValues(t, 1, r.obj("trial")["days_left"])
	r = e.doAt(t, "2026-09-30T10:00:00+03:30", http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	assert.Equal(t, "free", r.data()["tier"])
	assert.Equal(t, false, r.obj("trial")["is_active"])
	assert.Equal(t, false, r.data()["trial_available"])
	r = e.doAt(t, "2026-11-01T10:00:00+03:30", http.MethodPost, "/api/v1/plus/trial/start", tok, "en", "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "never a second trial")
}

func TestTrial_RaceStartsExactlyOne(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	uid, _ := e.user(t, "09120000802")
	var wg sync.WaitGroup
	errs := make([]error, 12)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = e.svc.StartTrial(context.Background(), uid, tNow)
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
			continue
		}
		assert.ErrorIs(t, err, plus.ErrTrialUsed)
	}
	assert.Equal(t, 1, ok)
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_trials WHERE user_id = ?`, uid))
}

func TestCheckoutVerify_HappyPathAndIdempotentReplay(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	uid, tok := e.user(t, "09120000803")
	e.discount(t, "RITME10", plus.DiscountPercent, 10, nil, 1, nil)

	// Preview: nothing is written.
	r := e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":2,"discount_code":" ritme10 ","preview":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"plan":{"id":2,"code":"plus_3m","title":"3 months","duration_months":3},"currency":"IRR","subtotal":2370000,"discount":237000,"discount_code":"RITME10","vat_rate_bps":1000,"vat":213300,"total":2346300}`,
		mustJSON(t, r.data()["quote"]))
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_invoices`))

	// Amount tampering on the way in: client-sent amounts are ignored, the server prices the plan.
	ref, authority, r := e.checkout(t, now, tok, `{"plan_id":2,"discount_code":"RITME10","total":1,"amount":1,"subtotal_rials":1}`)
	assert.Equal(t, "Invoice created", r.body["message"])
	inv := r.obj("invoice")
	assert.Regexp(t, `^RP[A-Z2-7]{16}$`, ref)
	assert.Equal(t, "pending", inv["status"])
	assert.EqualValues(t, 2346300, inv["total"])
	assert.Equal(t, "2026-09-23T10:30:00+03:30", inv["expires_at"])
	assert.Equal(t, "fake", r.obj("payment")["gateway"])
	assert.Equal(t, "FAKE-"+ref, authority)
	assert.Contains(t, r.obj("payment")["redirect_url"], "http://web.test/plus/return?")

	r = e.doAt(t, "2026-09-23T10:05:00+03:30", http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Payment confirmed. Ritme Plus is active", r.body["message"])
	assert.Equal(t, false, r.data()["already_verified"])
	inv = r.obj("invoice")
	assert.Equal(t, "paid", inv["status"])
	assert.Equal(t, "2026-09-23T10:05:00+03:30", inv["paid_at"])
	assert.Equal(t, map[string]any{"ref_id": "FAKE-REF-" + ref, "card_pan": nil, "amount": float64(2346300), "paid_at": "2026-09-23T10:05:00+03:30"}, inv["receipt"])
	sub := r.obj("status", "subscription")
	assert.Equal(t, "plus", r.obj("status")["tier"])
	assert.Equal(t, "2026-12-23T10:05:00+03:30", sub["ends_at"])
	assert.EqualValues(t, 91, sub["days_left"])
	assert.Equal(t, true, sub["auto_renew"])
	assert.Equal(t, "plus_3m", sub["plan"].(map[string]any)["code"])

	// Replayed verify (same body, later): 200, same invoice, no second period or receipt.
	r = e.doAt(t, "2026-09-24T10:00:00+03:30", http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.data()["already_verified"])
	assert.Equal(t, "This payment was already confirmed", r.body["message"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_receipts WHERE user_id = ?`, uid))

	// The code was used once: a second checkout with it is refused (per-user limit 1).
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":1,"discount_code":"RITME10"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.JSONEq(t, `{"success":false,"message":"Validation failed","errors":{"discount_code":["You have already used this discount code."]},"error_code":"discount_used"}`, r.raw)

	// History lists the paid invoice with its receipt.
	r = e.do(t, http.MethodGet, "/api/v1/plus/history", tok, "en", "")
	list, _ := r.body["data"].([]any)
	require.Len(t, list, 1)
	assert.Equal(t, ref, list[0].(map[string]any)["reference"])
	assert.NotNil(t, list[0].(map[string]any)["receipt"])

	// A trial is no longer offered after a subscription.
	r = e.do(t, http.MethodPost, "/api/v1/plus/trial/start", tok, "en", "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Equal(t, "trial_unavailable", r.body["error_code"])
}

func TestVerify_ConcurrentReplaysActivateOnce(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	uid, tok := e.user(t, "09120000804")
	ref, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	var wg sync.WaitGroup
	results := make([]error, 8)
	already := make([]bool, 8)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := e.svc.Verify(context.Background(), uid, ref, authority, map[string]string{"status": "OK"}, tNow)
			results[i], already[i] = err, res.AlreadyPaid
		}()
	}
	wg.Wait()
	fresh := 0
	for i, err := range results {
		require.NoError(t, err)
		if !already[i] {
			fresh++
		}
	}
	assert.Equal(t, 1, fresh)
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
}

func TestVerify_IDORAndAuthorityChecks(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	_, tokA := e.user(t, "09120000805")
	uidB, tokB := e.user(t, "09120000806")
	refA, authA, _ := e.checkout(t, now, tokA, `{"plan_id":1}`)
	refB, authB, _ := e.checkout(t, now, tokB, `{"plan_id":1}`)

	// B cannot verify (or even see) A's invoice.
	r := e.do(t, http.MethodPost, "/api/v1/plus/verify", tokB, "en", verifyBody(refA, authA, "OK"))
	require.Equal(t, http.StatusNotFound, r.status, r.raw)
	assert.JSONEq(t, `{"success":false,"message":"Invoice not found.","error_code":"invoice_not_found"}`, r.raw)
	// B cannot pay its own invoice with A's payment.
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tokB, "en", verifyBody(refB, authA, "OK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "authority_mismatch", r.body["error_code"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions`))

	// History is per user.
	r = e.do(t, http.MethodGet, "/api/v1/plus/history", tokB, "en", "")
	list, _ := r.body["data"].([]any)
	require.Len(t, list, 1)
	assert.Equal(t, refB, list[0].(map[string]any)["reference"])

	// A pays; B is still free.
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tokA, "en", verifyBody(refA, authA, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/plus/status", tokB, "en", "")
	assert.Equal(t, "free", r.data()["tier"])
	_ = uidB
	_ = authB
}

func TestVerify_DeclinedThenClosed(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	_, tok := e.user(t, "09120000807")
	ref, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	r := e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "fa", verifyBody(ref, authority, "NOK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "payment_failed", r.body["error_code"])
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "invoice_closed", r.body["error_code"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions`))
}

func TestVerify_SettledAmountTampering(t *testing.T) {
	e := setup(t, tamperGateway{amountDelta: -1000})
	uid, tok := e.user(t, "09120000808")
	ref, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	r := e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "amount_mismatch", r.body["error_code"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_receipts`))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_invoices WHERE status = 'failed'`))
}

func TestVerify_ReplayedBankReferenceRejected(t *testing.T) {
	e := setup(t, tamperGateway{refID: "BANK-123"})
	uid, tok := e.user(t, "09120000809")
	ref1, auth1, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	r := e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref1, auth1, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	ref2, auth2, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref2, auth2, "OK"))
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "payment_failed", r.body["error_code"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
}

func TestVerify_ProviderErrorKeepsInvoicePending(t *testing.T) {
	e := setup(t, tamperGateway{fail: true})
	_, tok := e.user(t, "09120000810")
	ref, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	r := e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusServiceUnavailable, r.status, r.raw)
	assert.Equal(t, "payment_unavailable", r.body["error_code"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_invoices WHERE status = 'pending'`))
}

func TestCheckout_NoGatewayIs503(t *testing.T) {
	e := setup(t, nil)
	_, tok := e.user(t, "09120000811")
	r := e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":1}`)
	require.Equal(t, http.StatusServiceUnavailable, r.status, r.raw)
	assert.JSONEq(t, `{"success":false,"message":"Payment is not available right now. Please try again later.","error_code":"payment_unavailable"}`, r.raw)
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM plus_invoices`))
	// Preview still prices.
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":1,"preview":true}`)
	assert.Equal(t, http.StatusOK, r.status)
}

func TestCheckout_Validation(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	_, tok := e.user(t, "09120000812")
	r := e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.JSONEq(t, `{"success":false,"message":"Validation failed","errors":{"plan_id":["The plan field is required."]}}`, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "fa", `{"plan_id":"abc","discount_code":["x"]}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.raw, `"plan_id"`)
	assert.Contains(t, r.raw, `"discount_code"`)
	_, err := e.db.Exec(`UPDATE plus_plans SET is_active = 0 WHERE id = ?`, plan6m)
	require.NoError(t, err)
	for _, id := range []string{"3", "99"} {
		r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":`+id+`}`)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
		assert.JSONEq(t, `{"success":false,"message":"Validation failed","errors":{"plan_id":["The selected plan is not available."]}}`, r.raw)
	}
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.raw, `"reference"`)
	assert.Contains(t, r.raw, `"authority"`)
}

func TestDiscount_LimitsAndExpiry(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	_, tokA := e.user(t, "09120000813")
	_, tokB := e.user(t, "09120000814")
	e.discount(t, "ONCE", plus.DiscountAmount, 100000, 1, nil, nil)
	e.discount(t, "OLD", plus.DiscountPercent, 50, nil, nil, "2026-09-23 10:00:00")
	e.discount(t, "LATER", plus.DiscountPercent, 50, nil, nil, "2026-09-23 10:00:01")

	r := e.do(t, http.MethodPost, "/api/v1/plus/checkout", tokA, "en", `{"plan_id":1,"discount_code":"nope","preview":true}`)
	assert.Equal(t, "discount_invalid", r.body["error_code"])
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tokA, "en", `{"plan_id":1,"discount_code":"OLD","preview":true}`)
	assert.Equal(t, "discount_expired", r.body["error_code"], "expires_at is exclusive")
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tokA, "en", `{"plan_id":1,"discount_code":"LATER","preview":true}`)
	assert.Equal(t, http.StatusOK, r.status, r.raw)

	// A's pending checkout reserves the only redemption …
	_, _, r = e.checkout(t, now, tokA, `{"plan_id":1,"discount_code":"ONCE"}`)
	assert.EqualValues(t, 100000, r.obj("invoice")["discount"])
	assert.EqualValues(t, (990000-100000)*11/10, r.obj("invoice")["total"])
	r = e.do(t, http.MethodPost, "/api/v1/plus/checkout", tokB, "en", `{"plan_id":1,"discount_code":"ONCE"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "discount_exhausted", r.body["error_code"])
	// … until it expires unpaid (30 minutes).
	refB, authB, _ := e.checkout(t, "2026-09-23T10:30:00+03:30", tokB, `{"plan_id":1,"discount_code":"ONCE"}`)
	r = e.doAt(t, "2026-09-23T10:31:00+03:30", http.MethodPost, "/api/v1/plus/verify", tokB, "en", verifyBody(refB, authB, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	// Paid now: exhausted for good.
	r = e.doAt(t, "2026-10-23T10:00:00+03:30", http.MethodPost, "/api/v1/plus/checkout", tokA, "en", `{"plan_id":1,"discount_code":"ONCE","preview":true}`)
	assert.Equal(t, "discount_exhausted", r.body["error_code"])
}

func TestDiscount_ConcurrentCheckoutsCannotOverrunLimit(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	e.discount(t, "RACE", plus.DiscountPercent, 20, 2, nil, nil)
	uids := make([]uint64, 8)
	for i := range uids {
		uids[i], _ = e.user(t, "0912000090"+string(rune('0'+i)))
	}
	var wg sync.WaitGroup
	errs := make([]error, len(uids))
	for i, uid := range uids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = e.svc.Checkout(context.Background(), uid, plan1m, "RACE", tNow)
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
			continue
		}
		var de *plus.DiscountError
		require.ErrorAs(t, err, &de)
		assert.Equal(t, plus.DiscountExhausted, de.Reason)
	}
	assert.Equal(t, 2, ok)
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM plus_invoices WHERE discount_code = 'RACE'`))
}

func TestCheckout_FullDiscountSettlesAtOnce(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	uid, tok := e.user(t, "09120000815")
	e.discount(t, "FREE", plus.DiscountPercent, 100, nil, 1, nil)
	r := e.do(t, http.MethodPost, "/api/v1/plus/checkout", tok, "en", `{"plan_id":1,"discount_code":"FREE"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Ritme Plus is active", r.body["message"])
	assert.Nil(t, r.data()["payment"])
	assert.Equal(t, "paid", r.obj("invoice")["status"])
	assert.EqualValues(t, 0, r.obj("invoice")["total"])
	assert.Nil(t, r.obj("invoice")["gateway"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
}

func TestPurchases_StackAndCancel(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	_, tok := e.user(t, "09120000816")

	r := e.do(t, http.MethodPost, "/api/v1/plus/cancel", tok, "en", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "no_subscription", r.body["error_code"])

	ref, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	r = e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	// A second purchase (the «ارتقا به ۶ ماهه» upgrade) starts when the first ends.
	ref, authority, _ = e.checkout(t, "2026-10-01T09:00:00+03:30", tok, `{"plan_id":3}`)
	r = e.doAt(t, "2026-10-01T09:00:00+03:30", http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var starts, ends time.Time
	require.NoError(t, e.db.QueryRow(`SELECT starts_at, ends_at FROM plus_subscriptions ORDER BY id DESC LIMIT 1`).Scan(&starts, &ends))
	assert.Equal(t, "2026-10-23 10:00:00", starts.In(tehran).Format(time.DateTime))
	assert.Equal(t, "2027-04-23 10:00:00", ends.In(tehran).Format(time.DateTime))

	r = e.doAt(t, "2026-10-01T09:00:00+03:30", http.MethodPost, "/api/v1/plus/cancel", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "تمدید خودکار خاموش شد. پلاس تا پایان اشتراکت فعال می\u200cماند", r.body["message"])
	assert.Equal(t, "plus", r.data()["tier"])
	sub := r.obj("subscription")
	assert.Equal(t, "canceled", sub["status"])
	assert.Equal(t, false, sub["auto_renew"])
	assert.Equal(t, "2026-10-01T09:00:00+03:30", sub["canceled_at"])
	assert.Equal(t, 2, 0+e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE status = 'canceled'`), "the queued period too")
	r = e.doAt(t, "2026-10-02T09:00:00+03:30", http.MethodPost, "/api/v1/plus/cancel", tok, "en", "")
	assert.Equal(t, http.StatusOK, r.status, "idempotent")

	// Still Plus through the queued period, free after it.
	r = e.doAt(t, "2027-04-23T09:59:59+03:30", http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	assert.Equal(t, "plus", r.data()["tier"])
	r = e.doAt(t, "2027-04-23T10:00:00+03:30", http.MethodGet, "/api/v1/plus/status", tok, "en", "")
	assert.Equal(t, "free", r.data()["tier"])
	assert.Nil(t, r.data()["subscription"])
}

func TestRestore(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	_, tok := e.user(t, "09120000817")
	ref, _, _ := e.checkout(t, now, tok, `{"plan_id":1}`)

	// The fake cannot confirm a payment without the callback: unpaid, still pending inside the reservation.
	r := e.do(t, http.MethodPost, "/api/v1/plus/restore", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "No purchase to restore", r.body["message"])
	assert.EqualValues(t, 0, r.data()["restored"])
	assert.Equal(t, "free", r.obj("status")["tier"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_invoices WHERE reference = ? AND status = 'pending'`, ref))
	// After the reservation it is closed.
	r = e.doAt(t, "2026-09-23T11:00:00+03:30", http.MethodPost, "/api/v1/plus/restore", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM plus_invoices WHERE reference = ? AND status = 'failed'`, ref))
}

func TestUsage_QuotaAndConcurrency(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	uid, tok := e.user(t, "09120000818")
	ctx := context.Background()

	_, err := e.svc.Consume(ctx, uid, plus.LabAI, tNow)
	require.ErrorIs(t, err, plus.ErrNotEntitled, "free users have no lab AI")
	ent, err := e.svc.Consume(ctx, uid, plus.AssistantUnlimited, tNow)
	require.NoError(t, err)
	assert.Equal(t, 4, ent.Remaining, "free assistant quota")
	_, err = e.svc.Consume(ctx, uid, "plus.unknown", tNow)
	require.ErrorIs(t, err, plus.ErrUnknownFeature)

	require.NoError(t, e.svc.StartTrial(ctx, uid, tNow))
	var wg sync.WaitGroup
	errs := make([]error, 25)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = e.svc.Consume(ctx, uid, plus.LabAI, tNow)
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else {
			assert.ErrorIs(t, err, plus.ErrQuotaExceeded)
		}
	}
	assert.Equal(t, 10, ok, "exactly the monthly quota")
	_, err = e.svc.Consume(ctx, uid, plus.DeepAnalysis, tNow)
	require.NoError(t, err)

	r := e.do(t, http.MethodGet, "/api/v1/plus/usage", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "trial", r.data()["tier"])
	assert.Equal(t, "2026-09-01", r.data()["period_start"])
	assert.Equal(t, "2026-10-01T00:00:00+03:30", r.data()["resets_at"])
	assert.Equal(t, map[string]any{"key": "plus.lab_ai", "allowed": false, "unlimited": false, "limit": float64(10), "used": float64(10), "remaining": float64(0)},
		entitlement(t, r.data()["features"], "plus.lab_ai"))
	assert.Equal(t, map[string]any{"key": "plus.deep_analysis", "allowed": true, "unlimited": true, "limit": nil, "used": float64(1), "remaining": nil},
		entitlement(t, r.data()["features"], "plus.deep_analysis"))

	// Next month the quota is fresh (the trial is over by then: buy a month first).
	ref, authority, _ := e.checkout(t, "2026-10-02T10:00:00+03:30", tok, `{"plan_id":1}`)
	r = e.doAt(t, "2026-10-02T10:00:00+03:30", http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	ent, err = e.svc.Consume(ctx, uid, plus.LabAI, mustTime("2026-10-02T10:00:00+03:30"))
	require.NoError(t, err)
	assert.Equal(t, 9, ent.Remaining)
	e2, err := e.svc.Entitlement(ctx, uid, plus.LabAI, mustTime("2026-10-02T10:00:00+03:30"))
	require.NoError(t, err)
	assert.Equal(t, ent, e2, "Entitlement reads without counting")
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

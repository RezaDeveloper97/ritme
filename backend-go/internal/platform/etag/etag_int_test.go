package etag_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"io"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/etag"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

func TestMain(m *testing.M) { testdb.Main(m) }

type sessionKey struct{}

// engineApp is an engine GET as T-M9-32+ will wire it: query counter → the auth query (passport, revisions in the
// same row) → etag.Middleware from the auth data, the request clock and the locale → a handler doing real work.
// queries receives each request's query count.
type engineApp struct {
	app     *fiber.App
	queries []int64
	hits    int
	hdr     string
	clk     clock.Clock
	pool    *sql.DB
}

func newEngineApp(t testing.TB) *engineApp {
	t.Helper()
	ctx := context.Background()
	pool := testdb.New(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	_, err = pool.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('etag-client', 'c', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`)
	require.NoError(t, err)
	res, err := pool.Exec(`INSERT INTO users (mobile, created_at, updated_at) VALUES ('09120000951', NOW(), NOW())`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = pool.Exec(`INSERT INTO user_revisions (user_id, engine_rev, analysis_rev, updated_at) VALUES (?, 3, 3, NOW())`, uid)
	require.NoError(t, err)

	q := store.New(pool)
	issuedAt := time.Now()
	tok, err := passport.NewIssuer(key, q, clock.At(issuedAt), 365).Issue(ctx, uint64(uid), issuedAt) //nolint:gosec // id
	require.NoError(t, err)
	v := passport.NewVerifier(&key.PublicKey, q, clock.At(issuedAt))

	ea := &engineApp{app: fiber.New(), hdr: "Bearer " + tok.AccessToken, clk: clock.At(issuedAt), pool: pool}
	ea.app.Use(func(c fiber.Ctx) error {
		cctx, counter := metrics.WithQueryCounter(c.Context())
		c.SetContext(cctx)
		err := c.Next()
		ea.queries = append(ea.queries, counter.Load())
		return err
	})
	ea.app.Use(func(c fiber.Ctx) error {
		s, err := v.Verify(c.Context(), c.Get(fiber.HeaderAuthorization))
		if err != nil {
			return fiber.ErrUnauthorized
		}
		c.Locals(sessionKey{}, s)
		return c.Next()
	})
	ea.app.Get("/cycle/today", etag.Middleware(func(c fiber.Ctx) (string, error) {
		s := c.Locals(sessionKey{}).(*passport.Session)
		today := civildate.LocalToday(c.Context(), ea.clk)
		return etag.Engine{
			UserID: s.User.ID, Rev: s.Revisions.Engine, Today: today, Locale: c.Get(fiber.HeaderAcceptLanguage),
			ParamsFP: engineparams.Fingerprint(today), ConfigFP: configcache.Fingerprint("message_contents"),
			Build: etag.Build(),
		}.Tag(), nil
	}), func(c fiber.Ctx) error {
		ea.hits++
		s := c.Locals(sessionKey{}).(*passport.Session)
		u, err := q.GetUserByID(c.Context(), s.User.ID) // stands for the engine work
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"data": fiber.Map{"id": u.ID}})
	})
	ea.get(t, "", "fa") // warm the client cache
	return ea
}

func (ea *engineApp) get(t testing.TB, inm, locale string) (int, string, string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "/cycle/today", nil)
	req.Header.Set(fiber.HeaderAuthorization, ea.hdr)
	req.Header.Set(fiber.HeaderAcceptLanguage, locale)
	if inm != "" {
		req.Header.Set(fiber.HeaderIfNoneMatch, inm)
	}
	res, err := ea.app.Test(req)
	require.NoError(t, err)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, string(body), res.Header.Get(fiber.HeaderETag)
}

func (ea *engineApp) last() int64 { return ea.queries[len(ea.queries)-1] }

// Same inputs → 304 with no body and only the auth query; a locale change → a new tag (200).
func TestEngineGetInt_304IsOnlyTheAuthQuery(t *testing.T) {
	ea := newEngineApp(t)
	code, body, tag := ea.get(t, "", "fa")
	require.Equal(t, fiber.StatusOK, code)
	assert.NotEmpty(t, body)
	assert.Equal(t, int64(2), ea.last(), "200: auth + the handler's work")
	hits := ea.hits

	code, body, tag2 := ea.get(t, tag, "fa")
	assert.Equal(t, fiber.StatusNotModified, code)
	assert.Empty(t, body)
	assert.Equal(t, tag, tag2)
	assert.Equal(t, int64(1), ea.last(), "304: only the auth query")
	assert.Equal(t, hits, ea.hits, "the handler never ran")

	code, _, tagEn := ea.get(t, tag, "en")
	assert.Equal(t, fiber.StatusOK, code, "locale change → new tag")
	assert.NotEqual(t, tag, tagEn)
}

// Budget: the 304 path ≤ 5 ms p95 locally with 1 query. Asserted only when a trivial round trip is fast (the dev
// machine is often loaded); the numbers are logged either way.
func TestEngineGetInt_304Budget(t *testing.T) {
	ea := newEngineApp(t)
	_, _, tag := ea.get(t, "", "fa")
	const n = 300
	d := make([]time.Duration, 0, n)
	for range n {
		start := time.Now()
		code, _, _ := ea.get(t, tag, "fa")
		d = append(d, time.Since(start))
		require.Equal(t, fiber.StatusNotModified, code)
		require.Equal(t, int64(1), ea.last())
	}
	slices.Sort(d)
	p50, p95 := d[n/2], d[n*95/100]
	t.Logf("304 path over %d requests: p50 %v, p95 %v (1 query each)", n, p50, p95)
	trip := make([]time.Duration, 0, 50)
	for range 50 {
		start := time.Now()
		_, err := ea.pool.Exec("SELECT 1")
		require.NoError(t, err)
		trip = append(trip, time.Since(start))
	}
	slices.Sort(trip)
	if trip[25] < time.Millisecond {
		assert.LessOrEqual(t, p95, 5*time.Millisecond, "304 p95 budget")
	} else {
		t.Logf("SELECT 1 p50 %v ≥ 1 ms: machine loaded, budget not asserted", trip[25])
	}
}

func BenchmarkEngineGetInt_304(b *testing.B) {
	ea := newEngineApp(b)
	_, _, tag := ea.get(b, "", "fa")
	b.ResetTimer()
	for b.Loop() {
		if code, _, _ := ea.get(b, tag, "fa"); code != fiber.StatusNotModified {
			b.Fatal(code)
		}
	}
}

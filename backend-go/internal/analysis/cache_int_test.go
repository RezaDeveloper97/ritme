package analysis_test

// T-M10-28 (D-177): the analysis cache end to end on MariaDB + miniredis — hit / miss, a past vitals edit (analysis_rev
// bump), a Plus purchase (the {plus} key part) and a registry change (cfgFP) each serve a fresh body, the 304 path, two
// accounts never share a body, the long-range gate runs before the cache, and the warm job fills and then skips.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/clinicalconfig"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/vitals"
	"github.com/ritme/backend-go/resources/translations"
)

type cacheEnv struct {
	*env
	mr     *miniredis.Miniredis
	h      *analysis.Handlers
	vitals *vitals.Service
}

func setupCache(t *testing.T) *cacheEnv {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	languages := i18n.NewRegistry(i18nstore.New(db), nil, quiet)
	locale := i18n.Middleware(languages)
	vs := vitals.NewService(db)
	h := analysis.NewHandlers(cycleservice.New(db, nil), healthlog.NewService(db),
		plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet),
		i18n.NewTranslationStore(translations.FS, ""), languages, clock.Real{}).
		WithVitals(vs).
		WithCache(analysis.NewCache(cache.NewFromClient(rdb, "ritme-go-t:"), "test", quiet))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/analysis/summary", locale, guard, h.Summary)
	app.Get("/api/v1/analysis/cycle", locale, guard, h.Cycle)
	return &cacheEnv{env: &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}, mr: mr, h: h, vitals: vs}
}

type tagged struct {
	response
	etag, cacheControl string
}

func (e *cacheEnv) fetch(t *testing.T, url, token, ifNoneMatch string) tagged {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, strings.NewReader(""))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, now)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	out := tagged{response: response{status: resp.StatusCode, raw: string(raw)}, etag: resp.Header.Get("ETag"),
		cacheControl: resp.Header.Get("Cache-Control")}
	if len(raw) > 0 {
		require.NoError(t, jsonUnmarshal(raw, &out.body))
	}
	return out
}

func (e *cacheEnv) keys() int { return len(e.mr.Keys()) }

func TestAnalysisCache_HitMissInvalidationAndETag(t *testing.T) {
	e := setupCache(t)
	a, tokA := e.user(t, "09120000101")
	b, tokB := e.user(t, "09120000102")
	seedA(t, e.env, a)
	e.periods(t, b, "2026-09-01")

	first := e.fetch(t, "/api/v1/analysis/summary", tokA, "")
	require.Equal(t, 200, first.status, first.raw)
	assert.Equal(t, "private, no-cache", first.cacheControl)
	assert.True(t, strings.HasPrefix(first.etag, `W/"u`+strconvBase36(a)+".a"), first.etag)
	assert.Equal(t, 1, e.keys(), "a miss stores the body")

	// A write that bypasses the revision (a raw insert) is not seen: the second request is a hit.
	e.periods(t, a, "2026-09-20")
	hit := e.fetch(t, "/api/v1/analysis/summary", tokA, "")
	require.Equal(t, 200, hit.status)
	assert.Equal(t, first.raw, hit.raw)
	assert.Equal(t, first.etag, hit.etag)
	assert.Equal(t, 1, e.keys())

	// 304 on a matching If-None-Match (no body).
	nm := e.fetch(t, "/api/v1/analysis/summary", tokA, first.etag)
	assert.Equal(t, http.StatusNotModified, nm.status)
	assert.Empty(t, nm.raw)
	assert.Equal(t, first.etag, nm.etag)

	// Two accounts never share a body: B's tag names B, and A's tag does not revalidate B's request.
	other := e.fetch(t, "/api/v1/analysis/summary", tokB, first.etag)
	require.Equal(t, 200, other.status, other.raw)
	assert.NotEqual(t, first.etag, other.etag)
	assert.True(t, strings.HasPrefix(other.etag, `W/"u`+strconvBase36(b)+".a"), other.etag)
	assert.NotEqual(t, first.raw, other.raw)

	// A past vitals edit (through the vitals service: analysis_rev + 1 in its transaction) → a new key and tag; the
	// fresh body also sees the raw period above.
	ctx := context.Background()
	at := time.Date(2026, 9, 2, 8, 0, 0, 0, civildate.Tehran)
	rd, err := e.vitals.Create(ctx, a, vitals.Input{Type: vitals.TypeBP, MeasuredAt: at, Systolic: 120, Diastolic: 80}, at)
	require.NoError(t, err)
	_, err = e.vitals.Update(ctx, a, rd.ID, vitals.Input{Type: vitals.TypeBP, MeasuredAt: at, Systolic: 135, Diastolic: 88}, at)
	require.NoError(t, err)
	edited := e.fetch(t, "/api/v1/analysis/summary", tokA, first.etag)
	require.Equal(t, 200, edited.status, "the old tag no longer matches")
	assert.NotEqual(t, first.etag, edited.etag)
	assert.NotEqual(t, first.raw, edited.raw)

	// A Plus purchase (a trial here) changes the {plus} part: a new body with the Plus sections open.
	assert.Equal(t, true, path(edited.data(), "sections", "mood_by_phase", "locked"))
	grantTrial(t, e.db, a)
	plusBody := e.fetch(t, "/api/v1/analysis/summary", tokA, edited.etag)
	require.Equal(t, 200, plusBody.status)
	assert.NotEqual(t, edited.etag, plusBody.etag)
	assert.Equal(t, false, path(plusBody.data(), "sections", "mood_by_phase", "locked"))

	// A long Plus range is cached for the trial user, and once the trial ends the gate answers 402 before the cache.
	long := e.fetch(t, "/api/v1/analysis/summary?range=1y", tokA, "")
	require.Equal(t, 200, long.status, long.raw)
	revokeTrial(t, e.db, a)
	gated := e.fetch(t, "/api/v1/analysis/summary?range=1y", tokA, long.etag)
	assert.Equal(t, http.StatusPaymentRequired, gated.status, gated.raw)
	assert.Equal(t, "plus_required", gated.body["error_code"])

	// A registry change (a new clinical-config version valid today) moves cfgFP.
	before := e.fetch(t, "/api/v1/analysis/cycle", tokA, "")
	require.Equal(t, 200, before.status)
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: clinicalconfig.KeyFIGOFrequency, Version: 2,
		Value: clinicalconfig.Value{"min": 25, "max": 36}, EffectiveFrom: civildate.MustParse("2026-01-01")}})
	after := e.fetch(t, "/api/v1/analysis/cycle", tokA, before.etag)
	require.Equal(t, 200, after.status, "a new registry version never revalidates the old body")
	assert.NotEqual(t, before.etag, after.etag)
}

func TestAnalysisCache_WarmJobFillsThenSkips(t *testing.T) {
	e := setupCache(t)
	a, tokA := e.user(t, "09120000201")
	b, _ := e.user(t, "09120000202")
	seedA(t, e.env, a)
	e.periods(t, b, "2026-09-01")

	w := analysis.NewWarmer(e.h, e.db, nil)
	at, err := time.Parse(time.RFC3339, now)
	require.NoError(t, err)
	st, err := w.Run(context.Background(), at)
	require.NoError(t, err)
	assert.Equal(t, analysis.WarmStats{Warmed: 2}, st, "both users have a token from today")
	assert.Equal(t, 2, e.keys())

	// The user's first open of the day is a hit on the warmed body.
	r := e.fetch(t, "/api/v1/analysis/summary", tokA, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 2, e.keys())

	// Idempotent: a second run the same day skips everyone and writes nothing.
	st, err = w.Run(context.Background(), at)
	require.NoError(t, err)
	assert.Equal(t, analysis.WarmStats{Skipped: 2}, st)
	assert.Equal(t, 2, e.keys())
}

func strconvBase36(id uint64) string { return strconv.FormatUint(id, 36) }

func jsonUnmarshal(raw []byte, v any) error { return json.Unmarshal(raw, v) }

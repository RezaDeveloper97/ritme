package projections_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/family"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	envNow      = "2026-09-23T10:00:00+03:30"
	fixtureKeys = "../../../contract/fixtures/keys"
)

// env is the production route registry (internal/http.Mount) on a fresh test database, the request clock pinned.
type env struct {
	t   *testing.T
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	keysPath, err := filepath.Abs(fixtureKeys)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe95', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	apihttp.Mount(app, &apihttp.Deps{
		Config: &config.Config{
			App:         config.App{Env: "testing", URL: "http://localhost"},
			SMS:         config.SMS{Provider: "log"},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keysPath,
		},
		DB:     db,
		Cache:  cache.NewFromClient(rdb, "ritme-go-m1106:"),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})
	return &env{t: t, db: db, app: app, iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

func (e *env) exec(q string, args ...any) uint64 {
	e.t.Helper()
	res, err := e.db.Exec(q, args...)
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	return uint64(id) //nolint:gosec // test id
}

// account creates a user with a life profile and returns (id, token).
func (e *env) account(mobile, gender, mode string) (uint64, string) {
	e.t.Helper()
	id := e.exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, '2026-07-01 09:00:00', '2026-07-01 09:00:00')`, mobile)
	var lm any
	if mode != "" {
		lm = mode
	}
	e.exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, ?, ?, '2026-07-01 09:00:00', '2026-07-01 09:00:00')`,
		id, gender, lm)
	tok, err := e.iss.Issue(context.Background(), id, time.Now())
	require.NoError(e.t, err)
	return id, tok.AccessToken
}

// link is an active link owner → member of role with v6 grants (domain code → level), stamped with the owner's
// current mode epoch like family.Grants.Apply.
func (e *env) link(owner, member uint64, role string, grants map[string]string) uint64 {
	e.t.Helper()
	cid := e.exec(`INSERT INTO companions (owner_id, companion_user_id, type, role, status, display_name, invited_at, accepted_at, created_at, updated_at)
		VALUES (?, ?, 'partner', ?, 'active', 'Ali', '2026-09-01 09:00:00', '2026-09-01 09:00:00', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`,
		owner, member, role)
	st, err := family.SyncMode(context.Background(), e.db, owner, time.Now())
	require.NoError(e.t, err)
	for d, l := range grants {
		e.exec(`INSERT INTO companion_grants (companion_id, section, level, mode_epoch, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())`,
			cid, d, l, st.Epoch)
	}
	return cid
}

func (e *env) cyclePeriod(owner uint64, lastStart string) {
	e.t.Helper()
	e.exec(`INSERT INTO user_profiles (user_id, last_period_start, period_duration, cycle_duration, created_at, updated_at)
		VALUES (?, ?, 5, 28, '2026-07-01 09:00:00', '2026-07-01 09:00:00')`, owner, lastStart)
}

func (e *env) pregnant(owner uint64, lmp string) {
	e.t.Helper()
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', ?, '2026-08-20 09:00:00', '2026-08-20 09:00:00')`, owner, lmp)
}

type reply struct {
	Status int
	Raw    string
	Cache  string
	Body   map[string]any
}

func (r reply) data() map[string]any {
	d, _ := r.Body["data"].(map[string]any)
	return d
}

// inner is data.data of a /family domain read.
func (r reply) inner() any { return r.data()["data"] }

func (e *env) get(path, token string) reply {
	e.t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set(clock.Header, envNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	out := reply{Status: resp.StatusCode, Raw: string(raw), Cache: resp.Header.Get(fiber.HeaderCacheControl)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func domainPath(owner uint64, d string) string {
	return fmt.Sprintf("/api/v1/family/owners/%d/domains/%s", owner, d)
}

func sectionPath(link uint64, s string) string {
	return fmt.Sprintf("/api/v1/companions/links/%d/sections/%s", link, s)
}

// keysOf is the sorted-free key set of a JSON object.
func keysOf(v any) []string {
	m, _ := v.(map[string]any)
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains(haystack string, needles ...string) []string {
	var hit []string
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			hit = append(hit, n)
		}
	}
	return hit
}

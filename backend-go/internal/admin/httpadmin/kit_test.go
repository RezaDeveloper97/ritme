package httpadmin_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type fakeAdmins map[uint64]store.Admin

func (f fakeAdmins) GetAdminByID(_ context.Context, id uint64) (store.Admin, error) {
	a, ok := f[id]
	if !ok {
		return store.Admin{}, sql.ErrNoRows
	}
	return a, nil
}

type env struct {
	app      *fiber.App
	mr       *miniredis.Miniredis
	sessions *httpadmin.Sessions
	admins   fakeAdmins
	kit      *httpadmin.Kit
}

func newEnv(t *testing.T, opts httpadmin.Options) *env {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	sessions := httpadmin.NewSessions(cache.NewFromClient(rdb, "ritme-go:"))
	admins := fakeAdmins{
		1: {ID: 1, Name: "Root", Email: "root@x.test", Role: httpadmin.RoleSuper, IsActive: true},
		2: {ID: 2, Name: "Ed", Email: "ed@x.test", Role: httpadmin.RoleEditor, IsActive: true},
		3: {ID: 3, Name: "Off", Email: "off@x.test", Role: httpadmin.RoleSuper, IsActive: false},
	}
	kit := httpadmin.NewKit(opts, sessions, admins, nil, quiet)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	p := httpadmin.Prefix
	ok := func(c fiber.Ctx) error { return httpadmin.OK(c, httpadmin.CurrentAdmin(c).ID) }
	app.Options(p+"/*", kit.Preflight)
	httpadmin.Handle(app, fiber.MethodPost, p+"/public", kit.Public(ok0))
	httpadmin.Handle(app, fiber.MethodGet, p+"/me", kit.Admin(ok))
	httpadmin.Handle(app, fiber.MethodPost, p+"/mutate", kit.Admin(ok))
	httpadmin.Handle(app, fiber.MethodGet, p+"/super", kit.Super(ok))
	return &env{app: app, mr: mr, sessions: sessions, admins: admins, kit: kit}
}

func ok0(c fiber.Ctx) error { return httpadmin.OK(c, nil) }

func devOpts() httpadmin.Options {
	return httpadmin.Options{Hosts: []string{"adpanell.ritme.app"}, Origins: []string{"http://localhost:3001"}, CookieSecure: true}
}

type reply struct {
	status int
	body   map[string]any
	header http.Header
}

func (e *env) do(t *testing.T, method, path string, mod func(r *http.Request)) reply {
	t.Helper()
	var body io.Reader
	if method != fiber.MethodGet {
		body = strings.NewReader(`{}`)
	}
	r := httptest.NewRequest(method, httpadmin.Prefix+path, body)
	r.Host = "adpanell.ritme.app"
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if mod != nil {
		mod(r)
	}
	resp, err := e.app.Test(r)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	out := reply{status: resp.StatusCode, header: resp.Header}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.body)
	}
	return out
}

func (e *env) login(t *testing.T, adminID uint64, remember bool) *httpadmin.Session {
	t.Helper()
	s, err := e.sessions.Create(context.Background(), adminID, remember, time.Now())
	require.NoError(t, err)
	return s
}

func withSession(k *httpadmin.Kit, s *httpadmin.Session, csrf bool) func(*http.Request) {
	return func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: k.SessionCookieName(), Value: s.ID()})
		if csrf {
			r.Header.Set(httpadmin.CSRFHeader, s.CSRF)
		}
	}
}

func TestHostGuard_OtherHost404(t *testing.T) {
	e := newEnv(t, devOpts())
	s := e.login(t, 1, false)
	res := e.do(t, fiber.MethodGet, "/me", func(r *http.Request) {
		withSession(e.kit, s, false)(r)
		r.Host = "api.ritme.app"
	})
	assert.Equal(t, 404, res.status)
	assert.Equal(t, 200, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false)).status)
}

func TestHostGuard_FailClosedWithoutHostsOutsideDev(t *testing.T) {
	e := newEnv(t, httpadmin.Options{})
	assert.Equal(t, 404, e.do(t, fiber.MethodPost, "/public", nil).status)
	e = newEnv(t, httpadmin.Options{Dev: true})
	assert.Equal(t, 200, e.do(t, fiber.MethodPost, "/public", nil).status)
}

func TestRequireAdmin(t *testing.T) {
	e := newEnv(t, devOpts())

	res := e.do(t, fiber.MethodGet, "/me", nil)
	assert.Equal(t, 401, res.status)
	assert.Equal(t, httpadmin.CodeUnauthenticated, res.body["error_code"])
	assert.Equal(t, false, res.body["success"])

	res = e.do(t, fiber.MethodGet, "/me", func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: e.kit.SessionCookieName(), Value: strings.Repeat("A", 43)})
	})
	assert.Equal(t, 401, res.status)
	assert.Equal(t, httpadmin.CodeSessionExpired, res.body["error_code"])

	s := e.login(t, 1, false)
	res = e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false))
	assert.Equal(t, 200, res.status)
	assert.EqualValues(t, 1, res.body["data"])
}

func TestRequireAdmin_InactiveAdminLosesSession(t *testing.T) {
	e := newEnv(t, devOpts())
	s := e.login(t, 3, false)
	res := e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false))
	assert.Equal(t, 401, res.status)
	assert.Equal(t, httpadmin.CodeAdminInactive, res.body["error_code"])
	assert.Contains(t, strings.Join(res.header.Values("Set-Cookie"), "\n"), "__Host-ritme_admin_session=;")
	got, err := e.sessions.Load(context.Background(), s.ID())
	require.NoError(t, err)
	assert.Nil(t, got, "session destroyed")

	// Deactivated mid-session (the Blade admin.active middleware).
	s = e.login(t, 2, false)
	require.Equal(t, 200, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false)).status)
	a := e.admins[2]
	a.IsActive = false
	e.admins[2] = a
	assert.Equal(t, 401, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false)).status)
	delete(e.admins, 2) // deleted admin: same answer
	assert.Equal(t, 401, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false)).status)
}

func TestSessionExpiry_SlidingIdleTimeout(t *testing.T) {
	e := newEnv(t, devOpts())
	s := e.login(t, 1, false)
	e.mr.FastForward(httpadmin.IdleTTL - time.Minute)
	require.Equal(t, 200, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false)).status, "still alive")
	e.mr.FastForward(httpadmin.IdleTTL - time.Minute) // slid forward by the last request
	require.Equal(t, 200, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false)).status)
	e.mr.FastForward(httpadmin.IdleTTL + time.Second)
	res := e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false))
	assert.Equal(t, 401, res.status)
	assert.Equal(t, httpadmin.CodeSessionExpired, res.body["error_code"])

	r := e.login(t, 1, true) // remember me: 30 days
	e.mr.FastForward(29 * 24 * time.Hour)
	assert.Equal(t, 200, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, r, false)).status)
	e.mr.FastForward(httpadmin.RememberTTL + time.Second)
	assert.Equal(t, 401, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, r, false)).status)
}

func TestCSRF(t *testing.T) {
	e := newEnv(t, devOpts())
	s := e.login(t, 1, false)

	res := e.do(t, fiber.MethodPost, "/mutate", withSession(e.kit, s, false))
	assert.Equal(t, httpadmin.StatusCSRF, res.status)
	assert.Equal(t, httpadmin.CodeCSRF, res.body["error_code"])

	res = e.do(t, fiber.MethodPost, "/mutate", func(r *http.Request) {
		withSession(e.kit, s, false)(r)
		r.Header.Set(httpadmin.CSRFHeader, "wrong")
	})
	assert.Equal(t, httpadmin.StatusCSRF, res.status)

	other := e.login(t, 1, false) // another session's token does not work either
	res = e.do(t, fiber.MethodPost, "/mutate", func(r *http.Request) {
		withSession(e.kit, s, false)(r)
		r.Header.Set(httpadmin.CSRFHeader, other.CSRF)
	})
	assert.Equal(t, httpadmin.StatusCSRF, res.status)

	assert.Equal(t, 200, e.do(t, fiber.MethodPost, "/mutate", withSession(e.kit, s, true)).status)
	assert.Equal(t, 200, e.do(t, fiber.MethodGet, "/me", withSession(e.kit, s, false)).status, "GET needs no token")
}

func TestRequireSuper(t *testing.T) {
	e := newEnv(t, devOpts())
	res := e.do(t, fiber.MethodGet, "/super", withSession(e.kit, e.login(t, 2, false), false))
	assert.Equal(t, 403, res.status)
	assert.Equal(t, httpadmin.CodeForbidden, res.body["error_code"])
	assert.Equal(t, 200, e.do(t, fiber.MethodGet, "/super", withSession(e.kit, e.login(t, 1, false), false)).status)
}

func TestOriginAndCORS(t *testing.T) {
	e := newEnv(t, devOpts())
	s := e.login(t, 1, false)

	// Cross-site mutating request from an unknown origin: refused before the session is used.
	res := e.do(t, fiber.MethodPost, "/mutate", func(r *http.Request) {
		withSession(e.kit, s, true)(r)
		r.Header.Set("Origin", "https://evil.example")
	})
	assert.Equal(t, 403, res.status)
	assert.Equal(t, httpadmin.CodeOrigin, res.body["error_code"])
	assert.Empty(t, res.header.Get("Access-Control-Allow-Origin"))

	// Same host origin.
	res = e.do(t, fiber.MethodPost, "/mutate", func(r *http.Request) {
		withSession(e.kit, s, true)(r)
		r.Header.Set("Origin", "https://adpanell.ritme.app")
	})
	assert.Equal(t, 200, res.status)
	assert.Empty(t, res.header.Get("Access-Control-Allow-Origin"))

	// Configured admin-web origin: credentials allowed.
	res = e.do(t, fiber.MethodPost, "/mutate", func(r *http.Request) {
		withSession(e.kit, s, true)(r)
		r.Header.Set("Origin", "http://localhost:3001")
	})
	assert.Equal(t, 200, res.status)
	assert.Equal(t, "http://localhost:3001", res.header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", res.header.Get("Access-Control-Allow-Credentials"))

	// Preflight.
	res = e.do(t, fiber.MethodOptions, "/mutate", func(r *http.Request) {
		r.Header.Set("Origin", "http://localhost:3001")
		r.Header.Set("Access-Control-Request-Method", "POST")
	})
	assert.Equal(t, 204, res.status)
	assert.Equal(t, "http://localhost:3001", res.header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, res.header.Get("Access-Control-Allow-Headers"), httpadmin.CSRFHeader)
	res = e.do(t, fiber.MethodOptions, "/mutate", func(r *http.Request) {
		r.Header.Set("Origin", "https://evil.example")
		r.Header.Set("Access-Control-Request-Method", "POST")
	})
	assert.Empty(t, res.header.Get("Access-Control-Allow-Origin"))
}

func TestPublicRequiresJSON(t *testing.T) {
	e := newEnv(t, devOpts())
	res := e.do(t, fiber.MethodPost, "/public", func(r *http.Request) {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	})
	assert.Equal(t, 415, res.status)
	assert.Equal(t, 200, e.do(t, fiber.MethodPost, "/public", nil).status)
}

func TestCookies(t *testing.T) {
	e := newEnv(t, devOpts())
	s := e.login(t, 1, false)
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error { e.kit.SetCookies(c, s); return nil })
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	require.NoError(t, err)
	cookies := resp.Header.Values("Set-Cookie")
	require.Len(t, cookies, 2)
	sessionCookie, csrfCookie := cookies[0], cookies[1]
	assert.True(t, strings.HasPrefix(sessionCookie, "__Host-ritme_admin_session="+s.ID()))
	lower := strings.ToLower(sessionCookie)
	for _, attr := range []string{"path=/", "httponly", "secure", "samesite=lax"} {
		assert.Contains(t, lower, attr)
	}
	assert.NotContains(t, lower, "domain=")
	assert.NotContains(t, lower, "max-age", "browser-session cookie without remember-me")
	assert.True(t, strings.HasPrefix(csrfCookie, "__Host-ritme_admin_csrf="+s.CSRF))
	assert.NotContains(t, strings.ToLower(csrfCookie), "httponly")

	r := e.login(t, 1, true)
	app2 := fiber.New()
	app2.Get("/", func(c fiber.Ctx) error { e.kit.SetCookies(c, r); return nil })
	resp, err = app2.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(resp.Header.Values("Set-Cookie")[0]), "max-age=2592000")
}

func TestSessions_StoredHashedAndDestroyAll(t *testing.T) {
	e := newEnv(t, devOpts())
	ctx := context.Background()
	a := e.login(t, 1, false)
	b := e.login(t, 1, true)
	c := e.login(t, 2, false)
	for _, k := range e.mr.Keys() {
		assert.NotContains(t, k, a.ID(), "raw session id never appears in Redis")
		assert.True(t, strings.HasPrefix(k, "ritme-go:admin-session"), k)
	}
	require.NoError(t, e.sessions.DestroyAll(ctx, 1, a))
	got, _ := e.sessions.Load(ctx, a.ID())
	assert.NotNil(t, got, "kept")
	got, _ = e.sessions.Load(ctx, b.ID())
	assert.Nil(t, got)
	got, _ = e.sessions.Load(ctx, c.ID())
	assert.NotNil(t, got, "other admin untouched")
	require.NoError(t, e.sessions.DestroyAll(ctx, 1, nil))
	got, _ = e.sessions.Load(ctx, a.ID())
	assert.Nil(t, got)
}

func TestOptionsFrom(t *testing.T) {
	env := map[string]string{"ADMIN_HOSTS": " AdPanell.ritme.app , ", "ADMIN_WEB_ORIGINS": "http://localhost:3001/", "ADMIN_COOKIE_SECURE": "false"}
	o := httpadmin.OptionsFrom(func(k string) (string, bool) { v, ok := env[k]; return v, ok }, "local")
	assert.Equal(t, []string{"adpanell.ritme.app"}, o.Hosts)
	assert.Equal(t, []string{"http://localhost:3001"}, o.Origins)
	assert.False(t, o.CookieSecure)
	assert.True(t, o.Dev)
	o = httpadmin.OptionsFrom(func(string) (string, bool) { return "", false }, "production")
	assert.True(t, o.CookieSecure)
	assert.False(t, o.Dev)
}

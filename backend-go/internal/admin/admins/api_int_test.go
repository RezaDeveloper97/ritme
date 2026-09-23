package admins_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

	"github.com/ritme/backend-go/internal/admin/admins"
	adminauth "github.com/ritme/backend-go/internal/admin/auth"
	"github.com/ritme/backend-go/internal/admin/dashboard"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/admin/users"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// Made by Laravel 12 (Illuminate\Hashing\BcryptHasher, rounds 12) for "Secret-pass-123".
const (
	laravelHash = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi"
	password    = "Secret-pass-123"
	host        = "adpanell.ritme.test"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type api struct {
	t   *testing.T
	db  *sql.DB
	app *fiber.App
	mr  *miniredis.Miniredis
}

// newAPI mounts the same routes as internal/http/routes_admin_core.go on a fresh DB.
func newAPI(t *testing.T) *api {
	t.Helper()
	db := testdb.New(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	c := cache.NewFromClient(rdb, "ritme-go:")

	locale := i18n.DefaultMiddleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	sessions := httpadmin.NewSessions(c)
	q := store.New(db)
	kit := httpadmin.NewKit(httpadmin.Options{Hosts: []string{host}, CookieSecure: true}, sessions, q, locale, quiet)
	authH := adminauth.NewHandlers(kit, q, ratelimit.New(c, clock.Real{}))
	dash := dashboard.NewHandlers(q)
	usersH := users.NewHandlers(db, quiet)
	adminsH := admins.NewHandlers(db, sessions, quiet)

	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(quiet),
		// As cmd/api: client IP from X-Forwarded-For behind the trusted nginx.
		ProxyHeader: fiber.HeaderXForwardedFor, TrustProxy: true, EnableIPValidation: true,
		TrustProxyConfig: fiber.TrustProxyConfig{Proxies: []string{"0.0.0.0/0", "::/0"}},
	})
	p := httpadmin.Prefix
	h := func(method, path string, chain httpadmin.Chain) { httpadmin.Handle(app, method, p+path, chain) }
	get, post, put, del := fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete
	h(post, "/auth/login", kit.Public(authH.Login))
	h(post, "/auth/logout", kit.Admin(authH.Logout))
	h(get, "/auth/me", kit.Admin(authH.Me))
	h(put, "/auth/password", kit.Admin(authH.ChangePassword))
	h(get, "/dashboard", kit.Admin(dash.Show))
	h(get, "/users", kit.Admin(usersH.List))
	h(get, "/users/:id", kit.Admin(usersH.Show))
	h(put, "/users/:id", kit.Admin(usersH.Update))
	h(post, "/users/:id/block", kit.Admin(usersH.Block))
	h(post, "/users/:id/unblock", kit.Admin(usersH.Unblock))
	h(del, "/users/:id", kit.Admin(usersH.Destroy))
	h(get, "/admins", kit.Super(adminsH.List))
	h(post, "/admins", kit.Super(adminsH.Store))
	h(get, "/admins/:id", kit.Super(adminsH.Show))
	h(put, "/admins/:id", kit.Super(adminsH.Update))
	h(del, "/admins/:id", kit.Super(adminsH.Destroy))

	a := &api{t: t, db: db, app: app, mr: mr}
	a.exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(1, 'Root', 'root@ritme.test', ?, 'super', 1, '2026-09-01 10:00:00', '2026-09-01 10:00:00'),
		(2, 'Editor', 'editor@ritme.test', ?, 'editor', 1, '2026-09-01 10:00:00', '2026-09-01 10:00:00'),
		(3, 'Gone', 'gone@ritme.test', ?, 'super', 0, '2026-09-01 10:00:00', '2026-09-01 10:00:00')`,
		laravelHash, laravelHash, laravelHash)
	return a
}

func (a *api) exec(query string, args ...any) {
	a.t.Helper()
	_, err := a.db.Exec(query, args...)
	require.NoError(a.t, err)
}

func (a *api) scalar(query string, args ...any) int {
	a.t.Helper()
	var n int
	require.NoError(a.t, a.db.QueryRow(query, args...).Scan(&n))
	return n
}

// client is a browser: it keeps the cookies and echoes the CSRF token.
type client struct {
	a       *api
	cookies map[string]string
	csrf    string
}

type resp struct {
	status int
	body   map[string]any
	header http.Header
}

func (r resp) data() map[string]any { d, _ := r.body["data"].(map[string]any); return d }

func (a *api) client() *client { return &client{a: a, cookies: map[string]string{}} }

func (c *client) do(method, path string, body any) resp {
	c.a.t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.a.t, err)
		rd = strings.NewReader(string(b))
	}
	r := httptest.NewRequest(method, httpadmin.Prefix+path, rd)
	r.Host = host
	r.Header.Set("Accept", "application/json")
	if rd != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.cookies {
		r.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	if c.csrf != "" {
		r.Header.Set(httpadmin.CSRFHeader, c.csrf)
	}
	res, err := c.a.app.Test(r, fiber.TestConfig{Timeout: 30 * time.Second}) // bcrypt cost 12 is slow
	require.NoError(c.a.t, err)
	defer func() { _ = res.Body.Close() }()
	for _, ck := range res.Cookies() {
		if ck.MaxAge < 0 || ck.Value == "" {
			delete(c.cookies, ck.Name)
		} else {
			c.cookies[ck.Name] = ck.Value
		}
	}
	raw, _ := io.ReadAll(res.Body)
	out := resp{status: res.StatusCode, header: res.Header}
	_ = json.Unmarshal(raw, &out.body)
	return out
}

func (c *client) login(email string) resp {
	c.a.t.Helper()
	r := c.do(fiber.MethodPost, "/auth/login", map[string]any{"email": email, "password": password})
	if r.status == 200 {
		c.csrf, _ = r.data()["csrf_token"].(string)
	}
	return r
}

func TestLogin_LaravelHashSessionAndLogout(t *testing.T) {
	a := newAPI(t)
	c := a.client()
	r := c.login("root@ritme.test")
	require.Equal(t, 200, r.status, r.body)
	assert.Equal(t, true, r.body["success"])
	admin := r.data()["admin"].(map[string]any)
	assert.Equal(t, "root@ritme.test", admin["email"])
	assert.Equal(t, "super", admin["role"])
	assert.NotContains(t, admin, "password")
	assert.NotEmpty(t, c.csrf)
	assert.Equal(t, 1, a.scalar(`SELECT COUNT(*) FROM admins WHERE id = 1 AND last_login_at IS NOT NULL`))
	assert.Contains(t, c.cookies, "__Host-ritme_admin_session")

	me := c.do(fiber.MethodGet, "/auth/me", nil)
	require.Equal(t, 200, me.status)
	assert.Equal(t, c.csrf, me.data()["csrf_token"])

	// Logout without the CSRF header is refused, with it the session ends.
	saved := c.csrf
	c.csrf = ""
	assert.Equal(t, 419, c.do(fiber.MethodPost, "/auth/logout", nil).status)
	c.csrf = saved
	assert.Equal(t, 200, c.do(fiber.MethodPost, "/auth/logout", nil).status)
	assert.Equal(t, 401, c.do(fiber.MethodGet, "/auth/me", nil).status)
}

func TestLogin_Failures(t *testing.T) {
	a := newAPI(t)
	c := a.client()

	r := c.do(fiber.MethodPost, "/auth/login", map[string]any{"email": "root@ritme.test", "password": "wrong"})
	assert.Equal(t, 422, r.status)
	assert.Equal(t, httpadmin.CodeInvalidLogin, r.body["error_code"])

	inactive := c.login("gone@ritme.test") // correct password, inactive account
	assert.Equal(t, 422, inactive.status)
	assert.Equal(t, r.body, inactive.body, "inactive and wrong password answer the same")

	unknown := c.login("nobody@ritme.test")
	assert.Equal(t, r.body, unknown.body)

	v := c.do(fiber.MethodPost, "/auth/login", map[string]any{"email": "not-an-email"})
	assert.Equal(t, 422, v.status)
	assert.Equal(t, httpadmin.CodeValidation, v.body["error_code"])
	errs := v.body["errors"].(map[string]any)
	assert.Contains(t, errs, "email")
	assert.Contains(t, errs, "password")
	assert.Empty(t, c.cookies)
}

func TestLogin_Throttle5PerMinutePerIPAndEmail(t *testing.T) {
	a := newAPI(t)
	c := a.client()
	for i := range 5 {
		r := c.do(fiber.MethodPost, "/auth/login", map[string]any{"email": "root@ritme.test", "password": "x"})
		require.Equal(t, 422, r.status, "attempt %d", i+1)
	}
	r := c.login("root@ritme.test") // even the right password
	assert.Equal(t, 429, r.status)
	assert.Equal(t, httpadmin.CodeThrottled, r.body["error_code"])
	assert.NotEmpty(t, r.header.Get("Retry-After"))
	assert.Equal(t, 200, c.login("editor@ritme.test").status, "other email from the same IP")
}

func TestLogin_PerEmailCapAcrossIPs(t *testing.T) {
	a := newAPI(t)
	c := a.client()
	for i := range adminauth.LoginMaxEmailAttempts {
		r := httptest.NewRequest(fiber.MethodPost, httpadmin.Prefix+"/auth/login",
			strings.NewReader(`{"email":"root@ritme.test","password":"x"}`))
		r.Host = host
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Forwarded-For", fmt.Sprintf("10.0.%d.%d", i/200, i%200+1)) // a spoofed IP per attempt
		res, err := a.app.Test(r, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		_ = res.Body.Close()
		require.Equal(t, 422, res.StatusCode, "attempt %d", i+1)
	}
	assert.Equal(t, 429, c.login("root@ritme.test").status)
}

func TestEditorCannotReachSuperEndpoints(t *testing.T) {
	a := newAPI(t)
	c := a.client()
	require.Equal(t, 200, c.login("editor@ritme.test").status)
	r := c.do(fiber.MethodGet, "/admins", nil)
	assert.Equal(t, 403, r.status)
	assert.Equal(t, httpadmin.CodeForbidden, r.body["error_code"])
	assert.Equal(t, 403, c.do(fiber.MethodPost, "/admins", map[string]any{"name": "x"}).status)
	assert.Equal(t, 403, c.do(fiber.MethodDelete, "/admins/1", nil).status)
	assert.Equal(t, 1, a.scalar(`SELECT COUNT(*) FROM admins WHERE id = 1`))
	assert.Equal(t, 200, c.do(fiber.MethodGet, "/dashboard", nil).status, "editors see the dashboard")
}

func TestDeactivatedAdminIsLoggedOut(t *testing.T) {
	a := newAPI(t)
	c := a.client()
	require.Equal(t, 200, c.login("editor@ritme.test").status)
	a.exec(`UPDATE admins SET is_active = 0 WHERE id = 2`)
	r := c.do(fiber.MethodGet, "/auth/me", nil)
	assert.Equal(t, 401, r.status)
	assert.Equal(t, httpadmin.CodeAdminInactive, r.body["error_code"])
	a.exec(`UPDATE admins SET is_active = 1 WHERE id = 2`)
	assert.Equal(t, 401, c.do(fiber.MethodGet, "/auth/me", nil).status, "the session was destroyed")
}

func TestChangeOwnPassword(t *testing.T) {
	a := newAPI(t)
	c, other := a.client(), a.client()
	require.Equal(t, 200, c.login("editor@ritme.test").status)
	require.Equal(t, 200, other.login("editor@ritme.test").status)

	r := c.do(fiber.MethodPut, "/auth/password", map[string]any{
		"current_password": "nope", "password": "brand-new-pass", "password_confirmation": "brand-new-pass"})
	assert.Equal(t, 422, r.status)
	assert.Contains(t, r.body["errors"], "current_password")

	r = c.do(fiber.MethodPut, "/auth/password", map[string]any{
		"current_password": password, "password": "brand-new-pass", "password_confirmation": "different"})
	assert.Equal(t, 422, r.status)
	assert.Contains(t, r.body["errors"], "password")

	r = c.do(fiber.MethodPut, "/auth/password", map[string]any{
		"current_password": password, "password": "short", "password_confirmation": "short"})
	assert.Equal(t, 422, r.status)

	r = c.do(fiber.MethodPut, "/auth/password", map[string]any{
		"current_password": password, "password": "brand-new-pass", "password_confirmation": "brand-new-pass"})
	require.Equal(t, 200, r.status, r.body)
	var hash string
	require.NoError(t, a.db.QueryRow(`SELECT password FROM admins WHERE id = 2`).Scan(&hash))
	assert.True(t, strings.HasPrefix(hash, "$2a$12$"))
	assert.True(t, adminauth.CheckPassword(hash, "brand-new-pass"))
	assert.Equal(t, 200, c.do(fiber.MethodGet, "/auth/me", nil).status, "current session kept")
	assert.Equal(t, 401, other.do(fiber.MethodGet, "/auth/me", nil).status, "other sessions ended")
}

func seedUsers(a *api) {
	a.exec(`INSERT INTO users (id, name, email, mobile, blocked_at, created_at, updated_at) VALUES
		(10, 'Sara Ahmadi', NULL, '09120000010', NULL, '2026-09-20 10:00:00', '2026-09-20 10:00:00'),
		(11, NULL, 'x_y@mail.test', '09120000011', '2026-09-21 09:00:00', '2026-09-21 10:00:00', '2026-09-21 10:00:00'),
		(12, 'Mina', NULL, '09120000012', NULL, '2026-09-22 10:00:00', '2026-09-22 10:00:00')`)
	a.exec(`INSERT INTO user_profiles (user_id, subscription_type, user_goal, cycle_duration, created_at, updated_at)
		VALUES (10, 'premium', 'ttc', 28, '2026-09-20 10:00:00', '2026-09-20 10:00:00')`)
	for i, uid := range []int{10, 10, 12} {
		a.exec(`INSERT INTO oauth_access_tokens (id, user_id, client_id, name, scopes, revoked, created_at, updated_at, expires_at)
			VALUES (?, ?, 'c1', 'auth_token', '[]', 0, NOW(), NOW(), NOW() + INTERVAL 365 DAY)`,
			fmt.Sprintf("%080d", i+1), uid)
	}
}

func ids(r resp) []float64 {
	var out []float64
	for _, it := range r.data()["items"].([]any) {
		out = append(out, it.(map[string]any)["id"].(float64))
	}
	return out
}

func TestUsers_ListFiltersAndShow(t *testing.T) {
	a := newAPI(t)
	seedUsers(a)
	c := a.client()
	require.Equal(t, 200, c.login("editor@ritme.test").status)

	r := c.do(fiber.MethodGet, "/users", nil)
	require.Equal(t, 200, r.status)
	assert.Equal(t, []float64{12, 11, 10}, ids(r), "latest first")
	meta := r.data()["meta"].(map[string]any)
	assert.EqualValues(t, 3, meta["total"])
	assert.EqualValues(t, 20, meta["per_page"])

	assert.Equal(t, []float64{11}, ids(c.do(fiber.MethodGet, "/users?status=blocked", nil)))
	assert.Equal(t, []float64{12, 10}, ids(c.do(fiber.MethodGet, "/users?status=active", nil)))
	assert.Equal(t, []float64{10}, ids(c.do(fiber.MethodGet, "/users?q=sara", nil)))
	assert.Equal(t, []float64{11}, ids(c.do(fiber.MethodGet, "/users?q=0000011", nil)))
	assert.Equal(t, []float64{11}, ids(c.do(fiber.MethodGet, "/users?q=x_y", nil)))
	assert.Empty(t, ids(c.do(fiber.MethodGet, "/users?q=%25", nil)), "LIKE wildcards are literal")
	page := c.do(fiber.MethodGet, "/users?per_page=2&page=2", nil)
	assert.Equal(t, []float64{10}, ids(page))
	assert.EqualValues(t, 2, page.data()["meta"].(map[string]any)["last_page"])

	s := c.do(fiber.MethodGet, "/users/10", nil)
	require.Equal(t, 200, s.status)
	u := s.data()["user"].(map[string]any)
	assert.Equal(t, "premium", u["subscription_type"])
	assert.Equal(t, "2026-09-20T10:00:00+03:30", u["created_at"])
	assert.EqualValues(t, 28, s.data()["profile"].(map[string]any)["cycle_duration"])
	assert.Nil(t, c.do(fiber.MethodGet, "/users/12", nil).data()["profile"])
	assert.Contains(t, s.data(), "stats")
	assert.Equal(t, 404, c.do(fiber.MethodGet, "/users/999", nil).status)
	assert.Equal(t, 404, c.do(fiber.MethodGet, "/users/abc", nil).status)
}

func TestUsers_UpdateBlockUnblockDelete(t *testing.T) {
	a := newAPI(t)
	seedUsers(a)
	c := a.client()
	require.Equal(t, 200, c.login("editor@ritme.test").status)

	// Update creates the profile on demand.
	r := c.do(fiber.MethodPut, "/users/12", map[string]any{"name": "Mina K", "subscription_type": "premium", "user_goal": "ttc"})
	require.Equal(t, 200, r.status, r.body)
	assert.Equal(t, 1, a.scalar(`SELECT COUNT(*) FROM user_profiles WHERE user_id = 12 AND subscription_type = 'premium' AND user_goal = 'ttc'`))
	assert.Equal(t, 1, a.scalar(`SELECT COUNT(*) FROM users WHERE id = 12 AND name = 'Mina K'`))
	bad := c.do(fiber.MethodPut, "/users/12", map[string]any{"subscription_type": "gold", "user_goal": "ttc"})
	assert.Equal(t, 422, bad.status)
	assert.Contains(t, bad.body["errors"], "subscription_type")

	// Block revokes every token of the user (and only theirs).
	b := c.do(fiber.MethodPost, "/users/10/block", nil)
	require.Equal(t, 200, b.status, b.body)
	assert.EqualValues(t, 2, b.data()["revoked_tokens"])
	assert.Equal(t, 0, a.scalar(`SELECT COUNT(*) FROM oauth_access_tokens WHERE user_id = 10 AND revoked = 0`))
	assert.Equal(t, 1, a.scalar(`SELECT COUNT(*) FROM oauth_access_tokens WHERE user_id = 12 AND revoked = 0`))
	assert.Equal(t, 1, a.scalar(`SELECT COUNT(*) FROM users WHERE id = 10 AND blocked_at IS NOT NULL`))

	u := c.do(fiber.MethodPost, "/users/10/unblock", nil)
	require.Equal(t, 200, u.status)
	assert.Equal(t, 1, a.scalar(`SELECT COUNT(*) FROM users WHERE id = 10 AND blocked_at IS NULL`))
	assert.Equal(t, 404, c.do(fiber.MethodPost, "/users/999/block", nil).status)

	// Delete revokes the tokens too (D-03).
	d := c.do(fiber.MethodDelete, "/users/12", nil)
	require.Equal(t, 200, d.status, d.body)
	assert.Equal(t, 0, a.scalar(`SELECT COUNT(*) FROM users WHERE id = 12`))
	assert.Equal(t, 0, a.scalar(`SELECT COUNT(*) FROM oauth_access_tokens WHERE user_id = 12 AND revoked = 0`))
	assert.Equal(t, 404, c.do(fiber.MethodDelete, "/users/12", nil).status)
}

func TestDashboard(t *testing.T) {
	a := newAPI(t)
	seedUsers(a)
	c := a.client()
	require.Equal(t, 200, c.login("root@ritme.test").status)
	r := c.do(fiber.MethodGet, "/dashboard", nil)
	require.Equal(t, 200, r.status)
	stats := r.data()["stats"].(map[string]any)
	assert.EqualValues(t, 3, stats["users"])
	assert.EqualValues(t, 1, stats["users_blocked"])
	assert.Len(t, r.data()["recent_users"], 3)
}

func TestAdmins_CRUDAndSelfProtection(t *testing.T) {
	a := newAPI(t)
	c := a.client()
	require.Equal(t, 200, c.login("root@ritme.test").status)

	list := c.do(fiber.MethodGet, "/admins", nil)
	require.Equal(t, 200, list.status)
	assert.Equal(t, []float64{3, 2, 1}, ids(list))

	create := map[string]any{"name": "New", "email": "new@ritme.test", "password": "new-admin-pass",
		"password_confirmation": "new-admin-pass", "role": "editor"}
	r := c.do(fiber.MethodPost, "/admins", create)
	require.Equal(t, 201, r.status, r.body)
	newID := r.data()["admin"].(map[string]any)["id"].(float64)
	assert.Equal(t, true, r.data()["admin"].(map[string]any)["is_active"])
	var hash string
	require.NoError(t, a.db.QueryRow(`SELECT password FROM admins WHERE email = 'new@ritme.test'`).Scan(&hash))
	assert.True(t, strings.HasPrefix(hash, "$2a$12$"))

	dup := c.do(fiber.MethodPost, "/admins", create)
	assert.Equal(t, 422, dup.status)
	assert.Contains(t, dup.body["errors"], "email")
	badRole := c.do(fiber.MethodPost, "/admins", map[string]any{"name": "N", "email": "n2@ritme.test",
		"password": "new-admin-pass", "password_confirmation": "new-admin-pass", "role": "owner"})
	assert.Equal(t, 422, badRole.status)

	// The new admin logs in; deactivating them ends that session.
	other := a.client()
	lr := other.do(fiber.MethodPost, "/auth/login", map[string]any{"email": "new@ritme.test", "password": "new-admin-pass"})
	require.Equal(t, 200, lr.status, lr.body)
	path := fmt.Sprintf("/admins/%d", int(newID))
	up := c.do(fiber.MethodPut, path, map[string]any{"name": "New", "email": "new@ritme.test", "role": "editor", "is_active": false})
	require.Equal(t, 200, up.status, up.body)
	assert.Equal(t, 401, other.do(fiber.MethodGet, "/auth/me", nil).status)

	// Self: no demotion, no deactivation, no deletion.
	self := c.do(fiber.MethodPut, "/admins/1", map[string]any{"name": "Root", "email": "root@ritme.test", "role": "editor"})
	assert.Equal(t, 422, self.status)
	assert.Equal(t, httpadmin.CodeSelf, self.body["error_code"])
	self = c.do(fiber.MethodPut, "/admins/1", map[string]any{"name": "Root", "email": "root@ritme.test", "role": "super", "is_active": false})
	assert.Equal(t, 422, self.status)
	assert.Equal(t, 422, c.do(fiber.MethodDelete, "/admins/1", nil).status)
	ok := c.do(fiber.MethodPut, "/admins/1", map[string]any{"name": "Root 2", "email": "root@ritme.test", "role": "super"})
	assert.Equal(t, 200, ok.status, ok.body)

	del := c.do(fiber.MethodDelete, path, nil)
	require.Equal(t, 200, del.status)
	assert.Equal(t, 404, c.do(fiber.MethodGet, path, nil).status)
}

func TestWrongHostIs404(t *testing.T) {
	a := newAPI(t)
	r := httptest.NewRequest(fiber.MethodPost, httpadmin.Prefix+"/auth/login",
		strings.NewReader(`{"email":"root@ritme.test","password":"`+password+`"}`))
	r.Host = "api.ritme.app"
	r.Header.Set("Content-Type", "application/json")
	res, err := a.app.Test(r)
	require.NoError(t, err)
	_ = res.Body.Close()
	assert.Equal(t, 404, res.StatusCode)
}

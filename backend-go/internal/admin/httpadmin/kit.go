package httpadmin

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"log/slog"
	"net/url"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Roles (App\Models\Admin::ROLE_*).
const (
	RoleSuper  = "super"
	RoleEditor = "editor"
)

// Roles lists the valid roles.
var Roles = []string{RoleSuper, RoleEditor}

// CSRFHeader carries the session's CSRF token on every mutating request.
const CSRFHeader = "X-CSRF-Token"

// AdminLoader loads an admin by id (store.Querier satisfies it).
type AdminLoader interface {
	GetAdminByID(ctx context.Context, id uint64) (store.Admin, error)
}

// Kit holds the admin middleware chains. Build one per routes file (Wire / NewKit).
type Kit struct {
	opts     Options
	sessions *Sessions
	admins   AdminLoader
	locale   fiber.Handler // i18n.DefaultMiddleware (setlocale:default)
	logger   *slog.Logger
}

// NewKit wires the chains. locale pins the request locale (i18n.DefaultMiddleware); nil
// leaves the locale unset (the default language is used anyway).
func NewKit(opts Options, sessions *Sessions, admins AdminLoader, locale fiber.Handler, logger *slog.Logger) *Kit {
	if logger == nil {
		logger = slog.Default()
	}
	if locale == nil {
		locale = func(c fiber.Ctx) error { return c.Next() }
	}
	if !opts.Dev && len(opts.Hosts) == 0 {
		logger.Error("admin API disabled: ADMIN_HOSTS is not set (required outside APP_ENV=local/testing)")
	}
	return &Kit{opts: opts, sessions: sessions, admins: admins, locale: locale, logger: logger}
}

// Sessions returns the session store.
func (k *Kit) Sessions() *Sessions { return k.sessions }

// Options returns the configuration.
func (k *Kit) Options() Options { return k.opts }

// Logger returns the kit's logger.
func (k *Kit) Logger() *slog.Logger { return k.logger }

// Chain is a route's handler list: middleware first, the endpoint last.
type Chain []any

// Handle registers chain for method + path. Fiber runs a route's handlers in argument
// order, so the chain's middleware always runs before the endpoint.
func Handle(r fiber.Router, method, path string, chain Chain) {
	r.Add([]string{method}, path, chain[0], chain[1:]...)
}

// Public is the chain for unauthenticated endpoints (login): host guard, CORS, Origin
// check, default locale, and a JSON body for mutating requests (a cross-site HTML form
// cannot send application/json without a CORS preflight).
func (k *Kit) Public(h fiber.Handler) Chain {
	return Chain{k.edge, k.locale, requireJSON, h}
}

// Admin is the chain for any active admin: host guard, CORS, Origin check, default
// locale, RequireAdmin and CSRF (no JSON requirement: uploads are multipart).
func (k *Kit) Admin(h fiber.Handler) Chain {
	return Chain{k.edge, k.locale, k.RequireAdmin, k.RequireCSRF, h}
}

// Super is Admin plus RequireSuper.
func (k *Kit) Super(h fiber.Handler) Chain {
	return Chain{k.edge, k.locale, k.RequireAdmin, k.RequireCSRF, RequireSuper, h}
}

// ---------------------------------------------------------------------------
// Edge: host guard, CORS, Origin check

func isSafeMethod(m string) bool {
	return m == fiber.MethodGet || m == fiber.MethodHead || m == fiber.MethodOptions
}

// requestHost is the Host header without port, lower-cased (nginx forwards $host).
func requestHost(c fiber.Ctx) string {
	h := strings.ToLower(string(c.Request().Host()))
	if i := strings.LastIndexByte(h, ':'); i >= 0 && !strings.HasSuffix(h, "]") {
		h = h[:i]
	}
	return h
}

func (k *Kit) hostAllowed(c fiber.Ctx) bool {
	if len(k.opts.Hosts) == 0 {
		return k.opts.Dev
	}
	return slices.Contains(k.opts.Hosts, requestHost(c))
}

// originAllowed: no Origin header (same-origin GETs, curl), an Origin on the admin host
// itself, or one of the configured admin-web origins.
func (k *Kit) originAllowed(origin string) (sameHost, listed bool) {
	if origin == "" {
		return true, false
	}
	if slices.Contains(k.opts.Origins, origin) {
		return false, true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false, false
	}
	host := strings.ToLower(u.Hostname())
	if len(k.opts.Hosts) == 0 {
		return k.opts.Dev, false
	}
	return slices.Contains(k.opts.Hosts, host), false
}

func (k *Kit) setCORS(c fiber.Ctx, origin string) {
	c.Set(fiber.HeaderAccessControlAllowOrigin, origin)
	c.Set(fiber.HeaderAccessControlAllowCredentials, "true")
	c.Vary(fiber.HeaderOrigin)
}

func (k *Kit) edge(c fiber.Ctx) error {
	if !k.hostAllowed(c) {
		return httpx.RouteNotFound(string(c.Request().URI().PathOriginal()))
	}
	origin := c.Get(fiber.HeaderOrigin)
	sameHost, listed := k.originAllowed(origin)
	if listed {
		k.setCORS(c, origin)
	}
	if !sameHost && !listed && !isSafeMethod(c.Method()) {
		return Fail(fiber.StatusForbidden, CodeOrigin, "Cross-site request refused.")
	}
	return c.Next()
}

// Preflight answers CORS preflights under Prefix for the configured admin-web origins.
// Register it once: r.Options(Prefix+"/*", kit.Preflight).
func (k *Kit) Preflight(c fiber.Ctx) error {
	if !k.hostAllowed(c) {
		return httpx.RouteNotFound(string(c.Request().URI().PathOriginal()))
	}
	origin := c.Get(fiber.HeaderOrigin)
	if _, listed := k.originAllowed(origin); listed {
		k.setCORS(c, origin)
		c.Set(fiber.HeaderAccessControlAllowMethods, "GET, POST, PUT, PATCH, DELETE")
		c.Set(fiber.HeaderAccessControlAllowHeaders, "Accept, Accept-Language, Content-Type, "+CSRFHeader)
		c.Set(fiber.HeaderAccessControlMaxAge, "600")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func requireJSON(c fiber.Ctx) error {
	if isSafeMethod(c.Method()) {
		return c.Next()
	}
	ct := strings.ToLower(strings.TrimSpace(strings.SplitN(c.Get(fiber.HeaderContentType), ";", 2)[0]))
	if ct != fiber.MIMEApplicationJSON {
		return Fail(fiber.StatusUnsupportedMediaType, CodeUnsupported, "Send the request body as application/json.")
	}
	return c.Next()
}

// ---------------------------------------------------------------------------
// Authentication

type localsKey int

const (
	adminKey localsKey = iota
	sessionKey
)

// CurrentAdmin is the authenticated admin (nil outside RequireAdmin).
func CurrentAdmin(c fiber.Ctx) *store.Admin {
	a, _ := c.Locals(adminKey).(*store.Admin)
	return a
}

// CurrentSession is the authenticated session (nil outside RequireAdmin).
func CurrentSession(c fiber.Ctx) *Session {
	s, _ := c.Locals(sessionKey).(*Session)
	return s
}

// RequireAdmin is auth:admin + admin.active: a valid session whose admin still exists and
// is active. Otherwise the session is destroyed, the cookies are cleared and the request
// answers 401.
func (k *Kit) RequireAdmin(c fiber.Ctx) error {
	raw := c.Cookies(k.SessionCookieName())
	if raw == "" {
		return Unauthenticated(CodeUnauthenticated)
	}
	sess, err := k.sessions.Load(c.Context(), raw)
	if err != nil {
		return err
	}
	if sess == nil {
		k.ClearCookies(c)
		return Unauthenticated(CodeSessionExpired)
	}
	admin, err := k.admins.GetAdminByID(c.Context(), sess.AdminID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err != nil || !admin.IsActive {
		if derr := k.sessions.Destroy(c.Context(), sess); derr != nil {
			return derr
		}
		k.ClearCookies(c)
		return Unauthenticated(CodeAdminInactive)
	}
	if err := k.sessions.Touch(c.Context(), sess); err != nil {
		return err
	}
	c.Locals(adminKey, &admin)
	c.Locals(sessionKey, sess)
	return c.Next()
}

// RequireCSRF checks X-CSRF-Token against the session token on mutating requests (419).
// It runs after RequireAdmin.
func (k *Kit) RequireCSRF(c fiber.Ctx) error {
	if isSafeMethod(c.Method()) {
		return c.Next()
	}
	sess := CurrentSession(c)
	got := c.Get(CSRFHeader)
	if sess == nil || got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(sess.CSRF)) != 1 {
		return Fail(StatusCSRF, CodeCSRF, "CSRF token mismatch.")
	}
	return c.Next()
}

// RequireSuper is admin.super (403). It runs after RequireAdmin.
func RequireSuper(c fiber.Ctx) error {
	if a := CurrentAdmin(c); a == nil || a.Role != RoleSuper {
		return Forbidden()
	}
	return c.Next()
}

// Wire builds the Kit from the shared dependencies (the routes_admin_*.go registrars):
// options from the environment (appEnv = config.App.Env), sessions in c, admins from db,
// and the default-language locale middleware (`setlocale:default`).
func Wire(appEnv string, db *sql.DB, c *cache.Client, logger *slog.Logger) *Kit {
	locale := i18n.DefaultMiddleware(i18n.NewRegistry(i18nstore.New(db), c, logger))
	return NewKit(OptionsFromEnv(appEnv), NewSessions(c), store.New(db), locale, logger)
}

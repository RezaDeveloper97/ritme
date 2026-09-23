package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Login throttle: 5 attempts per minute per client IP + email (AuthController's
// throttle:5,1, narrowed to the email so one office IP does not lock out every admin),
// plus 20 per minute per email from any IP, which caps guessing even when the client
// IP is spoofed through X-Forwarded-For.
const (
	LoginMaxAttempts      = 5
	LoginMaxEmailAttempts = 20
	LoginDecay            = time.Minute
)

// Querier is what the handlers need from the admin store.
type Querier interface {
	GetAdminByEmail(ctx context.Context, email string) (store.Admin, error)
	TouchAdminLastLogin(ctx context.Context, arg store.TouchAdminLastLoginParams) error
	UpdateAdminPassword(ctx context.Context, arg store.UpdateAdminPasswordParams) error
}

// Handlers are the /auth endpoints.
type Handlers struct {
	kit     *httpadmin.Kit
	q       Querier
	limiter *ratelimit.Limiter // nil = no throttle (unit tests)
	logger  *slog.Logger
}

// NewHandlers wires the handlers.
func NewHandlers(kit *httpadmin.Kit, q Querier, limiter *ratelimit.Limiter) *Handlers {
	return &Handlers{kit: kit, q: q, limiter: limiter, logger: kit.Logger()}
}

// invalidLogin is Laravel's `auth.failed` answer: the same for an unknown email, a wrong
// password and an inactive account (no account enumeration).
func invalidLogin() error {
	const msg = "These credentials do not match our records."
	return httpadmin.Fail(fiber.StatusUnprocessableEntity, httpadmin.CodeInvalidLogin, msg,
		"errors", jsonx.Obj("email", []string{msg}))
}

func throttleKey(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.Join(parts, "|"))))
	return hex.EncodeToString(sum[:])
}

// throttle counts one login attempt; ok=false carries the seconds to wait.
func (h *Handlers) throttle(c fiber.Ctx, email string) (retryAfter int, ok bool, err error) {
	if h.limiter == nil {
		return 0, true, nil
	}
	now := httpadmin.Now(c)
	for _, lim := range []struct {
		key string
		max int
	}{
		{"admin-login:" + throttleKey(email, c.IP()), LoginMaxAttempts},
		{"admin-login-email:" + throttleKey(email), LoginMaxEmailAttempts},
	} {
		res, err := h.limiter.Attempt(c.Context(), lim.key, lim.max, LoginDecay, now)
		if err != nil {
			return 0, false, err
		}
		if !res.Allowed {
			return res.RetryAfter(now), false, nil
		}
	}
	return 0, true, nil
}

// Login is POST /auth/login {email, password, remember?}.
func (h *Handlers) Login(c fiber.Ctx) error {
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("email", "required|string|email|max:255"),
		validation.F("password", "required|string|max:255"),
		validation.F("remember", "sometimes|boolean"),
	})
	if err != nil {
		return err
	}
	email := httpadmin.String(data, "email")
	password := httpadmin.String(data, "password")

	if retry, ok, err := h.throttle(c, email); err != nil {
		return err
	} else if !ok {
		h.logger.WarnContext(c.Context(), "admin login throttled", slog.String("ip", c.IP()))
		return httpadmin.Throttled(retry)
	}

	admin, err := h.q.GetAdminByEmail(c.Context(), email)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		burnTime(password)
		return h.failed(c)
	case err != nil:
		return err
	case !admin.IsActive:
		burnTime(password)
		return h.failed(c)
	case !CheckPassword(admin.Password, password):
		return h.failed(c)
	}

	// A new id on every login (session fixation); drop the session this browser had.
	sessions := h.kit.Sessions()
	if raw := c.Cookies(h.kit.SessionCookieName()); raw != "" {
		if old, err := sessions.Load(c.Context(), raw); err == nil && old != nil {
			_ = sessions.Destroy(c.Context(), old)
		}
	}
	now := httpadmin.Now(c)
	sess, err := sessions.Create(c.Context(), admin.ID, httpadmin.Bool(data, "remember"), now)
	if err != nil {
		return err
	}
	stamp := httpadmin.DBTime(now)
	if err := h.q.TouchAdminLastLogin(c.Context(), store.TouchAdminLastLoginParams{Now: stamp, ID: admin.ID}); err != nil {
		return err
	}
	admin.LastLoginAt, admin.UpdatedAt = stamp, stamp

	h.kit.SetCookies(c, sess)
	h.logger.InfoContext(c.Context(), "admin audit", slog.String("audit", "auth.login"),
		slog.Uint64("admin_id", admin.ID), slog.Bool("remember", sess.Remember), slog.String("ip", c.IP()))
	return httpadmin.OK(c, sessionJSON(&admin, sess), "Logged in.")
}

func (h *Handlers) failed(c fiber.Ctx) error {
	h.logger.InfoContext(c.Context(), "admin login failed", slog.String("ip", c.IP()))
	return invalidLogin()
}

// Logout is POST /auth/logout: destroys the session and clears the cookies.
func (h *Handlers) Logout(c fiber.Ctx) error {
	if sess := httpadmin.CurrentSession(c); sess != nil {
		if err := h.kit.Sessions().Destroy(c.Context(), sess); err != nil {
			return err
		}
	}
	h.kit.ClearCookies(c)
	httpadmin.Audit(c, h.logger, "auth.logout", "admin", httpadmin.CurrentAdmin(c).ID)
	return httpadmin.OK(c, nil, "Logged out.")
}

// Me is GET /auth/me: the admin and the session's CSRF token.
func (h *Handlers) Me(c fiber.Ctx) error {
	return httpadmin.OK(c, sessionJSON(httpadmin.CurrentAdmin(c), httpadmin.CurrentSession(c)))
}

// ChangePassword is PUT /auth/password {current_password, password, password_confirmation}
// (AccountController::updatePassword). Every other session of the admin is ended.
func (h *Handlers) ChangePassword(c fiber.Ctx) error {
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("current_password", "required|string"),
		validation.F("password", "required|string|min:8|max:72"),
		validation.F("password_confirmation", "nullable|string"),
	})
	if err != nil {
		return err
	}
	if msg, bad := Unconfirmed(c, data.Get); bad {
		return httpadmin.FieldError("password", msg)
	}
	admin := httpadmin.CurrentAdmin(c)
	if !CheckPassword(admin.Password, httpadmin.String(data, "current_password")) {
		return httpadmin.FieldError("current_password",
			httpadmin.Trans(c, "validation.current_password", nil))
	}
	hash, err := HashPassword(httpadmin.String(data, "password"))
	if err != nil {
		return err
	}
	if err := h.q.UpdateAdminPassword(c.Context(), store.UpdateAdminPasswordParams{
		Password: hash, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: admin.ID,
	}); err != nil {
		return err
	}
	if err := h.kit.Sessions().DestroyAll(c.Context(), admin.ID, httpadmin.CurrentSession(c)); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "auth.password_change", "admin", admin.ID)
	return httpadmin.OK(c, nil, "Password changed.")
}

// Unconfirmed is Laravel's `confirmed` rule on "password": password_confirmation must
// equal password. get reads the validated input.
func Unconfirmed(c fiber.Ctx, get func(string) (any, bool)) (string, bool) {
	p, _ := get("password")
	conf, _ := get("password_confirmation")
	ps, _ := p.(string)
	cs, _ := conf.(string)
	if ps == cs {
		return "", false
	}
	return httpadmin.Trans(c, "validation.confirmed",
		map[string]string{"attribute": httpadmin.AttributeName(c, "password")}), true
}

func sessionJSON(a *store.Admin, s *httpadmin.Session) *jsonx.OrderedMap {
	return jsonx.Obj("admin", httpadmin.AdminJSON(a), "csrf_token", s.CSRF)
}

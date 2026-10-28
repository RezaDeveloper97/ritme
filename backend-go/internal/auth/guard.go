package auth

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// The 401 error codes (re-exported from passport for callers of this package).
const (
	CodeRevoked         = passport.CodeRevoked
	CodeExpired         = passport.CodeExpired
	CodeUnauthenticated = passport.CodeUnauthenticated
)

// UnauthenticatedError is the API 401: compact {"message":"Unauthenticated.","error_code":…}
// (backend/bootstrap/app.php). It is the ONLY kind of 401 the API may answer.
type UnauthenticatedError struct{ Code string }

func (e *UnauthenticatedError) Error() string { return "auth: unauthenticated (" + e.Code + ")" }

// HTTPStatus implements httpx.StatusCoder.
func (e *UnauthenticatedError) HTTPStatus() int { return fiber.StatusUnauthorized }

// Render implements httpx.Renderer.
func (e *UnauthenticatedError) Render(c fiber.Ctx) error {
	return httpx.JSON(c, fiber.StatusUnauthorized, jsonx.Obj("message", "Unauthenticated.", "error_code", e.Code))
}

// AccessToken is the token the current request authenticated with ($user->token()).
type AccessToken struct {
	ID        string       // jti = oauth_access_tokens.id
	ClientID  string       // oauth_clients.id
	ExpiresAt sql.NullTime // oauth_access_tokens.expires_at (Tehran wall-clock)
}

// Revisions is the user's cache revision pair (user_revisions), read in the auth query (T-M9-31).
type Revisions = passport.Revisions

type localsKey int

const (
	userKey localsKey = iota
	tokenKey
	revisionsKey
)

// Guard is `auth:api`. Build one per domain with NewGuard (or MustGuard in a registrar).
type Guard struct {
	verifier *passport.Verifier
	logger   *slog.Logger
	initErr  error // set by MustGuard when the keys could not be loaded (server never starts)
}

// NewGuard loads the Passport public/private key pair from cfg.StoragePath (both files must
// exist) and returns the guard. It validates token times against the real clock.
func NewGuard(cfg *config.Config, db *sql.DB, logger *slog.Logger) (*Guard, error) {
	keys, err := passport.LoadKeys(cfg.StoragePath)
	if err != nil {
		return nil, err
	}
	return NewGuardWith(keys.Public, store.New(db), clock.Real{}, logger), nil
}

// NewGuardWith builds a guard from an already loaded public key (tests, custom clocks).
func NewGuardWith(pub *rsa.PublicKey, st passport.VerifierStore, clk clock.Clock, logger *slog.Logger) *Guard {
	if logger == nil {
		logger = slog.Default()
	}
	return &Guard{verifier: passport.NewVerifier(pub, st, clk), logger: logger}
}

// MustGuard is NewGuard for a routes_<domain>.go registrar: when the keys are missing or
// unreadable the server refuses to start (see FailStartup) and the returned guard answers
// every request with a 500.
//
// On the *fiber.App with a DB, every guard of the app shares one verifier (T-M9-09c): the keys are loaded once and
// the OAuth client cache is warmed once per process instead of once per guard. Sharing changes no rule — the token
// row is still read on every request, and a client revocation still reaches the guard within ClientCacheTTL (D-140).
func MustGuard(r fiber.Router, cfg *config.Config, db *sql.DB, logger *slog.Logger) *Guard {
	return mustGuard(r, db != nil, logger, func() (*passport.Verifier, error) {
		keys, err := passport.LoadKeys(cfg.StoragePath)
		if err != nil {
			return nil, err
		}
		return passport.NewVerifier(keys.Public, store.New(db), clock.Real{}), nil
	})
}

// mustGuard is MustGuard with the verifier builder injected (tests). share=false keeps one verifier per guard.
func mustGuard(r fiber.Router, share bool, logger *slog.Logger, build func() (*passport.Verifier, error)) *Guard {
	if logger == nil {
		logger = slog.Default()
	}
	var v *passport.Verifier
	var err error
	if app, ok := r.(*fiber.App); ok && share {
		v, err = sharedVerifier(app, logger, build)
	} else {
		v, err = build()
	}
	if err != nil {
		FailStartup(r, logger, err)
		return &Guard{initErr: err, logger: logger}
	}
	return &Guard{verifier: v, logger: logger}
}

// verifierStateKey holds the app's shared verifier in its State (same pattern as Jobs).
const verifierStateKey = "ritme.auth.verifier"

// clientWarmTimeout bounds the start-up client cache warm-up.
const clientWarmTimeout = 5 * time.Second

var verifierMu sync.Mutex

// sharedVerifier returns the app's verifier, building it and warming its client cache on first use. A build error
// (missing keys) is not stored, so every registrar still refuses to start; a warm-up error is only logged and the
// first request loads the clients as before.
func sharedVerifier(app *fiber.App, logger *slog.Logger, build func() (*passport.Verifier, error)) (*passport.Verifier, error) {
	verifierMu.Lock()
	defer verifierMu.Unlock()
	if v, ok := app.State().Get(verifierStateKey); ok {
		if pv, ok := v.(*passport.Verifier); ok {
			return pv, nil
		}
	}
	v, err := build()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), clientWarmTimeout)
	defer cancel()
	if err := v.WarmClients(ctx); err != nil {
		logger.Warn("auth: client cache warm-up failed, loading on first request", slog.String("error", err.Error()))
	}
	app.State().Set(verifierStateKey, v)
	return v, nil
}

// RequireUser is the `auth:api` middleware: Bearer header only (no cookies, no query),
// Passport-compatible validation, then the user and token are stored on the context
// (CurrentUser / CurrentToken). Every rejection is a 401 UnauthenticatedError with the
// exact Laravel error_code.
//
// Attach it per route, or on a group whose prefix is unique to your domain — a Fiber group
// middleware applies to every route under its prefix, including other domains' public ones.
func (g *Guard) RequireUser(c fiber.Ctx) error {
	s, err := g.authenticate(c.Context(), c.Get(fiber.HeaderAuthorization))
	if err != nil {
		return err
	}
	user := s.User
	c.Locals(userKey, &user)
	c.Locals(tokenKey, tokenOf(s))
	c.Locals(revisionsKey, s.Revisions)
	return c.Next()
}

// OptionalUser is `RequireUser` for public routes that still know a signed-in caller (CB-SHOP-10): no Authorization
// header → a guest (CurrentUser stays nil); a header that is present must be valid, otherwise 401 as usual.
func (g *Guard) OptionalUser(c fiber.Ctx) error {
	if c.Get(fiber.HeaderAuthorization) == "" {
		return c.Next()
	}
	return g.RequireUser(c)
}

// Authenticate validates an Authorization header value outside the middleware. Rejections
// are *UnauthenticatedError; other errors are infrastructure failures (500).
func (g *Guard) Authenticate(ctx context.Context, authorization string) (*User, *AccessToken, error) {
	s, err := g.authenticate(ctx, authorization)
	if err != nil {
		return nil, nil, err
	}
	user := s.User
	return &user, tokenOf(s), nil
}

func (g *Guard) authenticate(ctx context.Context, authorization string) (*passport.Session, error) {
	if g.initErr != nil {
		return nil, g.initErr
	}
	s, err := g.verifier.Verify(ctx, authorization)
	if err != nil {
		var ae *passport.AuthError
		if errors.As(err, &ae) {
			g.logger.DebugContext(ctx, "auth rejected", slog.String("code", ae.Code), slog.String("reason", ae.Reason))
			return nil, &UnauthenticatedError{Code: ae.Code}
		}
		return nil, err
	}
	return s, nil
}

func tokenOf(s *passport.Session) *AccessToken {
	return &AccessToken{ID: s.Token.ID, ClientID: s.Token.ClientID, ExpiresAt: s.Token.ExpiresAt}
}

// CurrentUser returns the authenticated user ($request->user()), or nil outside RequireUser.
func CurrentUser(c fiber.Ctx) *User {
	u, _ := c.Locals(userKey).(*User)
	return u
}

// CurrentUserID returns the authenticated user's id (false outside RequireUser).
func CurrentUserID(c fiber.Ctx) (uint64, bool) {
	if u := CurrentUser(c); u != nil {
		return u.ID, true
	}
	return 0, false
}

// CurrentRevisions returns the authenticated user's engine/analysis revisions (user_revisions, read in the same auth
// query — T-M9-31), 0 / 0 when the user has no row; ok is false outside RequireUser.
func CurrentRevisions(c fiber.Ctx) (Revisions, bool) {
	r, ok := c.Locals(revisionsKey).(Revisions)
	return r, ok
}

// CurrentToken returns the access token of the request, or nil outside RequireUser.
func CurrentToken(c fiber.Ctx) *AccessToken {
	t, _ := c.Locals(tokenKey).(*AccessToken)
	return t
}

// ThrottleIdentity is the ratelimit.Identity of Laravel's throttle middleware: the user id
// behind RequireUser, "" (→ client IP) otherwise.
func ThrottleIdentity(c fiber.Ctx) string {
	if id, ok := CurrentUserID(c); ok {
		return strconv.FormatUint(id, 10)
	}
	return ""
}

// TokenRevoker revokes tokens (satisfied by *store.Queries).
type TokenRevoker interface {
	RevokeAccessToken(ctx context.Context, arg store.RevokeAccessTokenParams) (int64, error)
	RevokeUserAccessTokens(ctx context.Context, arg store.RevokeUserAccessTokensParams) (int64, error)
}

// RevokeToken revokes one token (logout, refresh-session). now is the request clock
// (updated_at). Revoking is allowed only on logout, admin block, account deletion and
// refresh-session — never on a schedule.
func RevokeToken(ctx context.Context, q TokenRevoker, tokenID string, now time.Time) error {
	_, err := q.RevokeAccessToken(ctx, store.RevokeAccessTokenParams{UpdatedAt: dbTime(now), ID: tokenID})
	return err
}

// RevokeUserTokens revokes every token of a user — admin block and account deletion
// ($user->tokens()->update(['revoked' => true])). Blocked users are otherwise not checked
// by RequireUser (as in Laravel): blocking must call this.
func RevokeUserTokens(ctx context.Context, q TokenRevoker, userID uint64, now time.Time) (int64, error) {
	return q.RevokeUserAccessTokens(ctx, store.RevokeUserAccessTokensParams{
		UpdatedAt: dbTime(now),
		UserID:    sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // G115: ids fit int64
	})
}

// dbTime is a Carbon value as Eloquent writes it: 'Y-m-d H:i:s' in Asia/Tehran.
func dbTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

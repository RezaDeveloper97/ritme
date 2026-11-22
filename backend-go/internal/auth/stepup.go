package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Step-up (T-M12-06, D-213): the two account-wide actions a stolen long-lived bearer token must not reach on its own —
// GET /profile/export and DELETE /account — also need a fresh OTP. The signed-in user asks send-otp / verify-otp with
// `purpose: step_up` (the number is the token's user's, never the body's); a matching code answers a short-lived,
// single-use step-up token instead of a login. The protected route reads it from the X-Step-Up header and spends it.
//
// The token is 32 random bytes (base64url). Redis holds only sha256(token) → "step_up:<user id>:<expiry ms>", so the
// token is bound to the user and the purpose; expiry is compared against the request clock (X-Test-Now in tests),
// the key TTL only collects garbage. Consume is GET+DEL in one script: a token is spent by its first presentation,
// even by the wrong account (a leaked token is burnt, never retried).
const (
	// StepUpPurpose is the send-otp / verify-otp `purpose` that asks for a step-up token.
	StepUpPurpose = "step_up"
	// StepUpHeader carries the step-up token on the protected routes.
	StepUpHeader = "X-Step-Up"
	// StepUpTTL is how long a step-up token can be spent.
	StepUpTTL = 5 * time.Minute
	// CodeStepUpRequired is the 403 error_code of a protected route without a valid step-up token (never a 401: the
	// session is fine, clients must not sign out on it).
	CodeStepUpRequired = "step_up_required"
)

// StepUp issues and spends step-up tokens on Redis.
type StepUp struct {
	cache *cache.Client
	clock clock.Clock
}

// NewStepUp returns the store (nil cache → nil: every protected route then refuses, fail closed).
func NewStepUp(c *cache.Client) *StepUp {
	if c == nil {
		return nil
	}
	return &StepUp{cache: c, clock: clock.Real{}}
}

func (s *StepUp) key(token string) string {
	sum := sha256.Sum256([]byte(token))
	return s.cache.Key("stepup:" + hex.EncodeToString(sum[:]))
}

func stepUpValue(userID uint64, exp time.Time) string {
	return StepUpPurpose + ":" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatInt(exp.UnixMilli(), 10)
}

// Issue mints a step-up token for userID, valid StepUpTTL from now.
func (s *StepUp) Issue(ctx context.Context, userID uint64, now time.Time) (string, error) {
	if s == nil {
		return "", errors.New("auth: step-up: no cache")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: step-up token: %w", err)
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	if err := s.cache.Redis().Set(ctx, s.key(tok), stepUpValue(userID, now.Add(StepUpTTL)), StepUpTTL).Err(); err != nil {
		return "", fmt.Errorf("auth: step-up store: %w", err)
	}
	return tok, nil
}

// consumeScript is GET + DEL in one step: a token can be presented once.
var consumeScript = redis.NewScript(`
local v = redis.call('GET', KEYS[1])
if v then redis.call('DEL', KEYS[1]) end
return v
`)

// Consume spends token for userID: true only when it exists, is a step-up token of userID and has not expired at now.
// The token is deleted on any presentation.
func (s *StepUp) Consume(ctx context.Context, userID uint64, token string, now time.Time) (bool, error) {
	if s == nil || token == "" || len(token) > 128 {
		return false, nil
	}
	v, err := consumeScript.Run(ctx, s.cache.Redis(), []string{s.key(token)}).Text()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("auth: step-up consume: %w", err)
	}
	parts := strings.Split(v, ":")
	if len(parts) != 3 || parts[0] != StepUpPurpose || parts[1] != strconv.FormatUint(userID, 10) {
		return false, nil
	}
	exp, perr := strconv.ParseInt(parts[2], 10, 64)
	return perr == nil && exp > now.UnixMilli(), nil
}

// Require is the middleware of a step-up route; mount it after RequireUser. A missing, unknown, expired, spent or
// another account's token answers 403 step_up_required; a valid one is spent and the request goes on.
func (s *StepUp) Require(c fiber.Ctx) error {
	uid, ok := CurrentUserID(c)
	if !ok {
		return &UnauthenticatedError{Code: CodeUnauthenticated}
	}
	base := clock.Clock(clock.Real{})
	if s != nil {
		base = s.clock
	}
	ok, err := s.Consume(c.Context(), uid, strings.TrimSpace(c.Get(StepUpHeader)), clock.FromContext(c, base).Now())
	if err != nil {
		return err
	}
	if !ok {
		return httpx.Fail(fiber.StatusForbidden, StepUpMessage("required", i18n.Locale(c)), "error_code", CodeStepUpRequired)
	}
	return c.Next()
}

// The step-up copy is data: lang/<code>/stepup.json (a language without it falls back to English).
//
//go:embed lang/*/*.json
var stepUpLangFS embed.FS

var stepUpTranslator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(stepUpLangFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err)
	}
	return t
})

// StepUpMessage is the step-up line for key ("required", "verified") in locale.
func StepUpMessage(key, locale string) string {
	return stepUpTranslator().Trans("stepup."+key, nil, locale)
}

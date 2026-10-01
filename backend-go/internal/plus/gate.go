package plus

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Plus gating (B-N2-06). A Plus feature's route adds Gate.Require(key) after the locale and auth middleware; it
// checks the entitlement without counting a use. The handler counts the use with Service.Consume once the work
// succeeded and maps a Consume error through GateError (a quota can run out between the check and the use).
//
// Responses (httpx envelope, never 401 — clients wipe the session only on auth 401s):
//
//	402 {success:false, message, error_code:"plus_required", feature, reason:"locked"|"quota", limit, resets_at}
//	    the feature is not in the user's tier, or a free allowance is spent (upgrading helps)
//	429 {success:false, message, error_code:"plus_quota_exceeded", feature, reason:"quota", limit, resets_at}
//	    a Plus/trial user spent a monthly quota (plus.lab_ai 10/month); it renews at resets_at
const (
	CodePlusRequired  = "plus_required"
	CodeQuotaExceeded = "plus_quota_exceeded"
)

// Gate builds the Plus-gating middleware.
type Gate struct {
	svc   *Service
	clock clock.Clock
}

// NewGate wires the gate; base is the fallback clock (X-Test-Now pins it per request in tests).
func NewGate(svc *Service, base clock.Clock) *Gate { return &Gate{svc: svc, clock: base} }

// Require lets the request through only while the current user may use key now. Unknown keys panic at startup
// (a typo must not silently open or close a feature).
func (g *Gate) Require(key Key) fiber.Handler {
	if _, ok := Lookup(key); !ok {
		panic("plus: Require of unknown entitlement " + string(key))
	}
	return func(c fiber.Ctx) error {
		userID, ok := auth.CurrentUserID(c)
		if !ok {
			return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
		}
		now := clock.FromContext(c, g.clock).Now().In(civildate.Tehran).Truncate(time.Second)
		e, tier, err := g.svc.entitlement(c, userID, key, now)
		if err != nil {
			return err
		}
		if e.Allowed {
			return c.Next()
		}
		return Denied(e, tier, now, i18n.Locale(c))
	}
}

// GateError maps a Service.Consume error to the gating response (other errors pass through unchanged).
func (g *Gate) GateError(c fiber.Ctx, err error, e Entitlement) error {
	if !errors.Is(err, ErrNotEntitled) && !errors.Is(err, ErrQuotaExceeded) {
		return err
	}
	userID, _ := auth.CurrentUserID(c)
	now := clock.FromContext(c, g.clock).Now().In(civildate.Tehran).Truncate(time.Second)
	tier, _, _, terr := tierOf(c, g.svc.q, userID, now)
	if terr != nil {
		return terr
	}
	return Denied(e, tier, now, i18n.Locale(c))
}

// Denied is the response for an entitlement the user may not use now (see the package comment of the gate).
func Denied(e Entitlement, tier Tier, now time.Time, locale string) error {
	reason, limit, resets := "locked", any(nil), any(nil)
	if e.Limit > 0 {
		reason, limit, resets = "quota", e.Limit, iso(resetsAt(now))
	}
	if reason == "quota" && tier != TierFree {
		return httpx.Fail(fiber.StatusTooManyRequests, T("errors.quota_exceeded", locale),
			"error_code", CodeQuotaExceeded, "feature", string(e.Key), "reason", reason, "limit", limit, "resets_at", resets)
	}
	return httpx.Fail(fiber.StatusPaymentRequired, T("errors.plus_required", locale),
		"error_code", CodePlusRequired, "feature", string(e.Key), "reason", reason, "limit", limit, "resets_at", resets)
}

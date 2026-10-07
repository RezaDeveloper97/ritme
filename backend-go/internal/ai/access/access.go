// Package access is the HTTP gate every AI feature route passes before its handler runs (B-N6-05, hardened in
// B-N6-05b):
//
//	route: locale → auth RequireUser → Guard.Chain(feature)… → handler
//	Chain: per-user throttles → [plus.Gate.Require(key)] → Require(feature) → [reserve one Plus use]
//
// The throttles come right after auth (L2), so a locked or consent-less client hammering a feature is cut off
// before any database work. Require checks, in order:
//
//  1. the AI client exists (AI_PROVIDER none / no key → 503 ai_unavailable; no consent prompt for a feature that
//     cannot run),
//  2. the user accepted the text in force of the feature's consent (internal/consent; 403 consent_required with
//     the code, version and reason). A store error fails closed (500). Policies marked ExternalOnly (voice log,
//     whose UI has no consent sheet yet) skip this while the provider is the in-process fake — nothing leaves
//     the server then,
//  3. the global daily cost cap is not spent (internal/ai/usage.Budget; 503 ai_budget_exhausted, fail closed),
//  4. the user's own daily cost cap is not spent (AI_USER_DAILY_COST_CAP_USD; 429 ai_user_budget_exhausted, fail
//     closed),
//  5. the user has fewer than MaxConcurrentPerUser AI requests running on this instance (429 ai_busy) — the
//     budget checks read what was spent, so parallel calls must not all pass them at once,
//
// then puts the ai.Subject (user id for the usage log and the per-user cap, the user's name for PII redaction) on
// the request context (c.Context()), which handlers pass to the ai.Client.
//
// Plus quota is reserve-first (M3): the Plus gate answers 402/429 early without counting, then — after every
// other check passed — one use is counted atomically (plus.Service.Consume) before the handler runs, so parallel
// requests can never exceed the quota. The use is given back (Reservation.Refund) when the handler returns an
// error or an error status — the provider failed, the upload was invalid, a budget ran out — and kept otherwise,
// also for a silent recording or an empty answer (the provider was paid either way). A streaming handler that
// has already returned refunds explicitly: ReservationFrom(ctx).Refund() when the stream failed before its first
// delta.
package access

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"path"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/plus"
)

// Policy is what a feature needs before it may call a provider.
type Policy struct {
	Feature ai.Feature
	Consent string // consent code (internal/consent)
	// ExternalOnly: the consent is required only while calls leave the server (a real provider).
	ExternalOnly bool
	Plus         plus.Key // "" = not Plus-gated here (the feature gates itself)
	// Throttles are the per-user request limits Chain mounts first (B-N6-05b, M2 / L2).
	Throttles []Throttle
}

// Throttle is one per-user request limit of a feature: at most Max requests per Window, on its own counter Name.
type Throttle struct {
	Name   string
	Max    int
	Window time.Duration
}

// Voice logging throttles (B-N3-05 numbers; internal/voicelog re-exports them).
const (
	VoiceBurstMax  = 6  // per minute
	VoiceHourlyMax = 60 // per hour
)

// MaxConcurrentPerUser is how many AI requests one user may have running at once on one instance (M3).
const MaxConcurrentPerUser = 2

// policies is the single source of the feature → consent / Plus / throttle mapping.
var policies = []Policy{
	{Feature: ai.FeatureVoiceLog, Consent: consent.AIVoiceLog, ExternalOnly: true, Plus: plus.VoiceLog, Throttles: []Throttle{
		{Name: "voice-burst", Max: VoiceBurstMax, Window: time.Minute}, {Name: "voice-hourly", Max: VoiceHourlyMax, Window: time.Hour},
	}},
	{Feature: ai.FeatureLabAnalysis, Consent: consent.AILabAnalysis, Plus: plus.LabAI, Throttles: []Throttle{
		{Name: "ai-lab-burst", Max: 4, Window: time.Minute}, {Name: "ai-lab-hourly", Max: 20, Window: time.Hour},
	}},
	{Feature: ai.FeatureAssistant, Consent: consent.AIAssistant, Plus: plus.AssistantUnlimited, Throttles: []Throttle{
		{Name: "ai-assistant-burst", Max: 10, Window: time.Minute}, {Name: "ai-assistant-hourly", Max: 120, Window: time.Hour},
	}},
	// CB-REC-02: record document extraction (plus.doc_ai, 20 / month; free users fill the fields in by hand).
	{Feature: ai.FeatureDocExtract, Consent: consent.AIDocuments, Plus: plus.DocAI, Throttles: []Throttle{
		{Name: "ai-doc-burst", Max: 4, Window: time.Minute}, {Name: "ai-doc-hourly", Max: 20, Window: time.Hour},
	}},
}

// PolicyFor returns the policy of feature.
func PolicyFor(f ai.Feature) (Policy, bool) {
	for _, p := range policies {
		if p.Feature == f {
			return p, true
		}
	}
	return Policy{}, false
}

// Policies returns a copy of every policy.
func Policies() []Policy { return append([]Policy(nil), policies...) }

// Error codes of the gate and of ai errors mapped by Error.
const (
	CodeUnavailable     = "ai_unavailable"
	CodeFailed          = "ai_failed"
	CodeBudgetExhausted = "ai_budget_exhausted"
	CodeInvalidRequest  = "ai_invalid_request"
	// B-N6-05b
	CodeUserBudgetExhausted = "ai_user_budget_exhausted"
	CodeBusy                = "ai_busy"
	CodeTooMany             = "too_many_requests"
)

// Guard builds the AI gate.
type Guard struct {
	client   *ai.Client
	consents *consent.Service
	budget   ai.Limiter
	gate     *plus.Gate
	plus     *plus.Service
	limiter  *ratelimit.Limiter
	rejects  map[ai.Feature]ratelimit.Reject
	clock    clock.Clock
	logger   *slog.Logger
	inflight *inflight
}

// Options wires a Guard.
type Options struct {
	Client   *ai.Client       // nil = AI unavailable (every feature answers 503)
	Consents *consent.Service // required
	// Budget is the global cap (and, when it implements ai.UserLimiter, the per-user one). nil = no cap (tests);
	// production passes the usage.Budget the client uses too.
	Budget ai.Limiter
	Gate   *plus.Gate    // Chain: the early Plus check (402/429 without counting)
	Plus   *plus.Service // Chain: reserve-first quota (nil = the handler counts its use itself)
	// Limiter mounts the policies' throttles (nil = no throttles: no cache configured).
	Limiter *ratelimit.Limiter
	// Throttled overrides the 429 of a feature's throttles (voice keeps its own copy); default Throttled.
	Throttled map[ai.Feature]ratelimit.Reject
	Clock     clock.Clock // fallback when the request carries no test clock
	Logger    *slog.Logger
	// MaxConcurrent overrides MaxConcurrentPerUser (tests).
	MaxConcurrent int
}

// NewGuard wires the guard.
func NewGuard(o Options) *Guard {
	if o.Clock == nil {
		o.Clock = clock.Real{}
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.MaxConcurrent <= 0 {
		o.MaxConcurrent = MaxConcurrentPerUser
	}
	return &Guard{client: o.Client, consents: o.Consents, budget: o.Budget, gate: o.Gate, plus: o.Plus,
		limiter: o.Limiter, rejects: o.Throttled, clock: o.Clock, logger: o.Logger,
		inflight: &inflight{max: o.MaxConcurrent, n: map[uint64]int{}}}
}

// Chain is the feature's middleware after auth: its throttles, the Plus check (when the policy has a key and a
// gate is wired), Require(feature) and the Plus reservation (when a plus.Service is wired).
func (g *Guard) Chain(feature ai.Feature) []any {
	p := mustPolicy(feature)
	var out []any
	if g.limiter != nil {
		reject := g.rejects[feature]
		if reject == nil {
			reject = Throttled
		}
		for _, th := range p.Throttles {
			out = append(out, g.limiter.NamedWith(th.Name, th.Max, th.Window, auth.ThrottleIdentity, reject))
		}
	}
	if p.Plus != "" && g.gate != nil {
		out = append(out, g.gate.Require(p.Plus))
	}
	out = append(out, g.Require(feature))
	if p.Plus != "" && g.plus != nil {
		out = append(out, g.reserve(p))
	}
	return out
}

// Throttled is the default 429 of an AI feature's throttles (localized, with the throttle headers).
func Throttled(c fiber.Ctx, r ratelimit.Rejection) error {
	e := httpx.Fail(fiber.StatusTooManyRequests, T(CodeTooMany, i18n.Locale(c)), "error_code", CodeTooMany, "retry_after", r.RetryAfter)
	for k, v := range r.Headers() {
		e = e.WithHeader(k, v)
	}
	return e
}

// inflight counts the AI requests running per user on this instance.
type inflight struct {
	mu  sync.Mutex
	max int
	n   map[uint64]int
}

func (f *inflight) acquire(id uint64) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.n[id] >= f.max {
		return false
	}
	f.n[id]++
	return true
}

func (f *inflight) release(id uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.n[id] <= 1 {
		delete(f.n, id)
		return
	}
	f.n[id]--
}

// Reservation is one Plus use counted before the provider ran (reserve-first). Refund gives it back once.
type Reservation struct {
	svc    *plus.Service
	userID uint64
	key    plus.Key
	at     time.Time
	clock  clock.Clock
	logger *slog.Logger
	once   sync.Once
}

type reservationKey struct{}

// ReservationFrom is the reservation of the request (nil when the route reserves nothing).
func ReservationFrom(ctx context.Context) *Reservation {
	r, _ := ctx.Value(reservationKey{}).(*Reservation)
	return r
}

// Refund gives the reserved use back (idempotent; safe on a nil reservation). A failed refund is logged and
// leaves the use counted — the safe side.
func (r *Reservation) Refund(ctx context.Context) {
	if r == nil {
		return
	}
	r.once.Do(func() {
		ctx = context.WithoutCancel(ctx)
		if err := r.svc.Refund(ctx, r.userID, r.key, r.at, r.clock.Now()); err != nil {
			r.logger.LogAttrs(ctx, slog.LevelError, "ai: plus refund failed", slog.String("feature", string(r.key)),
				slog.String("error", err.Error()))
		}
	})
}

// reserve counts one use of p.Plus before the handler and refunds it when the handler failed.
func (g *Guard) reserve(p Policy) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, ok := auth.CurrentUserID(c)
		if !ok {
			return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
		}
		now := clock.FromContext(c, g.clock).Now().In(civildate.Tehran).Truncate(time.Second)
		e, err := g.plus.Consume(c, userID, p.Plus, now)
		if err != nil {
			if g.gate != nil {
				return g.gate.GateError(c, err, e)
			}
			return err
		}
		res := &Reservation{svc: g.plus, userID: userID, key: p.Plus, at: now, clock: clock.FromContext(c, g.clock), logger: g.logger}
		c.SetContext(context.WithValue(c.Context(), reservationKey{}, res))
		err = c.Next()
		if err != nil || c.Response().StatusCode() >= fiber.StatusBadRequest {
			res.Refund(c.Context())
		}
		return err
	}
}

func mustPolicy(feature ai.Feature) Policy {
	p, ok := PolicyFor(feature)
	if !ok {
		panic("access: no policy for AI feature " + string(feature)) // a typo must not open a feature
	}
	return p
}

// Require is the gate of feature (see the package comment). Mount after auth RequireUser.
func (g *Guard) Require(feature ai.Feature) fiber.Handler {
	p := mustPolicy(feature)
	return func(c fiber.Ctx) error {
		user := auth.CurrentUser(c)
		if user == nil {
			return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
		}
		locale := i18n.Locale(c)
		if g.client == nil {
			return Error(ai.ErrUnavailable, locale)
		}
		if !p.ExternalOnly || g.client.External() {
			if err := g.consents.Require(c, user.ID, p.Consent); err != nil {
				return Error(err, locale)
			}
		}
		if g.budget != nil {
			if err := g.budget.Allow(c); err != nil {
				return Error(err, locale)
			}
			if ul, ok := g.budget.(ai.UserLimiter); ok {
				if err := ul.AllowUser(c, user.ID); err != nil {
					return Error(err, locale)
				}
			}
		}
		if !g.inflight.acquire(user.ID) {
			return httpx.Fail(fiber.StatusTooManyRequests, T(CodeBusy, locale), "error_code", CodeBusy).WithHeader("Retry-After", "5")
		}
		defer g.inflight.release(user.ID)
		s := ai.Subject{UserID: user.ID}
		if user.Name.Valid {
			s.Names = []string{user.Name.String}
		}
		c.SetContext(ai.WithSubject(c.Context(), s))
		return c.Next()
	}
}

// Error maps an AI platform error to its response (other errors pass through unchanged):
//
//	*consent.RequiredError → 403 consent_required {consent, version, reason}
//	ai.ErrUnavailable      → 503 ai_unavailable
//	ai.ErrBudgetExceeded   → 503 ai_budget_exhausted
//	ai.ErrUserBudgetExceeded → 429 ai_user_budget_exhausted (B-N6-05b)
//	ai.ErrUpstream         → 503 ai_failed
//	ai.ErrInvalidRequest   → 422 ai_invalid_request
func Error(err error, locale string) error {
	var req *consent.RequiredError
	switch {
	case errors.As(err, &req):
		return consent.Required(req, locale)
	case errors.Is(err, ai.ErrUnavailable):
		return httpx.Fail(fiber.StatusServiceUnavailable, T(CodeUnavailable, locale), "error_code", CodeUnavailable)
	case errors.Is(err, ai.ErrUserBudgetExceeded):
		return httpx.Fail(fiber.StatusTooManyRequests, T(CodeUserBudgetExhausted, locale), "error_code", CodeUserBudgetExhausted)
	case errors.Is(err, ai.ErrBudgetExceeded):
		return httpx.Fail(fiber.StatusServiceUnavailable, T(CodeBudgetExhausted, locale), "error_code", CodeBudgetExhausted)
	case errors.Is(err, ai.ErrUpstream):
		return httpx.Fail(fiber.StatusServiceUnavailable, T(CodeFailed, locale), "error_code", CodeFailed)
	case errors.Is(err, ai.ErrInvalidRequest):
		return httpx.Fail(fiber.StatusUnprocessableEntity, T(CodeInvalidRequest, locale), "error_code", CodeInvalidRequest)
	}
	return err
}

// The gate's messages are data: lang/<code>/access.json (English fallback).
//
//go:embed lang/*/access.json
var langFS embed.FS

var messages = sync.OnceValue(func() map[string]map[string]string {
	out := map[string]map[string]string{}
	dirs, err := fs.ReadDir(langFS, "lang")
	if err != nil {
		panic(err)
	}
	for _, d := range dirs {
		raw, err := langFS.ReadFile(path.Join("lang", d.Name(), "access.json"))
		if err != nil {
			panic(err)
		}
		m := map[string]string{}
		if err := json.Unmarshal(raw, &m); err != nil {
			panic(err) // embedded files are checked by the package tests
		}
		out[d.Name()] = m
	}
	return out
})

// T is the gate message for key in locale (English fallback).
func T(key, locale string) string {
	m := messages()
	if s, ok := m[locale][key]; ok {
		return s
	}
	if s, ok := m["en"][key]; ok {
		return s
	}
	return key
}

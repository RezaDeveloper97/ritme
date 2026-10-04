// Package access is the HTTP gate every AI feature route passes before its handler runs (B-N6-05):
//
//	route: locale → auth RequireUser → [plus.Gate.Require(policy.Plus)] → access.Guard.Require(feature) → handler
//
// (Guard.Chain(feature) returns the last two in that order.) Require checks, in order:
//
//  1. the AI client exists (AI_PROVIDER none / no key → 503 ai_unavailable; no consent prompt for a feature that
//     cannot run),
//  2. the user accepted the text in force of the feature's consent (internal/consent; 403 consent_required with
//     the code, version and reason). A store error fails closed (500). Policies marked ExternalOnly (voice log,
//     whose UI has no consent sheet yet) skip this while the provider is the in-process fake — nothing leaves
//     the server then,
//  3. the global daily cost cap is not spent (internal/ai/usage.Budget; 503 ai_budget_exhausted, fail closed),
//
// then puts the ai.Subject (user id for the usage log, the user's name for PII redaction) on the request context
// (c.Context()), which handlers pass to the ai.Client. The Plus gate checks the quota without counting; the
// handler counts a use with plus.Service.Consume after the call succeeded (as voice log does).
package access

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
)

// Policy is what a feature needs before it may call a provider.
type Policy struct {
	Feature ai.Feature
	Consent string // consent code (internal/consent)
	// ExternalOnly: the consent is required only while calls leave the server (a real provider).
	ExternalOnly bool
	Plus         plus.Key // "" = not Plus-gated here (the feature gates itself)
}

// policies is the single source of the feature → consent / Plus mapping.
var policies = []Policy{
	{Feature: ai.FeatureVoiceLog, Consent: consent.AIVoiceLog, ExternalOnly: true, Plus: plus.VoiceLog},
	{Feature: ai.FeatureLabAnalysis, Consent: consent.AILabAnalysis, Plus: plus.LabAI},
	{Feature: ai.FeatureAssistant, Consent: consent.AIAssistant, Plus: plus.AssistantUnlimited},
	// CB-REC-02 picks its Plus key (entitlements B-N2-06) when it lands; the consent is required already.
	{Feature: ai.FeatureDocExtract, Consent: consent.AIDocuments},
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
)

// Guard builds the AI gate.
type Guard struct {
	client   *ai.Client
	consents *consent.Service
	budget   ai.Limiter
	gate     *plus.Gate
}

// Options wires a Guard.
type Options struct {
	Client   *ai.Client       // nil = AI unavailable (every feature answers 503)
	Consents *consent.Service // required
	Budget   ai.Limiter       // nil = no cap (tests); production passes the usage.Budget the client uses too
	Gate     *plus.Gate       // used by Chain for policies with a Plus key
}

// NewGuard wires the guard.
func NewGuard(o Options) *Guard {
	return &Guard{client: o.Client, consents: o.Consents, budget: o.Budget, gate: o.Gate}
}

// Chain is the Plus gate (when the policy has a key and a gate is wired) followed by Require(feature).
func (g *Guard) Chain(feature ai.Feature) []any {
	p := mustPolicy(feature)
	var out []any
	if p.Plus != "" && g.gate != nil {
		out = append(out, g.gate.Require(p.Plus))
	}
	return append(out, g.Require(feature))
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
		}
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
//	ai.ErrUpstream         → 503 ai_failed
//	ai.ErrInvalidRequest   → 422 ai_invalid_request
func Error(err error, locale string) error {
	var req *consent.RequiredError
	switch {
	case errors.As(err, &req):
		return consent.Required(req, locale)
	case errors.Is(err, ai.ErrUnavailable):
		return httpx.Fail(fiber.StatusServiceUnavailable, T(CodeUnavailable, locale), "error_code", CodeUnavailable)
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

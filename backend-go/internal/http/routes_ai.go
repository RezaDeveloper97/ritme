package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/ai/usage"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/plus"
)

// aiPlatform is the AI adapter platform a feature route needs (B-N6-05): the client for the configured provider
// with the durable usage + cost log and the global + per-user daily cost caps, and the gate (throttles, consent,
// caps, concurrency, reserve-first Plus quota — B-N6-05b) its routes pass first. Features: guard.Chain(ai.Feature…) after auth, then the handler calls client.* with c.Context().
type aiPlatform struct {
	client *ai.Client
	budget *usage.Budget
	guard  *access.Guard
}

func newAIPlatform(d *Deps, gate *plus.Gate, plusSvc *plus.Service, rejects map[ai.Feature]ratelimit.Reject) aiPlatform {
	budget := usage.NewBudget(d.DB, d.Config.AI.DailyCostCapUSD, clock.Real{}, d.Logger).WithUserCap(d.Config.AI.UserDailyCostCapUSD)
	client := ai.New(ai.Deps{App: d.Config.App, Config: d.Config.AI, Logger: d.Logger,
		Recorder: usage.NewRecorder(d.DB, clock.Real{}, d.Logger), Limiter: budget})
	var limiter *ratelimit.Limiter
	if d.Cache != nil {
		limiter = ratelimit.New(d.Cache, clock.Real{})
	}
	guard := access.NewGuard(access.Options{Client: client, Consents: consent.NewService(d.DB), Budget: budget, Gate: gate,
		Plus: plusSvc, Limiter: limiter, Throttled: rejects, Logger: d.Logger})
	return aiPlatform{client: client, budget: budget, guard: guard}
}

// Versioned consents (B-N6-05, internal/consent): every consent with its text in force, one consent, accept a
// version / withdraw. Go only — no Laravel route. auth:api, localized; writes are per-user throttled.
// The B-N1-12 toggles (GET/PUT /profile/consents) keep their routes; B-N6-05b made them version-aware (an AI grant
// names the version shown, `granted` only for the version in force).
func init() {
	Register("consent", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := consent.NewHandlers(consent.NewService(d.DB), clock.Real{})
		r.Get("/api/v1/consents", locale, guard, h.List)
		r.Get("/api/v1/consents/:code", locale, guard, h.Show)
		r.Put("/api/v1/consents/:code", locale, guard, writeThrottle(d), h.Update)
	})
}

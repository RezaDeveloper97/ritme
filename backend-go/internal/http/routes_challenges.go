package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/challenges"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Challenges v2 (bloom B-N9-07, D-87), Go only: multi-day challenges with per-day tasks, eligibility by life mode and
// cycle phase, and a once-only reward (badge + free Plus days). All auth:api, localized by Accept-Language, writes
// per-user throttled; every row is scoped by the user (a foreign or hidden challenge is a uniform 404).
func init() {
	Register("challenges", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := challenges.NewHandlers(challenges.NewService(d.DB).WithEngine(engineStates(r, d)), clock.Real{})
		writes := writeThrottle(d)

		p := "/api/v1/challenges"
		r.Get(p, locale, guard, h.Index)
		r.Get(p+"/rewards", locale, guard, h.Rewards)
		r.Post(p+"/:id/join", locale, guard, writes, h.Join)
		r.Post(p+"/:id/tasks/:task/complete", locale, guard, writes, h.CompleteTask)
	})
}

package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/pregnancy/tools"
)

// Pregnancy tools (bloom B-N5-03, D-60), Go only: the kick counter (sessions whose day total lands in the existing
// /pregnancy/fetal-movement log) and the contraction timer (5-1-1 through the pregnancy alert engine). All auth:api,
// localized by Accept-Language, writes per-user throttled; every row is scoped to the user (404 otherwise).
func init() {
	Register("pregnancy_tools", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := tools.NewHandlers(tools.NewService(d.DB, pregnancyalerts.New(store.New(d.DB))), clock.Real{})
		writes := writeThrottle(d)

		k := "/api/v1/pregnancy/kick-sessions"
		r.Get(k, locale, guard, h.Kicks)
		r.Post(k, locale, guard, writes, h.StartKicks)
		r.Post(k+"/:id/kicks", locale, guard, writes, h.Kick)
		r.Delete(k+"/:id/kicks", locale, guard, writes, h.UndoKick)
		r.Post(k+"/:id/stop", locale, guard, writes, h.StopKicks)
		r.Delete(k+"/:id", locale, guard, writes, h.DestroyKicks)

		c := "/api/v1/pregnancy/contractions"
		r.Get(c, locale, guard, h.Contractions)
		r.Post(c+"/start", locale, guard, writes, h.StartContraction)
		r.Post(c+"/stop", locale, guard, writes, h.StopContraction)
		r.Get(c+"/sessions/:id", locale, guard, h.Timing)
		r.Post(c+"/sessions/:id/finish", locale, guard, writes, h.FinishTiming)
		r.Delete(c+"/sessions/:id", locale, guard, writes, h.DestroyTiming)
	})
}

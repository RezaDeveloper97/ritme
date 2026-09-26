package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	pregnancyv2 "github.com/ritme/backend-go/internal/pregnancy/v2"
	"github.com/ritme/backend-go/internal/pregnancy/v2/daylog"
)

// Pregnancy v2 (docs/pregnancy-v2/README.md), Go only: dating preview, Today, week page and
// per-week state. All auth:api, localized by Accept-Language.
func init() {
	Register("pregnancy_v2", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := pregnancyv2.NewHandlers(store.New(d.DB), clock.Real{})
		const p = "/api/v1/pregnancy/v2"

		r.Post(p+"/dating-preview", locale, guard, h.DatingPreview)
		r.Get(p+"/today", locale, guard, h.Today)
		r.Get(p+"/weeks/:n", locale, guard, h.Week)
		r.Put(p+"/weeks/:n/state", locale, guard, h.WeekState)

		dl := daylog.NewHandlers(store.New(d.DB), clock.Real{})
		r.Get(p+"/report", locale, guard, dl.Report)
		r.Get(p+"/days/:date", locale, guard, dl.Show)
		r.Put(p+"/days/:date", locale, guard, dl.Update)
	})
}

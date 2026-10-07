package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/emergency"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// EmergencyCardPublicMax is the public emergency card read limit per client IP and minute.
const EmergencyCardPublicMax = 20

// Emergency card «کارت اضطراری» (canvas-build CB-REC-03, deviations.md D-72), Go only: the owner reads and edits her
// card (settings, emergency contact, masked insurance placeholder; the health data is read live from the record)
// and may turn on a public link; anyone holding that link reads the minimal card through the public, IP-throttled
// GET /emergency-cards/{token} (unknown or disabled → 404; logs collapse /api/v1/emergency-cards/*). Owner-only
// otherwise: no id parameter, no companion route.
func init() {
	Register("emergency", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := emergency.NewHandlers(emergency.NewService(d.DB, healthrecord.NewService(d.DB, nil)), clock.Real{})
		writes := writeThrottle(d)
		public := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			public = ratelimit.New(d.Cache, clock.Real{}).Named("emergency-card", EmergencyCardPublicMax, time.Minute, nil)
		}

		const p = "/api/v1/health-record/emergency-card"
		r.Get(p, locale, guard, h.Show)
		r.Put(p, locale, guard, writes, h.Update)
		r.Post(p+"/public-link", locale, guard, writes, h.EnablePublic)
		r.Delete(p+"/public-link", locale, guard, writes, h.DisablePublic)
		r.Get("/api/v1/emergency-cards/:token", public, locale, h.Public)
	})
}

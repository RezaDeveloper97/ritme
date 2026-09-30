package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Privacy & support (B-N1-12): consents (AI lab analysis, assistant profile, anonymous stats) and «گزارش مشکل».
// Go only — no Laravel counterpart. auth:api, localized by Accept-Language; writes are per-user throttled.
func init() {
	Register("privacy", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := profile.NewPrivacyHandlers(profile.PrivacyOptions{
			Store:       profilestore.New(d.DB),
			Clock:       clock.Real{},
			StoragePath: d.Config.StoragePath,
			Logger:      d.Logger,
		})
		writes := writeThrottle(d)

		r.Get("/api/v1/profile/consents", locale, guard, h.Consents)
		r.Put("/api/v1/profile/consents", locale, guard, writes, h.UpdateConsents)
		// Atomic hourly per-user limit before the body (and any screenshot) is decoded.
		reports := profile.SupportReportLimiter(d.Cache, clock.Real{})
		r.Post("/api/v1/support/reports", locale, guard, writes, reports, h.CreateSupportReport)
	})
}

package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/pelvic"
	pelvicstore "github.com/ritme/backend-go/internal/pelvic/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Pelvic floor program (CB-PELV-01), Go only — no Laravel counterpart (deviations.md D-32). All auth:api,
// localized by Accept-Language; levels come from the catalog group `pelvic_levels` (cached reader). Health data:
// user-scoped only. Static paths before {date}.
func init() {
	Register("pelvic", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		levels := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		h := pelvic.NewHandlers(pelvic.NewService(pelvicstore.New(d.DB), levels), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/pelvic", locale, guard, h.Show)
		r.Post("/api/v1/pelvic/program", locale, guard, writes, h.StartProgram)
		r.Delete("/api/v1/pelvic/program", locale, guard, writes, h.StopProgram)
		r.Post("/api/v1/pelvic/sessions", locale, guard, writes, h.AddSession)
		r.Get("/api/v1/pelvic/diary/:date", locale, guard, h.ShowDiary)
		r.Put("/api/v1/pelvic/diary/:date", locale, guard, writes, h.UpdateDiary)
	})
}

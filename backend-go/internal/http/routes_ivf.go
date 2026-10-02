package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/ivf"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// IVF treatment (CB-IVF-01), Go only — no Laravel counterpart (deviations.md D-51). All auth:api, localized by
// Accept-Language, writes per-user throttled; lists and clinical copy come from the catalog (cached reader), the
// companion context from bloom's companion links (B-N4-02). Health data: user-scoped only. Static paths before
// params.
func init() {
	Register("ivf", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		links := companion.NewService(d.DB, clock.Real{}, companion.Options{})
		h := ivf.NewHandlers(ivf.NewService(d.DB, cat, links), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/ivf", locale, guard, h.Show)
		r.Post("/api/v1/ivf/cycles", locale, guard, writes, h.StartCycle)
		r.Put("/api/v1/ivf/cycles/current", locale, guard, writes, h.UpdateCycle)
		r.Post("/api/v1/ivf/cycles/current/outcome", locale, guard, writes, h.RecordOutcome)
		r.Get("/api/v1/ivf/meds", locale, guard, h.ShowMeds)
		r.Post("/api/v1/ivf/meds", locale, guard, writes, h.AddMed)
		r.Put("/api/v1/ivf/meds/:id", locale, guard, writes, h.UpdateMed)
		r.Delete("/api/v1/ivf/meds/:id", locale, guard, writes, h.DeleteMed)
		r.Post("/api/v1/ivf/meds/:id/doses", locale, guard, writes, h.LogDose)
		r.Delete("/api/v1/ivf/meds/:id/doses", locale, guard, writes, h.UnlogDose)
		r.Get("/api/v1/ivf/scans", locale, guard, h.ShowScans)
		r.Put("/api/v1/ivf/scans/:date", locale, guard, writes, h.SaveScan)
		r.Delete("/api/v1/ivf/scans/:date", locale, guard, writes, h.DeleteScan)
		r.Get("/api/v1/ivf/tww", locale, guard, h.ShowTWW)
		r.Put("/api/v1/ivf/tww/:date", locale, guard, writes, h.SaveMood)
	})
}

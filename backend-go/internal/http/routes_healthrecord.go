package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/resources/translations"
)

// Health record «پرونده سلامت من» (bloom B-N6-03, D-64), Go only: the owner's summary aggregated from profile, life
// profile, care, cycle engine, vitals, pregnancy / postpartum, checkups and labs, plus the user-owned blood type,
// allergies and hand-added pregnancies. auth:api, localized by Accept-Language, writes per-user throttled. Owner-only:
// no id of another user, no companion route. Health values are never logged.
func init() {
	Register("healthrecord", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		labsSvc := labs.NewService(labs.Options{ // the record's lab rows (reads only)
			DB: d.DB, Catalog: catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger), Logger: d.Logger,
		})
		h := healthrecord.NewHandlers(healthrecord.NewService(d.DB, labsSvc).
			WithBundles(i18n.NewTranslationStore(translations.FS, d.Config.StoragePath)), clock.Real{})
		writes := writeThrottle(d)

		p := "/api/v1/health-record"
		r.Get(p, locale, guard, h.Show)
		r.Put(p+"/basics", locale, guard, writes, h.UpdateBasics)
		r.Post(p+"/pregnancies", locale, guard, writes, h.StorePregnancy)
		r.Put(p+"/pregnancies/:id", locale, guard, writes, h.UpdatePregnancy)
		r.Delete(p+"/pregnancies/:id", locale, guard, writes, h.DestroyPregnancy)
	})
}

package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/sharelinks"
	sharelinkstore "github.com/ritme/backend-go/internal/sharelinks/store"
	"github.com/ritme/backend-go/resources/translations"
)

// SharedReportMax is the public report read limit per client IP and minute.
const SharedReportMax = 30

// Doctor report share links (bloom B-N6-04, D-65), Go only: the owner creates (Plus-gated, plus.PDFShare), lists and
// revokes 7-day links to an encrypted snapshot of her share-audience health record; anyone with the link reads it
// through the public, IP-throttled GET /shared-reports/{token} (410 once expired or revoked). Expired snapshots are
// wiped hourly. Tokens and report contents are never logged.
func init() {
	Register("sharelinks", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		labsSvc := labs.NewService(labs.Options{ // the report's lab rows (reads only)
			DB: d.DB, Catalog: catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger), Logger: d.Logger,
		})
		bundles := i18n.NewTranslationStore(translations.FS, d.Config.StoragePath)
		reports := healthrecord.NewService(d.DB, labsSvc).WithBundles(bundles).
			WithProviders(menopauseReportSection(d, bundles))
		svc := sharelinks.NewService(sharelinkstore.New(d.DB), reports, d.Logger).
			WithDocuments(sharelinks.NewRecordDocuments(d.DB), recordShareFiles(d)) // CB-REC-03 summaries opened by QR
		h := sharelinks.NewHandlers(svc, clock.Real{})
		gate := plus.NewGate(plus.NewService(d.DB, d.Config.Plus, nil, d.Logger), clock.Real{}) // entitlements only
		writes := writeThrottle(d)
		public := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			public = ratelimit.New(d.Cache, clock.Real{}).Named("shared-report", SharedReportMax, time.Minute, nil)
		}
		if d.DB != nil {
			ctx, cancel := context.WithCancel(context.Background())
			now := func() time.Time { return time.Now().In(civildate.Tehran) }
			auth.OnLifecycle(r, func() error { go svc.PurgeLoop(ctx, now); return nil }, cancel)
		}

		p := "/api/v1/health-record/share-links"
		r.Get(p, locale, guard, h.Index)
		r.Post(p, locale, guard, gate.Require(plus.PDFShare), writes, h.Store)
		r.Delete(p+"/:id", locale, guard, writes, h.Destroy)
		r.Get("/api/v1/shared-reports/:token", public, locale, h.Show)
	})
}

package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Record documents, extras and timeline (canvas-build CB-REC-01, deviations.md D-70), Go only, on top of bloom's
// health record (routes_healthrecord.go, D-64) and the generic file storage (routes_files.go, D-59): category counts,
// the documents + labs timeline, surgeries / family history / the allergies' emergency-card flag, and document CRUD
// with files of purpose record_document. auth:api, localized by Accept-Language, writes per-user throttled.
// Owner-only: a foreign id is the uniform 404; no companion route; health values are never logged.
func init() {
	Register("healthrecord-documents", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)

		// The same vault policy as routes_files.go (FILE_KEY → LAB_FILE_KEY, development key only for local / testing
		// / contract); that file logs the key warnings. Documents only read rows, sign links and delete files here.
		current, previous, dev, missing := d.Config.Files.Resolve(d.Config.App, d.Config.LabFiles)
		if dev {
			current = files.DevKey("ritme-file-dev-key")
		}
		vault, err := files.NewVault(d.Config.StoragePath, current, previous, missing)
		if err != nil {
			panic(err) // config.Load already checked the key lengths
		}
		fileSvc := files.NewService(files.Options{DB: d.DB, Vault: vault, BaseURL: d.Config.App.URL})
		labsSvc := labs.NewService(labs.Options{ // lab sheets in the timeline and counts (reads only)
			DB: d.DB, Catalog: catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger), Logger: d.Logger,
		})
		h := healthrecord.NewDocumentHandlers(healthrecord.NewDocuments(d.DB, fileSvc, labsSvc), clock.Real{})

		const p = "/api/v1/health-record"
		r.Get(p+"/categories", locale, guard, h.Categories)
		r.Get(p+"/timeline", locale, guard, h.Timeline)
		r.Get(p+"/extras", locale, guard, h.ShowExtras)
		r.Put(p+"/extras", locale, guard, writes, h.UpdateExtras)
		r.Post(p+"/documents", locale, guard, writes, h.StoreDocument)
		r.Get(p+"/documents/:id", locale, guard, h.ShowDocument)
		r.Put(p+"/documents/:id", locale, guard, writes, h.UpdateDocument)
		r.Delete(p+"/documents/:id", locale, guard, writes, h.DestroyDocument)
	})
}

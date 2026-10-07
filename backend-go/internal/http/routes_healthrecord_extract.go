package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/healthrecord/extract"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/plus"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// Record document extraction (canvas-build CB-REC-02, deviations.md D-70), Go only, on top of the record documents
// (routes_healthrecord_documents.go). auth:api, localized, owner-only (a foreign id is the uniform 404).
// POST …/extract passes the AI gate in the B-N6-05b order: per-user throttles → plus.doc_ai (free users 402 — they
// fill the fields in by hand with PUT /health-record/documents/{id}) → versioned consent ai_documents → cost caps,
// concurrency → one reserved Plus use (given back when the extraction fails). The extraction runs as a DB-backed job
// on in-process workers started with the listener (no queue table: record_documents.extracted carries the job);
// QUEUE_CONNECTION=sync (contract stack) runs it inline. The pregnancy is re-dated only by POST …/dating {confirm}.
func init() {
	Register("healthrecord-extract", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		plusSvc := plus.NewService(d.DB, d.Config.Plus, nil, d.Logger)
		platform := newAIPlatform(d, plus.NewGate(plusSvc, clock.Real{}), plusSvc, nil)

		// The same vault policy as routes_files.go / routes_healthrecord_documents.go (that file logs key warnings).
		current, previous, dev, missing := d.Config.Files.Resolve(d.Config.App, d.Config.LabFiles)
		if dev {
			current = files.DevKey("ritme-file-dev-key")
		}
		vault, err := files.NewVault(d.Config.StoragePath, current, previous, missing)
		if err != nil {
			panic(err) // config.Load already checked the key lengths
		}
		fileSvc := files.NewService(files.Options{DB: d.DB, Vault: vault, BaseURL: d.Config.App.URL})
		labsSvc := labs.NewService(labs.Options{
			DB: d.DB, Catalog: catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger), Logger: d.Logger,
		})
		svc := extract.NewService(extract.Options{
			DB:        d.DB,
			Docs:      healthrecord.NewDocuments(d.DB, fileSvc, labsSvc),
			Files:     fileSvc,
			AI:        platform.client,
			Consents:  consent.NewService(d.DB),
			Plus:      plusSvc,
			Pregnancy: pregstore.New(d.DB),
			Logger:    d.Logger,
		})
		runner := extract.NewRunner(svc, extract.RunnerOptions{Sync: queue.ModeFromEnv() == queue.ModeSync})
		auth.OnLifecycle(r, runner.Start, runner.Shutdown)
		h := extract.NewHandlers(svc, clock.Real{})

		const p = "/api/v1/health-record/documents/:id"
		start := append([]any{guard}, platform.guard.Chain(ai.FeatureDocExtract)...)
		r.Post(p+"/extract", locale, append(start, h.Extract)...)
		r.Post(p+"/review", locale, guard, writes, h.Review)
		r.Get(p+"/dating", locale, guard, h.Dating)
		r.Post(p+"/dating", locale, guard, writes, h.ApplyDating)
		r.Delete(p+"/dating", locale, guard, writes, h.DismissDating)
	})
}

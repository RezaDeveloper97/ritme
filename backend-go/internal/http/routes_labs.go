package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/labs/files"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/plus"
)

// Lab analysis «تحلیل آزمایش» (bloom B-N6-06, internal/labs, D-61), Go only — no Laravel route. All auth:api,
// localized by Accept-Language, writes per-user throttled. POST /labs (the upload) passes the AI gate in the
// B-N6-05b order: per-user throttles → plus.lab_ai (10 / month; 402 / 429 before anything is read) → versioned
// consent ai_lab_analysis, global + per-user cost caps, concurrency → one reserved Plus use (refunded on failure).
// Sheets are stored AES-256-GCM encrypted under STORAGE_PATH/app/private/labs (LAB_FILE_KEY; production without it
// answers 503). Extraction and interpretation run as DB-backed jobs (lab_jobs) on in-process workers started with
// the listener; QUEUE_CONNECTION=sync (contract stack) runs them inline.
func init() {
	Register("labs", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(languages)
		writes := writeThrottle(d)
		plusSvc := plus.NewService(d.DB, d.Config.Plus, nil, d.Logger) // entitlements, counters, refunds
		platform := newAIPlatform(d, plus.NewGate(plusSvc, clock.Real{}), plusSvc, nil)

		disabled := d.Config.LabFiles.Missing(d.Config.App)
		if disabled {
			d.Logger.Error("LAB_FILE_KEY is not set in production: lab uploads and files are disabled (503)")
		}
		box, err := files.New(d.Config.StoragePath, d.Config.LabFiles.Key, d.Config.LabFiles.PreviousKeys, disabled)
		if err != nil {
			panic(err) // config.Load already checked the key length
		}
		svc := labs.NewService(labs.Options{
			DB:       d.DB,
			Files:    box,
			AI:       platform.client,
			Consents: consent.NewService(d.DB),
			Plus:     plusSvc,
			Catalog:  catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger),
			Logger:   d.Logger,
		})
		runner := labs.NewRunner(svc, labs.RunnerOptions{Sync: queue.ModeFromEnv() == queue.ModeSync, StoragePath: d.Config.StoragePath})
		auth.OnLifecycle(r, runner.Start, runner.Shutdown)
		h := labs.NewHandlers(svc, languages, clock.Real{})

		const p = "/api/v1/labs"
		upload := append([]any{guard}, platform.guard.Chain(ai.FeatureLabAnalysis)...)
		r.Get(p, locale, guard, h.Index)
		r.Post(p, locale, append(upload, h.Upload)...)
		r.Post(p+"/manual", locale, guard, writes, h.StoreManual)
		r.Get(p+"/trends", locale, guard, h.Trends)
		r.Get(p+"/markers", locale, guard, h.Catalog)
		r.Get(p+"/:id", locale, guard, h.Show)
		r.Put(p+"/:id", locale, guard, writes, h.Update)
		r.Delete(p+"/:id", locale, guard, writes, h.Destroy)
		r.Get(p+"/:id/status", locale, guard, h.Status)
		r.Post(p+"/:id/verify", locale, guard, writes, h.Verify)
		r.Post(p+"/:id/feedback", locale, guard, writes, h.Feedback)
		r.Post(p+"/:id/markers", locale, guard, writes, h.StoreMarker)
		r.Get(p+"/:id/markers/:mid", locale, guard, h.ShowMarker)
		r.Put(p+"/:id/markers/:mid", locale, guard, writes, h.UpdateMarker)
		r.Delete(p+"/:id/markers/:mid", locale, guard, writes, h.DestroyMarker)
		r.Get(p+"/:id/files/:fid", locale, guard, h.File)
		r.Delete(p+"/:id/files/:fid", locale, guard, writes, h.DestroyFile)
	})
}

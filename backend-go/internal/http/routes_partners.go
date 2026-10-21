package http

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/labs/files"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/partners"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// labImporter copies a partner's result into labs through the lab upload flow (encrypted sheet, extraction, verify).
type labImporter struct{ svc *labs.Service }

// ImportResult implements partners.Importer.
func (l labImporter) ImportResult(ctx context.Context, userID uint64, m partners.ImportMeta, pdf []byte, locale string, now time.Time) (uint64, error) {
	id, err := l.svc.CreateUpload(ctx, userID, labs.Meta{Category: m.Category, Title: m.Title},
		[]labs.Page{{Data: pdf, MIME: "application/pdf"}}, nil, locale, now)
	if errors.Is(err, labs.ErrStorage) {
		return 0, partners.ErrStorage
	}
	return id, err
}

// partnersService builds the service shared by the owner routes and the admin queue.
func partnersService(d *Deps) *partners.Service {
	box, err := loss.NewNoteBox(d.Config.PrivateNotes.Key, d.Config.PrivateNotes.PreviousKeys, d.Config.PrivateNotes.Missing(d.Config.App))
	if err != nil {
		panic(err) // config.Load already checked the key length
	}
	disabled := d.Config.LabFiles.Missing(d.Config.App)
	labBox, err := files.New(d.Config.StoragePath, d.Config.LabFiles.Key, d.Config.LabFiles.PreviousKeys, disabled)
	if err != nil {
		panic(err)
	}
	labSvc := labs.NewService(labs.Options{DB: d.DB, Files: labBox, Logger: d.Logger}) // jobs run on the labs runner (DB-backed)
	return partners.NewService(partners.Options{DB: d.DB, Sealer: box, Importer: labImporter{svc: labSvc}, Logger: d.Logger})
}

// Home lab test and screening booking requests (bloom B-N10-04, internal/partners, D-94), Go only, auth:api, localized,
// writes per-user throttled. GET /partners/lab-services is the active catalog (the Today cards exist only while one
// is active; never computed from health data); requests are owner-only (another user's id answers 404). Personal
// fields are sealed at rest and purged 30 days after the request closes; results enter labs only with consent.
func init() {
	Register("partners", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		svc := partnersService(d)
		if d.DB != nil {
			mustJobs(svc.RegisterJobs(auth.Jobs(r, d.Cache, d.Logger), tehranNow))
		}
		h := partners.NewHandlers(svc, clock.Real{})

		const p = "/api/v1/partners"
		r.Get(p+"/lab-services", locale, guard, h.Services)
		r.Get(p+"/lab-requests", locale, guard, h.Index)
		r.Post(p+"/lab-requests", locale, guard, writes, h.Store)
		r.Get(p+"/lab-requests/:id", locale, guard, h.Show)
		r.Post(p+"/lab-requests/:id/cancel", locale, guard, writes, h.Cancel)
		r.Post(p+"/lab-requests/:id/import-result", locale, guard, writes, h.ImportResult)
	})
	Register("admin_partners", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		partners.NewAdmin(partnersService(d), d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}

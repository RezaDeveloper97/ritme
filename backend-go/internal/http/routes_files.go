package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// Generic encrypted file storage (CB-CORE-05, internal/files, deviations.md D-59), Go only — no Laravel route.
// Uploads / reads / deletes are auth:api and owner-scoped; the signed download and the public image are
// unauthenticated (the HMAC link or the public purpose + random name is the credential) and IP-throttled.
// Keys: FILE_KEY, else LAB_FILE_KEY; a public development key only for APP_ENV local / testing / contract;
// anywhere else without a key every route answers 503 (fail closed). docs/canvas-build/files.md.
func init() {
	Register("files", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(languages)
		writes := writeThrottle(d)
		anon := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			anon = ratelimit.New(d.Cache, clock.Real{}).Named("files-anon", 300, time.Minute, auth.ThrottleIdentity)
		}

		current, previous, dev, missing := d.Config.Files.Resolve(d.Config.App, d.Config.LabFiles)
		if dev {
			current = files.DevKey("ritme-file-dev-key")
		}
		switch {
		case missing:
			d.Logger.Error("FILE_KEY / LAB_FILE_KEY is not set: file uploads and downloads are disabled (503)")
		case len(d.Config.Files.Key) == 0 && !dev:
			d.Logger.Warn("FILE_KEY is not set: generic files sealed with LAB_FILE_KEY")
		}
		vault, err := files.NewVault(d.Config.StoragePath, current, previous, missing)
		if err != nil {
			panic(err) // config.Load already checked the key lengths
		}
		svc := files.NewService(files.Options{DB: d.DB, Vault: vault, BaseURL: d.Config.App.URL})
		h := files.NewHandlers(svc, clock.Real{})
		if d.Config.StoragePath != "" && d.DB != nil {
			ctx, cancel := context.WithCancel(context.Background())
			auth.OnLifecycle(r, func() error { go svc.SweepLoop(ctx); return nil }, cancel) // orphan blobs, every 6 h
		}

		const p = "/api/v1/files"
		r.Post(p, locale, guard, writes, h.Upload)
		r.Get(p+"/public/:id/:name", anon, locale, h.Public)
		r.Get(p+"/:id/download", anon, locale, h.Download)
		r.Get(p+"/:id", locale, guard, h.Show)
		r.Delete(p+"/:id", locale, guard, writes, h.Destroy)
	})
}

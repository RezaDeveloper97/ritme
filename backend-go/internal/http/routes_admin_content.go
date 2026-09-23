package http

import (
	"github.com/gofiber/fiber/v3"

	admincontent "github.com/ritme/backend-go/internal/admin/content"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/languages"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/admin/messages"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	langfs "github.com/ritme/backend-go/resources/lang"
	"github.com/ritme/backend-go/resources/translations"
)

// Admin API II (T-M2-21): content CRUD and uploads, smart messages, languages and the
// translation editor under /api/admin/v1 — see docs/go-migration/admin-api.md. Uploads
// and language files are written to the backend-storage volume (STORAGE_PATH), which
// must therefore be mounted read-write for this service.
func init() {
	Register("admin_content", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		route := func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}

		admincontent.New(admincontent.Deps{
			DB: d.DB, Disk: media.NewDisk(d.Config.StoragePath), Optimizer: media.DefaultOptimizer,
			AppURL: d.Config.App.URL, Logger: d.Logger,
		}).Routes(route, kit)

		messages.New(d.DB, d.Logger).Routes(route, kit)

		laravel, err := languages.NewLaravelCache(d.Cache)
		if err != nil {
			d.Logger.Error("admin languages: Laravel cache flush disabled", "error", err.Error())
			laravel = nil
		}
		languages.New(languages.Deps{
			DB:       d.DB,
			Registry: i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger),
			Bundles:  languages.NewBundlesFromSeed(translations.FS, d.Config.StoragePath, langfs.FS),
			Laravel:  laravel,
			Logger:   d.Logger,
		}).Routes(route, kit)
	})
}

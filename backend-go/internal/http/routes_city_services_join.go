package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cityservices/join"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// Directory business applications — the 4-step «ثبت مجموعه» join (canvas CB-DIR-04, internal/cityservices/join),
// Go only, next to bloom's city services (routes_city_services.go, untouched). auth:api, localized, writes per-user
// throttled; uploads (photos / private documents through internal/files, same vault policy as routes_files.go) and
// submits have their own per-user limits. Owner-only: another account's application is the uniform 404.
const (
	// JoinSubmitMax submits per user per hour (a submit sends an SMS).
	JoinSubmitMax = 5
	// JoinUploadMax photo / document uploads per user per hour (each is decoded and re-encoded or encrypted).
	JoinUploadMax = 60
)

func init() {
	Register("city_services_join", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		submits, uploads := writes, writes
		if d.Cache != nil {
			rl := ratelimit.New(d.Cache, clock.Real{})
			reject := func(c fiber.Ctx, rj ratelimit.Rejection) error {
				return join.TooManySubmits(i18n.Locale(c), rj.RetryAfter, rj.Headers())
			}
			submits = rl.NamedWith("city-service-apply-submit", JoinSubmitMax, time.Hour, auth.ThrottleIdentity, reject)
			uploads = rl.NamedWith("city-service-apply-upload", JoinUploadMax, time.Hour, auth.ThrottleIdentity, reject)
		}

		// The same vault policy as routes_files.go (FILE_KEY → LAB_FILE_KEY, development key only for local / testing
		// / contract; fail closed elsewhere → 503). That file logs the key warnings.
		current, previous, dev, missing := d.Config.Files.Resolve(d.Config.App, d.Config.LabFiles)
		if dev {
			current = files.DevKey("ritme-file-dev-key")
		}
		vault, err := files.NewVault(d.Config.StoragePath, current, previous, missing)
		if err != nil {
			panic(err) // config.Load already checked the key lengths
		}
		fileSvc := files.NewService(files.Options{DB: d.DB, Vault: vault, BaseURL: d.Config.App.URL})
		svc := join.NewService(d.DB, fileSvc, join.NewNotifier("fake", d.Logger), d.Logger)
		h := join.NewHandlers(svc, clock.Real{})

		const p = "/api/v1/city-services/applications"
		r.Get(p, locale, guard, h.Index)
		r.Post(p, locale, guard, writes, h.Store)
		r.Get(p+"/:id", locale, guard, h.Show)
		r.Delete(p+"/:id", locale, guard, writes, h.Destroy)
		r.Get(p+"/:id/status", locale, guard, h.Status)
		r.Put(p+"/:id/steps/:step", locale, guard, writes, h.SaveStep)
		r.Post(p+"/:id/photos", locale, guard, writes, uploads, h.AddPhoto)
		r.Delete(p+"/:id/photos/:fileId", locale, guard, writes, h.RemovePhoto)
		r.Put(p+"/:id/documents/:slot", locale, guard, writes, uploads, h.PutDocument)
		r.Delete(p+"/:id/documents/:slot", locale, guard, writes, h.RemoveDocument)
		r.Post(p+"/:id/submit", locale, guard, writes, submits, h.Submit)
	})
}

package http

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/accountexport"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/notify"
	"github.com/ritme/backend-go/internal/profile"
)

// Profile / account (backend/routes/api.php, ProfileController). Middleware per route.
func init() {
	Register("profile", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		// User decision 2026-10-12 (D-96): the export carries the canvas health data; sealed text (insurance,
		// loss note, shop buyer PII) is opened for the owner with the PRIVATE_NOTE_KEY box (null without a key).
		box, err := loss.NewNoteBox(d.Config.PrivateNotes.Key, d.Config.PrivateNotes.PreviousKeys, d.Config.PrivateNotes.Missing(d.Config.App))
		if err != nil {
			panic(err) // config.Load already checked the key length
		}
		h := profile.NewHandlers(profile.Options{
			DB:       d.DB,
			Telegram: notify.NewTelegram(d.Config.Telegram, &http.Client{}, d.Logger),
			Debug:    d.Config.App.Debug,
			// B-N1-12: account deletion also removes support-report screenshots.
			StoragePath:    d.Config.StoragePath,
			Logger:         d.Logger,
			ExportSections: accountexport.Sections(d.DB, box),
			// T-M12-13: account deletion purges the user's Redis keys (engine cache, idempotency, message state).
			Cache: d.Cache,
		})

		r.Get("/api/v1/profile", locale, guard, h.Show)
		r.Post("/api/v1/profile", locale, guard, h.Store)
		r.Get("/api/v1/profile/export", locale, guard, h.Export)
		r.Delete("/api/v1/account", locale, guard, h.DestroyAccount)
	})
}

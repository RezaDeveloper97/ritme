package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/profile"
)

// Pregnancy loss path (CB-LOSS-01), Go only — no Laravel counterpart (deviations.md D-52). All auth:api,
// localized by Accept-Language, writes per-user throttled. Recording a loss durably stops pregnancy content
// (internal/pregnancy.EndForLoss + paused pregnancy reminders); copy comes from the catalog (loss_* groups). The
// private note is AES-256-GCM encrypted with PRIVATE_NOTE_KEY (missing outside local/testing → the note routes 503).
func init() {
	Register("loss", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		disabled := d.Config.PrivateNotes.Missing(d.Config.App)
		if disabled {
			d.Logger.Error("PRIVATE_NOTE_KEY is not set (only local/testing may use the development key): private notes are disabled (503)")
		}
		notes, err := loss.NewNoteBox(d.Config.PrivateNotes.Key, d.Config.PrivateNotes.PreviousKeys, disabled)
		if err != nil {
			panic(err) // config.Load already checked the key length
		}
		h := loss.NewHandlers(loss.NewService(loss.Deps{
			DB:      d.DB,
			Catalog: catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger),
			Modes:   profile.NewOnboardingHandlers(d.DB, clock.Real{}),
			Notes:   notes,
			Logger:  d.Logger,
		}), clock.Real{})

		const p = "/api/v1/loss"
		r.Get(p, locale, guard, h.Show)
		r.Post(p, locale, guard, writes, h.Store)
		r.Delete(p, locale, guard, writes, h.Destroy)
		r.Put(p+"/followup", locale, guard, writes, h.Followup)
		r.Post(p+"/moods", locale, guard, writes, h.StoreMood)
		r.Get(p+"/note", locale, guard, h.ShowNote)
		r.Put(p+"/note", locale, guard, writes, h.UpdateNote)
		r.Delete(p+"/note", locale, guard, writes, h.DestroyNote)
		r.Put(p+"/next-step", locale, guard, writes, h.NextStep)
	})
}

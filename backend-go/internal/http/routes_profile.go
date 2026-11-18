package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/accountexport"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/notify"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/profile"
)

// T-M12-06 (S-10/S-11): the full export and the account deletion are throttled per user and need a fresh OTP step-up
// (X-Step-Up, auth.StepUp; D-213), so a stolen long-lived token can neither pull every record nor erase the account.
const (
	accountExportPerHour = 3
	accountDeletePerHour = 3
	// exportConcurrency bounds the exports one process builds at once (each one reads every user table): a burst
	// from many accounts answers 429 export_busy instead of piling heavy queries onto the database.
	exportConcurrency = 4
	exportBusyRetry   = 30 // seconds
)

// exportSlots is the process-wide export semaphore: a full one refuses at once (never queues a request).
func exportSlots(n int) fiber.Handler {
	slots := make(chan struct{}, n)
	return func(c fiber.Ctx) error {
		select {
		case slots <- struct{}{}:
		default:
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(exportBusyRetry))
			return httpx.Fail(fiber.StatusTooManyRequests, "Too Many Attempts.", "error_code", "export_busy")
		}
		defer func() { <-slots }()
		return c.Next()
	}
}

// accountThrottle is a named per-user limiter (n per hour); no Redis (tests only) → none.
func accountThrottle(lim *ratelimit.Limiter, name string, n int) fiber.Handler {
	if lim == nil {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	return lim.Named(name, n, time.Hour, auth.ThrottleIdentity)
}

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
		var lim *ratelimit.Limiter
		if d.Cache != nil {
			lim = ratelimit.New(d.Cache, clock.Real{})
		}
		stepUp := auth.NewStepUp(d.Cache).Require // nil store → always 403 step_up_required (fail closed)
		// Order: the slot and the throttle come before the step-up, so a refused request never spends the token.
		r.Get("/api/v1/profile/export", locale, guard, exportSlots(exportConcurrency),
			accountThrottle(lim, "account_export", accountExportPerHour), stepUp, h.Export)
		r.Delete("/api/v1/account", locale, guard,
			accountThrottle(lim, "account_delete", accountDeletePerHour), stepUp, h.DestroyAccount)
	})
}

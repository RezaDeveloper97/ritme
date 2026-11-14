package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// AI credits (goose 00125): the user's balance and usage. Go only — no Laravel route. auth:api, localized, reads
// only: credits are spent by the AI gate (internal/ai/access, mounted by each AI feature's routes) and bought as
// credit packs in the plus domain (GET /plus/credit-packs, POST /plus/credit-packs/checkout → POST /plus/verify).
//
// At start-up the feature slots declared in code get their ai_features row (credits.Catalog.Sync), so a new AI
// feature shows up in the admin panel as soon as its code is deployed.
func init() {
	Register("credits", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		platform := newAIPlatform(d, nil)
		if d.DB != nil { // route wiring without a database (cmd/api unit tests) has nothing to sync
			go func() { // off the registration path: a slow or failed sync never delays or stops the API
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := platform.catalog.Sync(ctx, clock.Real{}.Now()); err != nil {
					d.Logger.ErrorContext(ctx, "credits: feature slot sync failed", "error", err.Error())
				}
			}()
		}
		h := credits.NewHandlers(platform.credits, platform.catalog, platform.client.External(), clock.Real{})

		r.Get("/api/v1/ai/credits", locale, guard, h.Show)
		r.Get("/api/v1/ai/credits/usage", locale, guard, h.Usage)
	})
}

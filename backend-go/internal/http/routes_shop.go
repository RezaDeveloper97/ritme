package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/shop"
	"github.com/ritme/backend-go/internal/teen"
)

// ShopClickMax is the per-user request limit per minute of POST /shop/products/:id/click.
const ShopClickMax = 30

// Shop catalog (bloom B-N10-03, D-91), Go only: categories and link-out products read-only for the app, plus the
// per-product daily click counter (no user id stored). Teen-mode accounts get no shop (CB-TEEN-01 policy).
func init() {
	Register("shop", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := shop.NewHandlers(shop.NewStore(d.DB), teen.NewPolicy(d.DB), d.Config.App.URL, clock.Real{})
		throttle := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			throttle = ratelimit.New(d.Cache, clock.Real{}).
				NamedWith("shop-click", ShopClickMax, time.Minute, auth.ThrottleIdentity, writeThrottled)
		}
		r.Get("/api/v1/shop/categories", locale, guard, h.Categories)
		r.Get("/api/v1/shop/products", locale, guard, h.Products)
		r.Get("/api/v1/shop/products/:id", locale, guard, h.Product)
		r.Post("/api/v1/shop/products/:id/click", locale, guard, throttle, h.Click)
	})
}

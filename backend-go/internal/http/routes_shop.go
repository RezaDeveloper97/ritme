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
	"github.com/ritme/backend-go/internal/shop/catalog"
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
		// CB-SHOP-02 (additive): the internal catalog read API — homes, filtered list, product detail with variants /
		// stock / reviews / bought-together, review list. Static /catalog paths never collide with bloom's /products/:id.
		// Teen-mode accounts get 403 teen_commercial_blocked here (bloom's link-out routes keep their 404 / empty list).
		ch := catalog.NewHandlers(catalog.NewService(d.DB), teen.NewPolicy(d.DB), d.Config.App.URL)
		r.Get("/api/v1/shop/catalog/home", locale, guard, ch.Home)
		r.Get("/api/v1/shop/catalog/products", locale, guard, ch.Products)
		r.Get("/api/v1/shop/catalog/products/:id", locale, guard, ch.Product)
		r.Get("/api/v1/shop/catalog/products/:id/reviews", locale, guard, ch.Reviews)
		r.Get("/api/v1/shop/categories", locale, guard, h.Categories)
		r.Get("/api/v1/shop/products", locale, guard, h.Products)
		r.Get("/api/v1/shop/products/:id", locale, guard, h.Product)
		r.Post("/api/v1/shop/products/:id/click", locale, guard, throttle, h.Click)
	})
}

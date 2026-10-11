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
	"github.com/ritme/backend-go/internal/shop/order"
	"github.com/ritme/backend-go/internal/teen"
)

// ShopClickMax is the per-user request limit per minute of POST /shop/products/:id/click.
const ShopClickMax = 30

// ShopDiscountMax is the per-user limit per hour of PUT /shop/cart/discount (security audit: code guessing).
const ShopDiscountMax = 10

// Shop catalog (bloom B-N10-03, D-91), Go only: categories and link-out products read-only for the app, plus the
// per-product daily click counter (no user id stored). Teen-mode accounts get no shop (CB-TEEN-01 policy).
func init() {
	Register("shop", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := shop.NewHandlers(shop.NewStore(d.DB), teen.NewPolicy(d.DB), d.Config.App.URL, clock.Real{})
		throttle := func(c fiber.Ctx) error { return c.Next() }
		// Security audit (CB-SHOP-03): every cart / address / order write shares the per-user write budget, and
		// applying a discount code has its own small hourly budget (code guessing).
		writes, discounts := writeThrottle(d), throttle
		if d.Cache != nil {
			rl := ratelimit.New(d.Cache, clock.Real{})
			throttle = rl.NamedWith("shop-click", ShopClickMax, time.Minute, auth.ThrottleIdentity, writeThrottled)
			discounts = rl.NamedWith("shop-discount", ShopDiscountMax, time.Hour, auth.ThrottleIdentity,
				func(c fiber.Ctx, rj ratelimit.Rejection) error {
					return order.TooManyDiscountTries(i18n.Locale(c), rj.RetryAfter, rj.Headers())
				})
		}
		// CB-SHOP-02 (additive): the internal catalog read API — homes, filtered list, product detail with variants /
		// stock / reviews / bought-together, review list. Static /catalog paths never collide with bloom's /products/:id.
		// Teen-mode accounts get 403 teen_commercial_blocked here (bloom's link-out routes keep their 404 / empty list).
		ch := catalog.NewHandlers(catalog.NewService(d.DB), teen.NewPolicy(d.DB), d.Config.App.URL)
		// CB-SHOP-10: the four catalog reads are public (desktop web shop, SEO/SSR) — optional auth: guests allowed, a
		// signed-in teen-mode account still gets 403, a malformed/expired token still 401.
		optional := auth.MustGuard(r, d.Config, d.DB, d.Logger).OptionalUser
		r.Get("/api/v1/shop/catalog/home", locale, optional, ch.Home)
		r.Get("/api/v1/shop/catalog/products", locale, optional, ch.Products)
		r.Get("/api/v1/shop/catalog/products/:id", locale, optional, ch.Product)
		r.Get("/api/v1/shop/catalog/products/:id/reviews", locale, optional, ch.Reviews)
		// CB-SHOP-03 (additive): server cart, discount code, addresses, COD checkout (idempotent, stock / slot guarded),
		// orders, cancel, returns. Owner-scoped (404 for anyone else's rows); teen-mode → 403 teen_commercial_blocked.
		oh := order.NewHandlers(order.NewService(d.DB), teen.NewPolicy(d.DB), order.NewFakeNotifier(d.Logger),
			d.Config.App.URL, clock.Real{}, d.Logger)
		r.Get("/api/v1/shop/cart", locale, guard, oh.Cart)
		r.Post("/api/v1/shop/cart/items", locale, guard, writes, oh.AddItem)
		r.Put("/api/v1/shop/cart/items/:variant_id", locale, guard, writes, oh.UpdateItem)
		r.Delete("/api/v1/shop/cart/items/:variant_id", locale, guard, writes, oh.RemoveItem)
		r.Put("/api/v1/shop/cart/discount", locale, guard, writes, discounts, oh.ApplyDiscount)
		r.Delete("/api/v1/shop/cart/discount", locale, guard, writes, oh.RemoveDiscount)
		r.Get("/api/v1/shop/addresses", locale, guard, oh.Addresses)
		r.Post("/api/v1/shop/addresses", locale, guard, writes, oh.StoreAddress)
		r.Put("/api/v1/shop/addresses/:id", locale, guard, writes, oh.UpdateAddress)
		r.Delete("/api/v1/shop/addresses/:id", locale, guard, writes, oh.DeleteAddress)
		r.Get("/api/v1/shop/checkout", locale, guard, oh.Checkout)
		r.Get("/api/v1/shop/orders", locale, guard, oh.Orders)
		r.Post("/api/v1/shop/orders", locale, guard, writes, oh.PlaceOrder)
		r.Get("/api/v1/shop/returns", locale, guard, oh.Returns)
		r.Get("/api/v1/shop/orders/:id", locale, guard, oh.Order)
		r.Post("/api/v1/shop/orders/:id/cancel", locale, guard, writes, oh.CancelOrder)
		r.Post("/api/v1/shop/orders/:id/returns", locale, guard, writes, oh.RequestReturn)
		// CB-SHOP-12 F-5: her review of a delivered order item (verified purchase, one per item, pending moderation).
		r.Post("/api/v1/shop/orders/:id/items/:item_id/review", locale, guard, writes, oh.CreateReview)
		r.Get("/api/v1/shop/categories", locale, guard, h.Categories)
		r.Get("/api/v1/shop/products", locale, guard, h.Products)
		r.Get("/api/v1/shop/products/:id", locale, guard, h.Product)
		r.Post("/api/v1/shop/products/:id/click", locale, guard, throttle, h.Click)
	})
}

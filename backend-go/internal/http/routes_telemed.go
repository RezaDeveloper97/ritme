package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/telemed"
)

// Doctors directory «پزشکان و ماماها» (bloom B-N7-02, D-67) and visit booking (B-N7-03, D-80), Go only: the
// directory with filters, a doctor's profile, free slots and reviews; booking = hold a slot → pay through the payments
// adapter (PAYMENT_PROVIDER, fake outside production) with the Plus visit discount → confirmed (care appointment +
// reminder), free cancel / reschedule until 2 h before, scoped data-share consent. All auth:api and localized by
// Accept-Language; writes per-user throttled; a user only ever sees her own bookings and reviews (anything else is a
// uniform 404). Lapsed holds are expired and ended visits completed by a once-a-minute sweeper.
func init() {
	Register("telemed", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		bookings := telemedBookings(d, true)
		h := telemed.NewHandlers(bookings.Directory(), clock.Real{}, d.Config.App.URL)
		bh := telemed.NewBookingHandlers(bookings, clock.Real{}, d.Config.App.URL)
		writes := writeThrottle(d)
		if d.DB != nil {
			ctx, cancel := context.WithCancel(context.Background())
			now := func() time.Time { return time.Now().In(civildate.Tehran) }
			auth.OnLifecycle(r, func() error { go bookings.SweepLoop(ctx, now); return nil }, cancel)
		}

		p := "/api/v1/telemed/doctors"
		r.Get(p, locale, guard, h.List)
		r.Get(p+"/filters", locale, guard, h.Filters)
		r.Get(p+"/:id", locale, guard, h.Show)
		r.Get(p+"/:id/slots", locale, guard, h.Slots)
		r.Get(p+"/:id/reviews", locale, guard, h.Reviews)
		r.Post(p+"/:id/reviews", locale, guard, writes, h.StoreReview)
		r.Delete(p+"/:id/reviews/:review", locale, guard, writes, h.DestroyReview)

		b := "/api/v1/telemed/bookings"
		r.Get(b, locale, guard, bh.Index)
		r.Get(b+"/quote", locale, guard, bh.Quote)
		r.Post(b, locale, guard, writes, bh.Store)
		r.Post(b+"/verify", locale, guard, writes, bh.Verify)
		r.Get(b+"/:id", locale, guard, bh.Show)
		r.Post(b+"/:id/cancel", locale, guard, writes, bh.Cancel)
		r.Post(b+"/:id/reschedule", locale, guard, writes, bh.Reschedule)
		r.Put(b+"/:id/consent", locale, guard, writes, bh.UpdateConsent)
		r.Delete(b+"/:id/consent", locale, guard, writes, bh.DestroyConsent)
	})
}

// telemedBookings builds the booking service (routes_services.go reuses it read-only for the upcoming card). With
// pay, the payments adapter is attached (nil when no provider is active: paid bookings answer 503).
func telemedBookings(d *Deps, pay bool) *telemed.Bookings {
	reader := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
	plusSvc := plus.NewService(d.DB, d.Config.Plus, nil, d.Logger) // entitlements only
	opts := telemed.BookingOptions{
		Config: d.Config.Telemed, Clock: clock.Real{}, Logger: d.Logger,
		Discount: func(ctx context.Context, userID uint64, now time.Time) (bool, error) {
			e, err := plusSvc.Entitlement(ctx, userID, plus.VisitDiscount, now)
			return e.Allowed, err
		},
	}
	if pay {
		if gw := payments.New(paymentDeps(d)); gw != nil { // a nil *Gateway must stay a nil interface
			opts.Gateway = gw
		}
	}
	return telemed.NewBookings(d.DB, reader, opts)
}

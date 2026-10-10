package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/assistant"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/telemed"
)

// AI assistant clinic «کلینیک دستیار ریتمی» (bloom B-N7-06, internal/assistant, D-83), Go only — no Laravel route.
// All auth:api, localized, writes per-user throttled. POST …/messages streams SSE: PrepareMessage (validation,
// ownership, server-side emergency check → 115 card without the model) runs before the AI gate (throttles →
// plus.assistant_unlimited → consent ai_assistant, cost caps, concurrency → one reserved Plus use); summaries pass the
// gate without the Plus reservation (they do not count against the free quota).
func init() {
	Register("assistant", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		plusSvc := plus.NewService(d.DB, d.Config.Plus, nil, d.Logger)
		platform := newAIPlatform(d, plus.NewGate(plusSvc, clock.Real{}), plusSvc, nil)
		svc := assistant.NewService(assistant.Options{
			DB:              d.DB,
			AI:              platform.client,
			Consents:        consent.NewService(d.DB),
			Profile:         assistant.RecordProfile(healthrecord.NewService(d.DB, nil)),
			Photos:          recordShareFiles(d),
			Bookings:        telemedBookings(d, false),
			BookingNotFound: telemed.ErrBookingNotFound,
			Logger:          d.Logger,
		})
		h := assistant.NewHandlers(svc, plusSvc, clock.Real{})

		const p = "/api/v1/assistant"
		chat := append([]any{guard, writes, h.PrepareMessage}, platform.guard.Chain(ai.FeatureAssistant)...)
		r.Get(p, locale, guard, h.Hub)
		r.Get(p+"/departments/:code", locale, guard, h.ShowDepartment)
		r.Get(p+"/profile", locale, guard, h.ShowProfile)
		r.Put(p+"/profile", locale, guard, writes, h.UpdateProfile)
		r.Post(p+"/photos", locale, guard, writes, h.StorePhoto)
		r.Get(p+"/conversations", locale, guard, h.Index)
		r.Post(p+"/conversations", locale, guard, writes, h.Store)
		r.Delete(p+"/conversations", locale, guard, writes, h.DestroyAll)
		r.Get(p+"/conversations/:id", locale, guard, h.Show)
		r.Delete(p+"/conversations/:id", locale, guard, writes, h.Destroy)
		r.Post(p+"/conversations/:id/messages", locale, append(chat, h.StreamMessage)...)
		r.Post(p+"/conversations/:id/summary", locale, guard, writes, platform.guard.Require(ai.FeatureAssistant), h.Summary)
		r.Post(p+"/conversations/:id/handoff", locale, guard, writes, h.Handoff)
	})
}

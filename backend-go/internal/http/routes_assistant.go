package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/assistant"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
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
		box, err := loss.NewNoteBox(d.Config.PrivateNotes.Key, d.Config.PrivateNotes.PreviousKeys, d.Config.PrivateNotes.Missing(d.Config.App))
		if err != nil {
			panic(err) // config.Load already checked the key length
		}
		// B-N7-06b: bodies and summaries sealed with the PRIVATE_NOTE_KEY box; without a key (outside local / testing)
		// every assistant route answers 503 before auth (fail closed).
		guard := assistant.SealGate(box, auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser)
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		plusSvc := plus.NewService(d.DB, d.Config.Plus, nil, d.Logger)
		platform := newAIPlatform(d, plus.NewGate(plusSvc, clock.Real{}), plusSvc, nil)
		svc := assistant.NewService(assistant.Options{
			DB:              d.DB,
			AI:              platform.client,
			Consents:        consent.NewService(d.DB),
			Profile:         assistant.RecordProfile(healthrecord.NewService(d.DB, nil).WithEngine(engineStates(r, d))), // T-M9-35
			Photos:          recordShareFiles(d),
			Bookings:        telemedBookings(d, false),
			Box:             box,
			BookingNotFound: telemed.ErrBookingNotFound,
			Logger:          d.Logger,
		})
		h := assistant.NewHandlers(svc, plusSvc, clock.Real{})

		const p = "/api/v1/assistant"
		// The emergency path answers before the AI gate's throttles: a messages throttle of its own bounds it (B-N7-06b M2).
		chat := append([]any{guard, writes, messagesThrottle(d), h.PrepareMessage}, platform.guard.Chain(ai.FeatureAssistant)...)
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
		// Summaries skip the Plus reservation but not the feature's throttles (B-N7-06b M2).
		summary := append(append([]any{guard, writes}, platform.guard.Gate(ai.FeatureAssistant)...), h.Summary)
		r.Post(p+"/conversations/:id/summary", locale, summary...)
		r.Post(p+"/conversations/:id/handoff", locale, guard, writes, h.Handoff)
	})
}

// assistantMessagesMax: every assistant message per user and minute, emergency ones included (B-N7-06b M2) — the
// emergency path answers before the AI gate's throttles.
const assistantMessagesMax = 20

func messagesThrottle(d *Deps) fiber.Handler {
	if d.Cache == nil {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	return ratelimit.New(d.Cache, clock.Real{}).
		NamedWith("assistant-messages", assistantMessagesMax, time.Minute, auth.ThrottleIdentity, access.Throttled)
}

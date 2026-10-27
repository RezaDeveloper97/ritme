package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/conditionnudges"
	"github.com/ritme/backend-go/internal/messages/menomessages"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// Smart messages (backend/routes/api.php, prefix messages, auth:api).
func init() {
	Register("messages", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := messages.NewHandlers(d.DB, clock.Real{})
		const p = "/api/v1/messages"

		r.Get(p+"/daily", locale, guard, h.Daily)
		r.Get(p+"/mode", locale, guard, h.Mode)

		// Smart messages v2 (B-N9-05, D-86): the rule engine's selection for a placement and the event log
		// (impression / click / dismiss / goal; per-user rate limit, ids and the event name only).
		r.Get(p+"/selected", locale, guard, h.Selected)
		events := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			events = ratelimit.New(d.Cache, clock.Real{}).Named("message-events", 120, time.Minute, auth.ThrottleIdentity)
		}
		r.Post(p+"/events", locale, guard, events, h.Event)

		// Heavy pain / heavy bleeding nudges of the condition programs (CB-COND-06b): Go-only, D-41.
		nudges := conditionnudges.NewHandlers(d.DB, clock.Real{}).WithEngine(engineStates(r, d))
		r.Get(p+"/nudges", locale, guard, nudges.Index)

		// Menopause alerts and stage tips (CB-MENO-12): Go-only, D-43. Facts from the menopause API's rules, texts
		// from its catalog and message_contents group menopause_message.
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		meno := menomessages.NewHandlers(d.DB, menopause.NewService(d.DB, cat), clock.Real{})
		r.Get(p+"/menopause", locale, guard, meno.Index)
	})
}

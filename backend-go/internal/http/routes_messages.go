package http

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/segments"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/challenges"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/conditionnudges"
	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/inbox"
	"github.com/ritme/backend-go/internal/messages/menomessages"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/messages/rules"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/platform/usertz"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Smart messages (backend/routes/api.php, prefix messages, auth:api).
func init() {
	Register("messages", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := messages.NewHandlers(d.DB, clock.Real{}).WithEngine(engineStates(r, d)) // T-M9-38: v6 facts under shadow|v6
		h.WithSegments(segments.NewResolver(d.DB))                                   // T-M10-32: the `segment` fact
		// T-M10-08b: the pregnancy alert backend (preg.* rules on the inbox) no longer depends on the pregnancy-tools
		// group registering; installing it twice is harmless (same pool).
		inboxalerts.Install(d.DB, d.Logger)
		const p = "/api/v1/messages"

		// T-M10-06 (D-163): the buffered event log (drained before shutdown), the shown/eligible marks, the
		// server-matched goals (events.Goal, M-17) and the rollup / retention jobs (M-13).
		mq := mstore.New(d.DB)
		writer := events.NewWriter(mq, d.Logger)
		if app, ok := r.(*fiber.App); ok {
			app.Hooks().OnPreShutdown(func() error {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				writer.Close(ctx)
				return nil
			})
		}
		marks := events.NewMarks(d.Cache)
		h.WithFeed(writer, marks, modules.NewService(d.DB, d.Cache, d.Logger))
		events.Install(events.NewGoals(mq, writer, marks, d.Logger))
		mustJobs(events.NewJobs(mq).RegisterJobs(auth.Jobs(r, d.Cache, d.Logger), tehranNow))

		// T-M10-07 (D-164): the alerts inbox — evaluated inside the feed (feed.InstallInbox) and by inbox.EvaluateNow
		// for write paths, its lifecycle API under /api/v1/alerts, the daily sweep and the hourly push slot (M-09:
		// delivered to the in-app notifications inbox with the neutral text).
		langs := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		zones := profilestore.New(d.DB)
		box := inbox.New(mq, d.DB, d.Logger).WithPreferences(zones).
			WithLanguages(func(ctx context.Context) []string { return langs.All(ctx).Codes() }).
			WithFacts(func(ctx context.Context, userID uint64, at time.Time) (rules.Facts, time.Time, error) {
				tz, err := zones.GetUserTimezone(ctx, userID)
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					return nil, time.Time{}, err
				}
				local := at.In(usertz.ZoneOf(tz))
				f, err := messages.BuildFacts(ctx, h, userID, civildate.FromTime(local))
				return f, local, err
			})
		feed.InstallInbox(box)
		// T-M10-10 (D-167): today's challenge through the daily_challenge slot (catalogue snapshot, runs off the fact
		// profile row: no query on a warm feed).
		feed.InstallChallenges(challenges.NewService(d.DB).WithEngine(engineStates(r, d)))
		inbox.Install(box)
		mustJobs(box.RegisterJobs(auth.Jobs(r, d.Cache, d.Logger), tehranNow))
		alerts := inbox.NewHandlers(box, clock.Real{})
		const ap = "/api/v1/alerts"
		r.Get(ap, locale, guard, alerts.Index)
		r.Post(ap+"/:id/ack", locale, guard, alerts.Ack)
		r.Post(ap+"/:id/visit-note", locale, guard, alerts.VisitNote)
		r.Post(ap+"/:id/dismiss", locale, guard, alerts.Dismiss)

		r.Get(p+"/daily", locale, guard, h.Daily)
		r.Get(p+"/mode", locale, guard, h.Mode)

		// Smart messages v2 (B-N9-05, D-86): the rule engine's selection for a placement and the event log
		// (impression / click / dismiss / goal; per-user rate limit, ids and the event name only).
		r.Get(p+"/feed", locale, guard, h.Feed)         // T-M10-06: every home slot in one call (Go-only, D-163)
		r.Get(p+"/selected", locale, guard, h.Selected) // T-M10-06: the feed of one slot
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
		messages.InstallModuleReads(messages.NewModuleReads(d.DB, menopause.NewService(d.DB, cat))) // T-M10-09b: meno.*/cond.* facts in the feed and the inbox jobs
		meno := menomessages.NewHandlers(d.DB, menopause.NewService(d.DB, cat), clock.Real{})
		r.Get(p+"/menopause", locale, guard, meno.Index)
	})
}

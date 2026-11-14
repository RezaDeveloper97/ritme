package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/plus"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
	"github.com/ritme/backend-go/internal/remindersms"
)

// Care reminders (docs/care-reminders/README.md), Go only — no Laravel counterpart
// (deviations.md). All auth:api, localized by Accept-Language. Static paths before {id}.
func init() {
	Register("care", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := care.NewHandlers(d.DB, clock.Real{})
		// «ثبت برای …» (B-N4-02): for_user_id on show / store / update of meds and appointments, live companion grant.
		// family.Decide behind every delegated access (T-M11-05); reads coalesced in Redis (F-16, D-203).
		h.SetDelegation(companion.NewDelegation(d.DB, clock.Real{}, d.Logger).WithReadAudit(family.NewReadAudit(d.Cache)))
		writes := writeThrottle(d) // per-user write limit (security audit M3-M7 #5)
		// for_user_id GETs: per-companion limit (B-N4-08b CMP-M2, companion_guards.go).
		delegatedReads := delegatedReadThrottle(d)

		prefs := profilestore.New(d.DB)
		h.SetPush(d.Config.Push, prefs)
		// Web Push sender for due doses (VAPID_* set; API_RUN_WORKERS / the worker role decide who runs it).
		if d.DB != nil && d.Config.Push.Enabled() {
			pusher := care.NewPusher(d.DB, prefs, d.Config.Push, d.Logger)
			ctx, cancel := context.WithCancel(context.Background())
			now := func() time.Time { return time.Now().In(civildate.Tehran) }
			auth.OnLifecycle(r, func() error { go pusher.Loop(ctx, now); return nil }, cancel)
		}

		// «یادآور پیامکی» (Ritme Plus): due doses and appointment reminders also by SMS. Off unless
		// REMINDER_SMS_PROVIDER is set; care itself never asks Plus — the audience comes from here.
		if d.DB != nil && d.Config.ReminderSMS.Provider != config.CompanionSMSNone {
			sender, err := remindersms.NewSender(d.Config, nil, d.Logger)
			if err != nil {
				panic(err)
			}
			if sender.Delivers() {
				languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
				worker := remindersms.NewWorker(d.DB, prefs, plus.NewService(d.DB, d.Config.Plus, nil, d.Logger).PlusUsers,
					sender, languages.DefaultCode, d.Config.ReminderSMS.DailyCap, d.Logger)
				ctx, cancel := context.WithCancel(context.Background())
				now := func() time.Time { return time.Now().In(civildate.Tehran) }
				auth.OnLifecycle(r, func() error { go worker.Loop(ctx, now); return nil }, cancel)
			}
		}

		r.Get("/api/v1/care/enums", locale, guard, h.Enums)
		r.Get("/api/v1/care/upcoming", locale, guard, h.Upcoming)
		r.Get("/api/v1/push/config", locale, guard, h.PushConfig)
		r.Post("/api/v1/push/subscriptions", locale, guard, writes, h.Subscribe)
		r.Delete("/api/v1/push/subscriptions", locale, guard, writes, h.Unsubscribe)
		r.Get("/api/v1/care/today", locale, guard, h.Today)
		r.Get("/api/v1/care/medications", locale, guard, h.ListMedications)
		r.Post("/api/v1/care/medications", locale, guard, writes, h.StoreMedication)
		r.Get("/api/v1/care/medications/:id", locale, guard, delegatedReads, h.ShowMedication)
		r.Put("/api/v1/care/medications/:id", locale, guard, writes, h.UpdateMedication)
		r.Delete("/api/v1/care/medications/:id", locale, guard, writes, h.DestroyMedication)
		r.Post("/api/v1/care/medications/:id/intakes", locale, guard, writes, h.TakeIntake)
		r.Delete("/api/v1/care/medications/:id/intakes", locale, guard, writes, h.UntakeIntake)
		r.Get("/api/v1/care/appointments", locale, guard, h.ListAppointments)
		r.Post("/api/v1/care/appointments", locale, guard, writes, h.StoreAppointment)
		r.Get("/api/v1/care/appointments/:id", locale, guard, delegatedReads, h.ShowAppointment)
		r.Put("/api/v1/care/appointments/:id", locale, guard, writes, h.UpdateAppointment)
		r.Delete("/api/v1/care/appointments/:id", locale, guard, writes, h.DestroyAppointment)
		r.Post("/api/v1/care/appointments/:id/cancel", locale, guard, writes, h.CancelAppointment)
		r.Patch("/api/v1/care/appointments/:id/prep/:itemId", locale, guard, writes, h.TogglePrepItem)
	})
}

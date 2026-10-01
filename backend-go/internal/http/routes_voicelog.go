package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/voicelog"
	"github.com/ritme/backend-go/resources/translations"
)

// Voice logging (B-N3-05, internal/voicelog), Go only — no Laravel route. auth:api, localized, Plus-gated
// (plus.voice_log: 402 for free users before anything is read), then per-user burst + hourly throttles.
// The AI provider comes from AI_PROVIDER (internal/ai; fake outside production, none in production).
func init() {
	Register("voicelog", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(languages)
		plusSvc := plus.NewService(d.DB, d.Config.Plus, nil, d.Logger) // entitlements + counters only
		gate := plus.NewGate(plusSvc, clock.Real{})
		client := ai.New(ai.Deps{App: d.Config.App, Config: d.Config.AI, Logger: d.Logger})
		h := voicelog.NewHandlers(voicelog.NewService(client, healthlog.NewService(d.DB)), plusSvc, gate,
			i18n.NewTranslationStore(translations.FS, d.Config.StoragePath), languages, clock.Real{})

		throttles := []any{}
		if d.Cache != nil {
			l := ratelimit.New(d.Cache, clock.Real{})
			throttles = append(throttles,
				l.NamedWith("voice-burst", voicelog.BurstMax, time.Minute, auth.ThrottleIdentity, voicelog.Throttled),
				l.NamedWith("voice-hourly", voicelog.HourlyMax, time.Hour, auth.ThrottleIdentity, voicelog.Throttled))
		}
		handlers := append([]any{guard, gate.Require(plus.VoiceLog)}, throttles...)
		r.Post("/api/v1/logs/voice", locale, append(handlers, h.Voice)...)
	})
}

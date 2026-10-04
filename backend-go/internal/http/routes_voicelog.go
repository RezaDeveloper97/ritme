package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/conditions"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/pelvic"
	pelvicstore "github.com/ritme/backend-go/internal/pelvic/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/voicelog"
	"github.com/ritme/backend-go/resources/translations"
)

// Voice logging (B-N3-05, internal/voicelog), Go only — no Laravel route. auth:api, localized, Plus-gated
// (plus.voice_log: 402 for free users before anything is read), then per-user burst + hourly throttles.
// The AI provider comes from AI_PROVIDER (internal/ai; fake outside production, none in production).
// CB-VOICE-01: the canvas items (hot flashes, pain diary, pill, bladder diary) are understood too and saved by
// POST /logs/voice/commit through each domain's own service (no AI there: auth + write throttle only).
func init() {
	Register("voicelog", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(languages)
		plusSvc := plus.NewService(d.DB, d.Config.Plus, nil, d.Logger) // entitlements + counters only
		gate := plus.NewGate(plusSvc, clock.Real{})
		// B-N6-05: usage + cost log, daily cost cap and the consent gate (ai_voice_log, only while the provider
		// is external — the voice UI has no consent sheet yet; the fake keeps everything in-process).
		platform := newAIPlatform(d, gate)
		client := platform.client
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		svc := voicelog.NewService(client, healthlog.NewService(d.DB)).WithWriters(voicelog.Writers{
			Flashes: menopause.NewService(d.DB, cat),
			Pain:    conditions.NewService(d.DB, cat),
			Pills:   contraception.NewService(d.DB),
			Bladder: pelvic.NewService(pelvicstore.New(d.DB), cat),
		})
		h := voicelog.NewHandlers(svc, plusSvc, gate,
			i18n.NewTranslationStore(translations.FS, d.Config.StoragePath), languages, clock.Real{})

		throttles := []any{}
		if d.Cache != nil {
			l := ratelimit.New(d.Cache, clock.Real{})
			throttles = append(throttles,
				l.NamedWith("voice-burst", voicelog.BurstMax, time.Minute, auth.ThrottleIdentity, voicelog.Throttled),
				l.NamedWith("voice-hourly", voicelog.HourlyMax, time.Hour, auth.ThrottleIdentity, voicelog.Throttled))
		}
		handlers := append(append([]any{guard}, platform.guard.Chain(ai.FeatureVoiceLog)...), throttles...)
		r.Post("/api/v1/logs/voice", locale, append(handlers, h.Voice)...)
		r.Post("/api/v1/logs/voice/commit", locale, guard, writeThrottle(d), h.Commit)
	})
}

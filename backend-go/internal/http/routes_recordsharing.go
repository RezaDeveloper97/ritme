package http

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/sharelinks"
	sharelinkstore "github.com/ritme/backend-go/internal/sharelinks/store"
	"github.com/ritme/backend-go/resources/translations"
)

// Doctor-code brute-force guards (CB-REC-03). A code has ~39 bits (sharelinks.CodeAlphabet^8) and at most
// sharelinks.MaxActiveSummaries live per user, so online guessing is what matters:
//
//   - ShareCodeLookupsPerIP lookups per client IP per ShareCodeLookupWindow (the framework 429), whatever the result;
//   - a global failed-lookup circuit breaker: ShareCodeFailuresPerMinute refusals within a minute, or
//     ShareCodeFailuresPerHour within an hour, from all callers together pause every code lookup for
//     ShareCodeBreakerCooldown (429 too_many_attempts with Retry-After) and log an error-level alert.
//
// Even with 10^5 live codes a guesser needs ~6.6·10^6 guesses per hit; at ≤ 300 refusals per cool-down cycle that is
// years. Trade-off (accepted): anyone can pause code lookups for 15 minutes with ~60 wrong codes; the QR link
// (256-bit token) keeps working meanwhile.
const (
	ShareCodeLookupsPerIP      = 10
	ShareCodeLookupWindow      = 10 * time.Minute
	ShareCodeFailuresPerMinute = 60
	ShareCodeFailuresPerHour   = 300
	ShareCodeBreakerCooldown   = 15 * time.Minute
)

// redisCodeBreaker is sharelinks.CodeBreaker on Redis (global keys shared by every API instance; the same counting
// script as the companion accept breaker).
type redisCodeBreaker struct {
	c                *cache.Client
	perMinute, perHr int
	cooldown         time.Duration
}

func (b redisCodeBreaker) key(k string) string { return b.c.Key("share-code-breaker:" + k) }

func (b redisCodeBreaker) Open(ctx context.Context) (bool, int, error) {
	ttl, err := b.c.Redis().TTL(ctx, b.key("open")).Result()
	if err != nil {
		return false, 0, fmt.Errorf("share code breaker: %w", err)
	}
	if ttl <= 0 {
		return false, 0, nil
	}
	return true, max(1, int(ttl/time.Second)), nil
}

func (b redisCodeBreaker) Failed(ctx context.Context) (bool, error) {
	n, err := breakerScript.Run(ctx, b.c.Redis(), []string{b.key("min"), b.key("hour"), b.key("open")},
		b.perMinute, b.perHr, int(b.cooldown/time.Second)).Int()
	if err != nil {
		return false, fmt.Errorf("share code breaker: %w", err)
	}
	return n == 1, nil
}

// recordShareFiles is the file service a summary's document links are signed with: the same vault policy as
// routes_files.go (FILE_KEY → LAB_FILE_KEY, development key only for local / testing / contract), reads and signed
// links only. Also used by routes_sharelinks.go, whose GET /shared-reports/{token} opens summaries by their QR.
func recordShareFiles(d *Deps) *files.Service {
	current, previous, dev, missing := d.Config.Files.Resolve(d.Config.App, d.Config.LabFiles)
	if dev {
		current = files.DevKey("ritme-file-dev-key")
	}
	vault, err := files.NewVault(d.Config.StoragePath, current, previous, missing)
	if err != nil {
		panic(err) // config.Load already checked the key lengths
	}
	return files.NewService(files.Options{DB: d.DB, Vault: vault, BaseURL: d.Config.App.URL})
}

// companionFamily adapts companion.Service.RecordScope to sharelinks.FamilySource.
type companionFamily struct{ svc *companion.Service }

func (f companionFamily) Family(ctx context.Context, ownerID uint64) ([]sharelinks.FamilyMember, error) {
	rows, err := f.svc.RecordScope(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]sharelinks.FamilyMember, 0, len(rows))
	for _, r := range rows {
		out = append(out, sharelinks.FamilyMember{ID: r.CompanionID, Type: string(r.Type), Status: string(r.Status),
			Name: r.Name, Meds: string(r.Meds), Appointments: string(r.Appointments)})
	}
	return out, nil
}

// Record sharing «اشتراک‌گذاری پرونده» (canvas-build CB-REC-03, deviations.md D-71), Go only, on top of bloom's share
// links (routes_sharelinks.go, D-65): the 24h doctor summary with a short code + QR (Go-only route group
// /health-record/share-codes, not Plus-gated), the sharing overview (doctor codes, companion scope = meds &
// appointments only, never documents, latest access log), the access log of every link, and the public code lookup
// POST /shared-reports/code (IP-throttled, breaker-guarded; logs collapse /api/v1/shared-reports/*). The QR's token
// URL is served by bloom's GET /shared-reports/{token}, which now writes the access log too. Codes, tokens and
// report contents are never logged.
func init() {
	Register("recordsharing", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		labsSvc := labs.NewService(labs.Options{ // the summary's lab rows (reads only)
			DB: d.DB, Catalog: catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger), Logger: d.Logger,
		})
		reports := healthrecord.NewService(d.DB, labsSvc).
			WithBundles(i18n.NewTranslationStore(translations.FS, d.Config.StoragePath))

		fileSvc := recordShareFiles(d)

		pepperMissing := d.Config.Sharing.PepperMissing(d.Config.App)
		if pepperMissing {
			d.Logger.Warn("SHARE_CODE_PEPPER is not set in production: 24h doctor codes are disabled (503)")
		}
		svc := sharelinks.NewService(sharelinkstore.New(d.DB), reports, d.Logger).
			WithCodes([]byte(d.Config.Sharing.CodePepper), pepperMissing).
			WithDocuments(sharelinks.NewRecordDocuments(d.DB), fileSvc)
		family := companionFamily{svc: companion.NewService(d.DB, clock.Real{}, companion.Options{
			CodePepper: []byte(d.Config.Companion.CodePepper), // unused by RecordScope (reads only)
		})}

		var breaker sharelinks.CodeBreaker
		lookups := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			breaker = redisCodeBreaker{c: d.Cache, perMinute: ShareCodeFailuresPerMinute,
				perHr: ShareCodeFailuresPerHour, cooldown: ShareCodeBreakerCooldown}
			lookups = ratelimit.New(d.Cache, clock.Real{}).Named("share-code", ShareCodeLookupsPerIP, ShareCodeLookupWindow, nil)
		}
		onTrip := func(ctx context.Context) {
			// The alert: a burst of wrong doctor codes across callers (blind guessing). No code, no user data.
			d.Logger.ErrorContext(ctx, "share code circuit breaker tripped: code lookups paused for everyone")
		}
		h := sharelinks.NewSharingHandlers(svc, clock.Real{}, family, d.Config.Sharing.WebURL, breaker, onTrip)
		writes := writeThrottle(d)

		const p = "/api/v1/health-record"
		r.Get(p+"/sharing", locale, guard, h.Overview)
		r.Get(p+"/share-access", locale, guard, h.Access)
		r.Post(p+"/share-codes", locale, guard, writes, h.StoreCode)
		r.Delete(p+"/share-codes/:id", locale, guard, writes, h.DestroyCode)
		r.Post("/api/v1/shared-reports/code", lookups, locale, h.OpenCode)
	})
}

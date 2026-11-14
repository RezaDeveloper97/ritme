package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/children"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/companion/shared"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/family/invites"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/sms"
	"github.com/ritme/backend-go/internal/teen"
)

// Companion «همدم» throttles (bloom B-N4-02). A code has 32^6 ≈ 1.07e9 values and lives 24 h, so with these caps a
// guesser needs tens of thousands of accounts × IPs per hit; a bound invite also locks after 5 wrong-account tries.
const (
	// CompanionAcceptPerUser is how many accept attempts one account may send per CompanionAcceptWindow.
	CompanionAcceptPerUser = 10
	// CompanionAcceptPerIP is how many accept attempts one client IP may send per CompanionAcceptWindow.
	CompanionAcceptPerIP = 30
	// CompanionAcceptWindow is the accept throttles' window.
	CompanionAcceptWindow = time.Hour
	// CompanionInvitesPerUser caps invite creation + renewal (each may send an SMS) per CompanionAcceptWindow.
	CompanionInvitesPerUser = 20
)

// Companion & family (bloom B-N4-02, D-48), Go only. auth:api, localized by Accept-Language. Owner side: list, invite
// (SMS adapter or shared code), show, renew, grants, shared children, revoke, audit trail. Companion side: accept by
// code (throttled per user and per IP, one generic refusal), links, leave, and the access-filtered, audited section
// reads. «ثبت برای …» writes live on the care routes (for_user_id, routes_care.go).
func init() {
	Register("companion", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		sender, err := sms.New(d.Config, nil, d.Logger)
		if err != nil {
			panic(err) // config.Load already refused unknown providers and fake in production
		}
		// FAM-L1 / CMP-L3 (T-M11-08): outside local / testing / contract the invite pepper is mandatory — the server
		// refuses to start without it; the routes also answer 503 as a second line.
		if err := d.Config.Companion.CheckPepper(d.Config.App); err != nil {
			auth.FailStartup(r, d.Logger, err)
		}
		disabled := d.Config.Companion.PepperMissing(d.Config.App)
		teenPolicy := teen.NewPolicy(d.DB)
		plusGate := plus.NewGate(plus.NewService(d.DB, d.Config.Plus, nil, d.Logger), clock.Real{}) // entitlements only
		h := companion.NewHandlers(d.DB, clock.Real{}, companion.HandlerOptions{
			Service: companion.Options{CodePepper: []byte(d.Config.Companion.CodePepper),
				ReadAudit:         family.NewReadAudit(d.Cache), // F-16, D-203: Redis read-audit coalescing
				MinAcceptDuration: invites.DefaultMinDuration,   // CMP-L2: constant-time accept (D-196 invite service)
				Mismatch:          inviteMismatchGate(d)},       // CMP-L8: one mismatch per account
			SMS:      sender,
			Reader:   shared.NewReader(d.DB).WithEngine(engineStates(r, d)),
			Children: children.NewService(d.DB, nil), // spouse shared children: owner's own children only (B-N5-02)
			Disabled: disabled,
			Logger:   d.Logger,
			SendGate: companionSendGate(d),
			// «ریتمی همراه» is a Ritme Plus feature: a partner / spouse invite needs plus.companion. A teen's parent
			// link stays free, and a teen-mode account is never shown a paywall (CB-TEEN-01: it cannot buy Plus) —
			// its partner invite goes on to the teen_parent_only refusal.
			InviteGate: func(c fiber.Ctx, t companion.Type) error {
				if t == companion.TypeParent {
					return nil
				}
				if userID, ok := auth.CurrentUserID(c); ok {
					commercial, err := teenPolicy.AllowsCommercial(c.Context(), userID)
					if err != nil {
						return err
					}
					if !commercial {
						return nil
					}
				}
				return plusGate.Check(c, plus.Companion)
			},
			Breaker: companionAcceptBreaker(d), // CMP-M3 (companion_guards.go)
		})
		acceptUser := companionThrottle(d, "companion-accept", CompanionAcceptPerUser, auth.ThrottleIdentity)
		acceptIP := companionThrottle(d, "companion-accept-ip", CompanionAcceptPerIP, nil)
		invites := companionThrottle(d, "companion-invite", CompanionInvitesPerUser, auth.ThrottleIdentity)

		c := "/api/v1/companions"
		r.Get(c, locale, guard, h.Index)
		r.Post(c, locale, guard, writes, invites, h.Store)
		r.Get(c+"/audit", locale, guard, h.Audit)
		r.Post(c+"/accept", locale, guard, acceptIP, acceptUser, h.Accept)
		r.Get(c+"/links", locale, guard, h.Links)
		r.Delete(c+"/links/:id", locale, guard, writes, h.Leave)
		r.Get(c+"/links/:id/sections/:section", locale, guard, companionReadThrottle(d), h.Section)
		r.Get(c+"/:id", locale, guard, h.Show)
		r.Delete(c+"/:id", locale, guard, writes, h.Destroy)
		r.Post(c+"/:id/renew", locale, guard, writes, invites, h.Renew)
		r.Put(c+"/:id/grants", locale, guard, writes, h.UpdateGrants)
		r.Put(c+"/:id/children", locale, guard, writes, h.UpdateChildren)
		r.Put(c+"/:id/preferences", locale, guard, writes, h.UpdatePreferences) // «ریتمی همراه» v2
	})
}

// companionThrottle is a named per-user (identity) or per-IP (nil identity) limit with the localized companion 429.
// Without Redis (unit tests) it lets everything through.
func companionThrottle(d *Deps, name string, maxAttempts int, identity ratelimit.Identity) fiber.Handler {
	if d.Cache == nil {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	return ratelimit.New(d.Cache, clock.Real{}).NamedWith(name, maxAttempts, CompanionAcceptWindow, identity,
		func(c fiber.Ctx, r ratelimit.Rejection) error {
			return companion.TooManyAttempts(i18n.Locale(c), r.RetryAfter, r.Headers())
		})
}

// redisSendGate adapts the Redis fixed-window limiter to companion.SendGate (request clock, Go key prefix).
type redisSendGate struct{ l *ratelimit.Limiter }

func (g redisSendGate) Allow(ctx context.Context, key string, maxHits int, window time.Duration) (bool, error) {
	res, err := g.l.Attempt(ctx, key, maxHits, window, clock.FromContext(ctx, clock.Real{}).Now())
	return res.Allowed, err
}

// companionSendGate is the invite-SMS gate; without Redis (unit tests) there is none.
func companionSendGate(d *Deps) companion.SendGate {
	if d.Cache == nil {
		return nil
	}
	return redisSendGate{l: ratelimit.New(d.Cache, clock.Real{})}
}

// inviteMismatchGate counts a phone-bound code's wrong-account tries once per account (CMP-L8); without Redis (unit
// tests) every try counts.
func inviteMismatchGate(d *Deps) invites.Gate {
	if d.Cache == nil {
		return nil
	}
	return redisSendGate{l: ratelimit.New(d.Cache, clock.Real{})}
}

// companionInviteSender is the /companions invite SMS (adapter, CB-PRIV-01 wording, gates) for POST /family/members.
func companionInviteSender(d *Deps) invites.Sender {
	sender, err := sms.New(d.Config, nil, d.Logger)
	if err != nil {
		panic(err) // config.Load already refused unknown providers and fake in production
	}
	h := companion.NewHandlers(d.DB, clock.Real{}, companion.HandlerOptions{SMS: sender, Logger: d.Logger,
		SendGate: companionSendGate(d)})
	return h.SendInviteCode
}

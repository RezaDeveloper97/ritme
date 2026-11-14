package http

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/family/invites"
	"github.com/ritme/backend-go/internal/family/projections"
	"github.com/ritme/backend-go/internal/family/sharerequests"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Family v6 (engine v6 M11, T-M11-05 / T-M11-06 / T-M11-08; D-193, D-194, D-196, D-204), Go only: GET
// /api/v1/family/me lists the caller's links as owner (grants with their state) and as member (effective domains
// only); GET /api/v1/family/owners/{ownerId}/domains/{domain} is the member projection of one domain (allow-list DTOs,
// internal/family/projections). Every member read and write of the family feature is decided by family.Decide;
// /companions/* stays the compatibility layer (routes_companion.go). Every /family/* answer is private, no-store.
//
// Invites v6 (T-M11-08, internal/family/invites): POST /family/members (owner: role, template, edits, expiry, urgent
// switch, mutual request, phone or code, consent version), POST /family/members/{linkId}/confirm | cancel, and the
// member side POST /family/invites/preview | accept | decline and GET /family/invites/pending. Preview, accept and
// decline share the /companions/accept per-user and per-IP throttles (same limiter names, one budget) and its global
// failed-code breaker, and answer every refusal with the same generic 422. Share requests
// (internal/family/sharerequests): POST /family/share-requests, GET ?box=in|out, POST /{id}/approve | decline.
// The daily family.invites_sweep job expires stale invites and time-boxed links (docs/engine-v6/jobs.md).
func init() {
	Register("family", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		langs := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(langs)
		h := family.NewHandlers(d.DB, clock.Real{})
		p := projections.NewHandlers(d.DB, projections.New(d.DB).WithEngine(engineStates(r, d)),
			projections.NewAuditor(d.DB, family.NewReadAudit(d.Cache)), clock.Real{})
		r.Get("/api/v1/family/me", projections.NoStore, locale, guard, h.Me)
		r.Get("/api/v1/family/owners/:ownerId/domains/:domain", projections.NoStore, locale, guard, companionReadThrottle(d), p.Domain)

		invOpt := invites.Options{Pepper: []byte(d.Config.Companion.CodePepper), MinDuration: invites.DefaultMinDuration,
			Mismatch: inviteMismatchGate(d)}
		inv := invites.NewHandlers(d.DB, clock.Real{}, invites.HandlerOptions{
			Service:  invOpt,
			Disabled: d.Config.Companion.PepperMissing(d.Config.App),
			Breaker:  companionAcceptBreaker(d), // the same global breaker as POST /companions/accept (CMP-M3)
			Send:     companionInviteSender(d),
			Logger:   d.Logger,
		})
		// The same named limiters as /companions/accept: preview, accept and decline draw on one budget (FAM-L3).
		codeUser := companionThrottle(d, "companion-accept", CompanionAcceptPerUser, auth.ThrottleIdentity)
		codeIP := companionThrottle(d, "companion-accept-ip", CompanionAcceptPerIP, nil)
		creates := companionThrottle(d, "companion-invite", CompanionInvitesPerUser, auth.ThrottleIdentity)
		writes := writeThrottle(d)
		sweep := invites.NewService(d.DB, clock.Real{}, invOpt)
		mustJobs(sweep.RegisterJobs(auth.Jobs(r, d.Cache, d.Logger), tehranNow,
			func(ctx context.Context) []string { return langs.All(ctx).Codes() }))

		f := "/api/v1/family"
		r.Post(f+"/members", locale, guard, writes, creates, inv.Store)
		r.Post(f+"/members/:linkId/confirm", locale, guard, writes, inv.Confirm)
		r.Post(f+"/members/:linkId/cancel", locale, guard, writes, inv.Cancel)
		r.Get(f+"/invites/pending", locale, guard, inv.Pending)
		r.Post(f+"/invites/preview", locale, guard, codeIP, codeUser, inv.Preview)
		r.Post(f+"/invites/accept", locale, guard, codeIP, codeUser, inv.Accept)
		r.Post(f+"/invites/decline", locale, guard, codeIP, codeUser, inv.Decline)

		sr := sharerequests.NewHandlers(d.DB, clock.Real{})
		r.Get(f+"/share-requests", locale, guard, sr.Index)
		r.Post(f+"/share-requests", locale, guard, writes, sr.Store)
		r.Post(f+"/share-requests/:id/approve", locale, guard, writes, sr.Approve)
		r.Post(f+"/share-requests/:id/decline", locale, guard, writes, sr.Decline)
	})
}

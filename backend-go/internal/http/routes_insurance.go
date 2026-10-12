package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/insurance"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// insuranceService builds the service shared by the owner routes and the admin catalog.
func insuranceService(d *Deps) *insurance.Service {
	box, err := loss.NewNoteBox(d.Config.PrivateNotes.Key, d.Config.PrivateNotes.PreviousKeys, d.Config.PrivateNotes.Missing(d.Config.App))
	if err != nil {
		panic(err) // config.Load already checked the key length
	}
	return insurance.NewService(d.DB, box)
}

// Insurance self-tracking (canvas CB-INS-01, internal/insurance, deviations.md D-INS-01), Go only, auth:api,
// localized, writes per-user throttled: her policy, coverage usage, claims with the timeline she maintains and record
// document links, health questionnaire drafts prefilled from her record. Nothing is sent to an insurer. Every row is
// owner-only (another user's id answers 404). The «how it works» page stays bloom B-N10-06 (/info-sections).
func init() {
	Register("insurance", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		h := insurance.NewHandlers(insuranceService(d), clock.Real{})

		const p = "/api/v1/insurance"
		r.Get(p+"/insurers", locale, guard, h.Insurers)
		r.Get(p+"/centres", locale, guard, h.Centres)
		r.Get(p+"/policy", locale, guard, h.Policy)
		r.Put(p+"/policy", locale, guard, writes, h.SavePolicy)
		r.Delete(p+"/policy", locale, guard, writes, h.DeletePolicy)
		// Policy history (user decision 2026-10-12, 00093): /policy above stays the current policy (the home).
		r.Get(p+"/policies", locale, guard, h.Policies)
		r.Post(p+"/policies", locale, guard, writes, h.StorePolicy)
		r.Get(p+"/policies/:id", locale, guard, h.ShowPolicy)
		r.Put(p+"/policies/:id", locale, guard, writes, h.UpdatePolicy)
		r.Delete(p+"/policies/:id", locale, guard, writes, h.DestroyPolicy)
		r.Post(p+"/policies/:id/renew", locale, guard, writes, h.RenewPolicy)
		r.Get(p+"/claims", locale, guard, h.Claims)
		r.Post(p+"/claims", locale, guard, writes, h.StoreClaim)
		r.Get(p+"/claims/:id", locale, guard, h.ShowClaim)
		r.Put(p+"/claims/:id", locale, guard, writes, h.UpdateClaim)
		r.Delete(p+"/claims/:id", locale, guard, writes, h.DestroyClaim)
		r.Post(p+"/claims/:id/events", locale, guard, writes, h.StoreEvent)
		r.Delete(p+"/claims/:id/events/:eventId", locale, guard, writes, h.DestroyEvent)
		r.Post(p+"/claims/:id/documents", locale, guard, writes, h.AttachDocument)
		r.Delete(p+"/claims/:id/documents/:documentId", locale, guard, writes, h.DetachDocument)
		r.Get(p+"/questionnaires", locale, guard, h.Questionnaires)
		r.Post(p+"/questionnaires", locale, guard, writes, h.StoreQuestionnaire)
		r.Get(p+"/questionnaires/:id", locale, guard, h.ShowQuestionnaire)
		r.Put(p+"/questionnaires/:id", locale, guard, writes, h.UpdateQuestionnaire)
		r.Delete(p+"/questionnaires/:id", locale, guard, writes, h.DestroyQuestionnaire)
		r.Post(p+"/questionnaires/:id/confirm", locale, guard, writes, h.ConfirmQuestionnaire)
		r.Post(p+"/questionnaires/:id/status", locale, guard, writes, h.AdvanceQuestionnaire)
	})
	Register("admin_insurance", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		insurance.NewAdmin(insuranceService(d), d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}

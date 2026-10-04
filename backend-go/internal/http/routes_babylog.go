package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/babylog"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/children"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Baby logs (bloom B-N5-03, D-60), Go only: feeding / sleep sessions and diapers of a child, with day summaries.
// All auth:api, localized by Accept-Language, writes per-user throttled. The child is resolved by children.Service:
// the owner reads and writes, the owner's spouse through a shared family reads (audited) and gets 403 on writes;
// anything else is a uniform 404. Nothing child-related is logged.
func init() {
	Register("babylog", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		h := babylog.NewHandlers(babylog.NewService(d.DB, children.NewService(d.DB, cat)), clock.Real{})
		writes := writeThrottle(d)

		p := "/api/v1/children/:id"
		r.Get(p+"/feeds", locale, guard, h.Feeds)
		r.Post(p+"/feeds", locale, guard, writes, h.StoreFeed)
		r.Post(p+"/feeds/start", locale, guard, writes, h.StartFeed)
		r.Put(p+"/feeds/:fid", locale, guard, writes, h.UpdateFeed)
		r.Delete(p+"/feeds/:fid", locale, guard, writes, h.DestroyFeed)
		r.Post(p+"/feeds/:fid/side", locale, guard, writes, h.FeedSide)
		r.Post(p+"/feeds/:fid/stop", locale, guard, writes, h.StopFeed)

		r.Get(p+"/sleeps", locale, guard, h.Sleeps)
		r.Post(p+"/sleeps", locale, guard, writes, h.StoreSleep)
		r.Post(p+"/sleeps/start", locale, guard, writes, h.StartSleep)
		r.Put(p+"/sleeps/:sid", locale, guard, writes, h.UpdateSleep)
		r.Delete(p+"/sleeps/:sid", locale, guard, writes, h.DestroySleep)
		r.Post(p+"/sleeps/:sid/stop", locale, guard, writes, h.StopSleep)

		r.Get(p+"/diapers", locale, guard, h.Diapers)
		r.Post(p+"/diapers", locale, guard, writes, h.StoreDiaper)
		r.Put(p+"/diapers/:did", locale, guard, writes, h.UpdateDiaper)
		r.Delete(p+"/diapers/:did", locale, guard, writes, h.DestroyDiaper)

		r.Get(p+"/baby-logs", locale, guard, h.Summary)
	})
}

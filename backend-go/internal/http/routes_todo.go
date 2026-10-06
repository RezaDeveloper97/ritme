package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/todo"
)

// To-do list «کارهای من» (bloom B-N6-08, D-66), Go only: tasks with categories, shopping-list items and the cycle
// suggestion. All auth:api, localized by Accept-Language, writes per-user throttled; every row is scoped by the user
// (a foreign id is a uniform 404). Titles are never logged.
func init() {
	Register("todo", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := todo.NewHandlers(todo.NewService(d.DB), clock.Real{})
		writes := writeThrottle(d)

		p := "/api/v1/todo"
		r.Get(p, locale, guard, h.Board)
		r.Post(p+"/tasks", locale, guard, writes, h.Store)
		r.Get(p+"/tasks/:id", locale, guard, h.Show)
		r.Put(p+"/tasks/:id", locale, guard, writes, h.Update)
		r.Delete(p+"/tasks/:id", locale, guard, writes, h.Destroy)
		r.Post(p+"/tasks/:id/items", locale, guard, writes, h.StoreItem)
		r.Put(p+"/tasks/:id/items/:item", locale, guard, writes, h.UpdateItem)
		r.Delete(p+"/tasks/:id/items/:item", locale, guard, writes, h.DestroyItem)
		r.Post(p+"/suggestions/:key/accept", locale, guard, writes, h.AcceptSuggestion)
		r.Post(p+"/suggestions/:key/dismiss", locale, guard, writes, h.DismissSuggestion)
	})
}

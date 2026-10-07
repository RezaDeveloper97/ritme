package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	adminlearning "github.com/ritme/backend-go/internal/admin/learning"
)

// Admin moderation of courses «مدرسین و دوره‌ها» (bloom B-N8-08) under /api/admin/v1/learning/* —
// docs/go-migration/admin-api.md §20: instructors (approve / revoke: super only), courses with usage, the content
// review queue (approve / flag / unpublish) and usage stats. Every action writes a learning_moderation_log row.
func init() {
	Register("admin_learning", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		adminlearning.NewHandlers(d.DB, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}

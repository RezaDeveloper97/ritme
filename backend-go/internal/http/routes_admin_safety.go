package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/safetyreview"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/safety"
)

// Safety & clinical content dashboard and review queue (bloom B-N9-10, D-90) under /api/admin/v1/safety —
// docs/go-migration/admin-api.md «Safety & clinical content». Also runs the urgent-path scenarios at start-up and
// nightly (internal/safety; the CI twin is .github/workflows/safety-scenarios-nightly.yml).
func init() {
	Register("admin_safety", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		safetyreview.New(d.DB, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
		ctx, cancel := context.WithCancel(context.Background())
		now := func() time.Time { return time.Now().In(civildate.Tehran) }
		auth.OnLifecycle(r, func() error { go safety.Loop(ctx, d.Logger, now); return nil }, cancel)
	})
}

package http

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
)

// Every admin route maps to a capability of the B-N9-02 matrix; only the super-only modules (admins, roles,
// languages) fall through to "?" (super / editor only), so a new module cannot open up to every role by accident.
func TestAdminRoutes_EveryRouteHasACapability(t *testing.T) {
	keys, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	db, err := sql.Open("mysql", "inventory:inventory@tcp(127.0.0.1:1)/inventory")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	rc := cache.New(config.Redis{Host: "127.0.0.1", Port: 1, Prefix: "ritme-go-test:"})
	t.Cleanup(func() { _ = rc.Redis().Close() })
	app := fiber.New()
	Mount(app, &Deps{
		Config: &config.Config{
			App: config.App{Env: "testing", URL: "http://localhost"}, SMS: config.SMS{Provider: "log"},
			Passport: config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30}, StoragePath: keys,
		},
		DB: db, Cache: rc, Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})
	superOnly := []string{"/admins", "/roles", "/languages", "/translations"}
	n := 0
	for _, r := range app.GetRoutes(true) {
		if !strings.HasPrefix(r.Path, httpadmin.Prefix+"/") || r.Method == fiber.MethodOptions || r.Method == fiber.MethodHead {
			continue
		}
		n++
		if httpadmin.RouteCapability(r.Path) != "?" {
			continue
		}
		rest := strings.TrimPrefix(r.Path, httpadmin.Prefix)
		ok := false
		for _, p := range superOnly {
			ok = ok || rest == p || strings.HasPrefix(rest, p+"/")
		}
		assert.True(t, ok, "admin route %s %s has no capability in httpadmin.capabilityOf", r.Method, r.Path)
	}
	assert.Greater(t, n, 100)
}

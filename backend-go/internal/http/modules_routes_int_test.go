package http_test

// T-M12-18 (T-10): every module kill-switch closes every route of its group. The route→module map is read from the
// router itself (Fiber's route table of the fully mounted app, owner resolved by the production gate modules.ModuleOf),
// never from a list written here, so a route added under a module's prefix is covered without touching this file.
// A route that LOOKS like a module's (its path segment names the module) but is not gated fails the test, and so does
// a gated prefix with no route behind it.

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/http/testenv"
	"github.com/ritme/backend-go/internal/modules"
)

// uiOnlyModules have no API routes by design (modules.go: «UI-only»); a module outside this set with no gated route
// is a bug.
var uiOnlyModules = map[string]bool{
	modules.MyCourses: true, modules.Devices: true, modules.Reminders: true, modules.HomeArticles: true,
	// «من» rows (user request 2026-11-25): their APIs serve other screens too, so the switch only hides the UI.
	modules.Services: true, modules.Vitals: true, modules.Backup: true, modules.Support: true,
}

// notModuleRoutes are first path segments that merely contain a module word but belong to no switchable module
// (an explicit, reviewed exemption; empty today).
var notModuleRoutes = map[string]string{}

type apiRoute struct{ method, path string }

// mountedRoutes is every /api/v1 route of the booted app, once per method+path (HEAD and middleware entries dropped).
func mountedRoutes(e *testenv.Env) []apiRoute {
	seen := map[apiRoute]bool{}
	var out []apiRoute
	for _, r := range e.App.GetRoutes(true) {
		if !strings.HasPrefix(r.Path, "/api/v1") || r.Method == fiber.MethodHead || r.Method == fiber.MethodOptions || r.Method == "USE" {
			continue
		}
		k := apiRoute{r.Method, r.Path}
		if !seen[k] && !strings.ContainsAny(r.Path, "*+") {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path+out[i].method < out[j].path+out[j].method })
	return out
}

// concrete turns :param segments into a value.
func concrete(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "1"
		}
	}
	return strings.Join(parts, "/")
}

func norm(s string) string { return strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(s)) }

// namesModule reports the module a path's own words point at (segment after /api/v1, and after /analysis).
func namesModule(path string) string {
	segs := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	cands := segs[:1]
	if segs[0] == "analysis" && len(segs) > 1 {
		cands = append(cands, segs[1])
	}
	for _, c := range cands {
		n := norm(c)
		for _, k := range modules.Keys {
			kn := norm(k)
			if uiOnlyModules[k] || kn == "record" || kn == "learning" {
				continue
			}
			if strings.Contains(n, kn) {
				return k
			}
		}
	}
	return ""
}

func setModule(t *testing.T, e *testenv.Env, key string, on bool) {
	t.Helper()
	require.NoError(t, modules.NewService(e.DB, e.Cache, nil).Set(context.Background(), map[string]bool{key: on}, time.Now()))
}

func TestModules_EveryRouteOfEveryModuleHonoursItsSwitch(t *testing.T) {
	e := testenv.New(t)
	routes := mountedRoutes(e)
	require.Greater(t, len(routes), 300, "the route table looks empty")

	byModule := map[string][]apiRoute{}
	prefixSeen := map[string]bool{}
	for _, r := range routes {
		m := modules.ModuleOf(concrete(r.path))
		if m == "" {
			// A route whose own path names a module must be gated (unless it is under the always-open allowlist).
			open := false
			for _, a := range modules.AlwaysOpen() {
				open = open || strings.HasPrefix(r.path, a)
			}
			if want := namesModule(r.path); want != "" && !open {
				if _, exempt := notModuleRoutes[strings.Split(strings.TrimPrefix(r.path, "/api/v1/"), "/")[0]]; !exempt {
					assert.Failf(t, "ungated module route", "%s %s looks like module %q but no gate prefix covers it (internal/modules routePrefixes)", r.method, r.path, want)
				}
			}
			continue
		}
		byModule[m] = append(byModule[m], r)
		for p := range modules.Prefixes() {
			if r.path == p || strings.HasPrefix(r.path, p+"/") {
				prefixSeen[p] = true
			}
		}
	}
	for p, m := range modules.Prefixes() {
		assert.True(t, prefixSeen[p], "gate prefix %s (%s) has no route behind it", p, m)
	}
	for _, k := range modules.Keys {
		if uiOnlyModules[k] {
			assert.Empty(t, byModule[k], "%s is declared UI-only but owns routes", k)
		} else {
			assert.NotEmpty(t, byModule[k], "module %s has no gated route", k)
		}
	}

	total := 0
	for _, k := range modules.Keys {
		total += len(byModule[k])
	}
	t.Logf("routes: %d under /api/v1, %d gated across %d modules (%d UI-only)", len(routes), total, len(byModule), len(uiOnlyModules))

	for _, k := range modules.Keys {
		if len(byModule[k]) == 0 {
			continue
		}
		t.Run(k, func(t *testing.T) {
			setModule(t, e, k, false)
			for _, r := range byModule[k] {
				rep := e.Do(r.method, concrete(r.path), nil, "")
				assert.Equal(t, fiber.StatusForbidden, rep.Status, "%s %s switched off: %s", r.method, r.path, rep.Raw)
				assert.Equal(t, modules.CodeDisabled, rep.Body["error_code"], "%s %s: %s", r.method, r.path, rep.Raw)
				assert.Equal(t, k, rep.Body["module"], "%s %s", r.method, r.path)
			}
			// Boundary: switching back on reopens every route (never module_disabled again).
			setModule(t, e, k, true)
			for _, r := range byModule[k] {
				rep := e.Do(r.method, concrete(r.path), nil, "")
				assert.NotEqual(t, modules.CodeDisabled, rep.Body["error_code"], "%s %s switched on: %s", r.method, r.path, rep.Raw)
			}
		})
	}
}

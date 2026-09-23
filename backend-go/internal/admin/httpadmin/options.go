// Package httpadmin is the shared HTTP layer of the admin API (/api/admin/v1, see
// docs/go-migration/admin-api.md): host guard, CORS for the admin-web origin, Redis-backed
// admin sessions with HttpOnly cookies, CSRF (X-CSRF-Token), RequireAdmin / RequireSuper,
// the admin JSON envelope, pagination and audit logging.
//
// Every admin routes file builds one Kit and attaches its chains per route, so the
// protection never depends on the order in which domain files are mounted:
//
//	kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger) // in a routes_admin_*.go registrar
//	httpadmin.Handle(r, fiber.MethodGet, httpadmin.Prefix+"/articles", kit.Admin(h.List))    // any active admin
//	httpadmin.Handle(r, fiber.MethodPost, httpadmin.Prefix+"/languages", kit.Super(h.Store)) // super admins only
//
// Never register an admin endpoint as r.Get(path, handler, middleware...): Fiber runs
// the handlers in argument order, so the endpoint would run before the guards.
package httpadmin

import (
	"os"
	"strings"
)

// Prefix is the base path of the admin API.
const Prefix = "/api/admin/v1"

// Options configure the admin HTTP layer. They come from the environment
// (OptionsFromEnv) because the admin API is the only consumer.
type Options struct {
	// Hosts are the host names the admin API answers on (ADMIN_HOSTS, comma separated,
	// e.g. "adpanell.ritme.app"). Any other Host gets a 404, like the /admin split in
	// nginx today. Empty = every host when APP_ENV is local/testing; in any other
	// environment the admin API is disabled (fail closed) until the variable is set.
	Hosts []string
	// Origins are the browser origins allowed to call the API cross-origin with
	// credentials (ADMIN_WEB_ORIGINS, e.g. "http://localhost:3001"). Same-origin
	// requests (admin-web served on the admin host) need no entry.
	Origins []string
	// CookieSecure marks the session cookies Secure and gives them the __Host- prefix
	// (ADMIN_COOKIE_SECURE, default true; set false only for plain-http local dev).
	CookieSecure bool
	// Dev is APP_ENV ∈ {local, testing}: an empty Hosts list then means "any host".
	Dev bool
}

// OptionsFromEnv reads ADMIN_HOSTS, ADMIN_WEB_ORIGINS and ADMIN_COOKIE_SECURE; appEnv is
// APP_ENV (config.App.Env).
func OptionsFromEnv(appEnv string) Options {
	return OptionsFrom(os.LookupEnv, appEnv)
}

// OptionsFrom is OptionsFromEnv over a lookup function (tests).
func OptionsFrom(lookup func(string) (string, bool), appEnv string) Options {
	get := func(k string) string { v, _ := lookup(k); return strings.TrimSpace(v) }
	secure := true
	switch strings.ToLower(get("ADMIN_COOKIE_SECURE")) {
	case "false", "0", "no", "off", "(false)":
		secure = false
	}
	return Options{
		Hosts:        splitList(get("ADMIN_HOSTS"), true),
		Origins:      splitList(get("ADMIN_WEB_ORIGINS"), false),
		CookieSecure: secure,
		Dev:          appEnv == "local" || appEnv == "testing",
	}
}

func splitList(s string, lower bool) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimRight(strings.TrimSpace(p), "/")
		if lower {
			p = strings.ToLower(p)
		}
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

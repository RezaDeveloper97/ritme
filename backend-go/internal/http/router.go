// Package http is the route registry of the Go API.
//
// Every domain owns exactly one file, routes_<domain>.go, that registers itself from
// init():
//
//	func init() {
//		Register("content", func(r fiber.Router, d *Deps) {
//			v1 := r.Group("/api/v1")
//			v1.Get("/languages", ...)          // static routes first,
//			v1.Get("/languages/:code/messages", ...) // then param routes
//		})
//	}
//
// No shared file lists the domains, so parallel tasks never edit the same file.
// Domains are mounted in name order (deterministic); keep static-before-param order
// inside your own file. Fiber answers HEAD for every GET route automatically.
package http

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
)

// Deps are the shared services handed to every domain registrar.
type Deps struct {
	Config *config.Config
	DB     *sql.DB
	Cache  *cache.Client
	Logger *slog.Logger
}

// Registrar mounts one domain's routes.
type Registrar func(r fiber.Router, d *Deps)

// Registry holds named registrars.
type Registry struct {
	mu      sync.Mutex
	domains map[string]Registrar
}

// NewRegistry returns an empty registry (tests); production code uses the default one.
func NewRegistry() *Registry { return &Registry{domains: map[string]Registrar{}} }

// Register adds a domain. It panics on an empty name, a nil registrar or a duplicate
// name, so a copy-pasted routes file fails at start-up instead of silently shadowing.
func (reg *Registry) Register(domain string, fn Registrar) {
	if domain == "" || fn == nil {
		panic("http: Register needs a domain name and a registrar")
	}
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if _, dup := reg.domains[domain]; dup {
		panic(fmt.Sprintf("http: domain %q registered twice", domain))
	}
	reg.domains[domain] = fn
}

// Domains returns the registered domain names in mount order.
func (reg *Registry) Domains() []string {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	names := make([]string, 0, len(reg.domains))
	for name := range reg.domains {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Mount registers every domain's routes on r in name order.
func (reg *Registry) Mount(r fiber.Router, d *Deps) {
	for _, name := range reg.Domains() {
		reg.mu.Lock()
		fn := reg.domains[name]
		reg.mu.Unlock()
		fn(r, d)
	}
}

var defaultRegistry = NewRegistry()

// Default returns the registry that routes_<domain>.go files fill from init().
func Default() *Registry { return defaultRegistry }

// Register adds a domain to the default registry. Call it from init() in routes_<domain>.go.
func Register(domain string, fn Registrar) { defaultRegistry.Register(domain, fn) }

// Domains lists the domains in the default registry.
func Domains() []string { return defaultRegistry.Domains() }

// Mount mounts the default registry.
func Mount(r fiber.Router, d *Deps) { defaultRegistry.Mount(r, d) }

// Package modules holds the admin kill-switches for whole app modules (table app_modules, migration 00094). A
// disabled module disappears from onboarding and from the app (the frontend reads GET /api/v1/modules) and the API
// answers its routes with 403 module_disabled (Gate). A missing row means enabled, and any read error fails open, so
// a broken table can never switch a live feature off.
package modules

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/modules/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Module keys.
const (
	Companion     = "companion"     // همدم: the male companion account and the female side's companions
	TTC           = "ttc"           // اقدام به بارداری (fertility tiles / log / BBT / insights)
	IVF           = "ivf"           // درمان ناباروری (IVF / IUI)
	Menopause     = "menopause"     // یائسگی
	Teen          = "teen"          // نوجوان
	Postpartum    = "postpartum"    // پس از زایمان
	Children      = "children"      // کودک (children + baby log)
	Contraception = "contraception" // روش پیشگیری
	Programs      = "programs"      // برنامه‌های مراقبتی (pain / PMDD / bleeding / pelvic floor)
	MotherChild   = "mother_child"  // برای مادر و کودک (city services: classes, pools, play houses)
	Learning      = "learning"      // آموزش و دوره‌ها (the whole /learn module, «دوره‌های من» included)
	MyCourses     = "my_courses"    // «دوره‌های من» row of the «من» hub only (UI-only, no routes of its own)
	Shop          = "shop"          // فروشگاه (catalog, cart, orders, layette checklist)
	Insurance     = "insurance"     // بیمه
	Record        = "record"        // پرونده سلامت (the emergency card stays reachable)
	Devices       = "devices"       // ساعت و گجت‌ها (UI-only, no routes of its own)
	Doctors       = "doctors"       // پزشک و ماما (telemed directory, booking, visits and chat)
	Todo          = "todo"          // لیست کارها
	Reminders     = "reminders"     // «یادآورها» hub + «یادآورهای امروز» card (UI-only: other modules book through /care)
	HomeArticles  = "home_articles" // «بر اساس سیکل فعلی‌ات» articles strip of home (UI-only)
	// «من» hub rows an admin may hide (user request 2026-11-25). UI-only: their APIs serve other screens too
	// (vitals readings feed the log sheet, pregnancy and alerts; support reports; export), so nothing is gated.
	Services = "services" // «خدمات» hub (/services): its «من» row; the sub-hubs keep their own switches
	Vitals   = "vitals"   // «علائم حیاتی» (/vitals): «من» row, the hub tile, the log sheet's add links
	Backup   = "backup"   // «پشتیبان و خروجی داده» (/profile/backup) row of «من»
	Support  = "support"  // «راهنما» / «گفتگو با پشتیبانی» rows of «من» (/profile/support)
)

// Keys lists every module in display order.
var Keys = []string{
	Companion, TTC, IVF, Menopause, Teen, Postpartum, Children, Contraception,
	Programs, MotherChild, Learning, MyCourses, Shop, Insurance, Record, Devices,
	Doctors, Todo, Reminders, HomeArticles, Services, Vitals, Backup, Support,
}

// CodeDisabled is the error_code of the 403 a disabled module's routes answer.
const CodeDisabled = "module_disabled"

// Table is the switches table; Set publishes it (configcache) so every API process reloads its snapshot.
const Table = "app_modules"

// States maps each module key to whether it is enabled (every key present).
type States map[string]bool

// Enabled reports whether key is on (unknown keys are on).
func (s States) Enabled(key string) bool {
	on, ok := s[key]
	return !ok || on
}

// JSON is {key: bool} in Keys order.
func (s States) JSON() map[string]bool {
	out := make(map[string]bool, len(Keys))
	for _, k := range Keys {
		out[k] = s.Enabled(k)
	}
	return out
}

// Service reads and writes the switches. Reads come from an in-process snapshot (configcache, T-M9-13): a gated
// request costs no query and no Redis round trip; a write publishes app_modules so every API process reloads on its
// next read (the 60 s TTL covers a missed message).
type Service struct {
	q      *store.Queries
	states *configcache.Cache[States]
	logger *slog.Logger
}

// NewService wires the service. The Redis client is no longer read here (configcache owns the pub/sub link); the
// parameter stays for the existing call sites.
func NewService(db *sql.DB, _ *cache.Client, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Service{q: store.New(db), logger: logger}
	s.states = configcache.New(Table, []string{Table}, s.load)
	return s
}

func allOn() States {
	s := States{}
	for _, k := range Keys {
		s[k] = true
	}
	return s
}

// States is every module's switch. It never fails: an unreadable table means everything is on (not cached, so the
// next request retries). The map is shared: callers must not modify it.
func (s *Service) States(ctx context.Context) States {
	st, err := s.states.Get(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "modules: unreadable app_modules, treating every module as enabled", "error", err)
		return allOn()
	}
	return st
}

func (s *Service) load(ctx context.Context) (States, error) {
	rows, err := s.q.ListModules(ctx)
	if err != nil {
		return nil, err
	}
	st := allOn()
	for _, r := range rows {
		if slices.Contains(Keys, r.Key) {
			st[r.Key] = r.Enabled
		}
	}
	return st, nil
}

// Enabled reports whether one module is on.
func (s *Service) Enabled(ctx context.Context, key string) bool { return s.States(ctx).Enabled(key) }

// Set stores the given switches (keys outside Keys are ignored by the caller's validation) and publishes the change.
func (s *Service) Set(ctx context.Context, changes map[string]bool, now time.Time) error {
	stamp := sql.NullTime{Time: now, Valid: true}
	for _, k := range Keys {
		on, ok := changes[k]
		if !ok {
			continue
		}
		if err := s.q.UpsertModule(ctx, store.UpsertModuleParams{Key: k, Enabled: on, CreatedAt: stamp, UpdatedAt: stamp}); err != nil {
			configcache.Changed(ctx, Table) // earlier keys may have been stored
			return fmt.Errorf("modules: set %s: %w", k, err)
		}
	}
	configcache.Changed(ctx, Table)
	return nil
}

// routePrefixes maps an /api/v1 path prefix to the module that owns it. A prefix matches the path itself or the
// path followed by "/".
var routePrefixes = []struct{ prefix, module string }{
	{"/api/v1/companions", Companion},
	{"/api/v1/companion", Companion},
	{"/api/v1/fertility", TTC},
	{"/api/v1/analysis/fertility", TTC},
	{"/api/v1/analysis/ttc", TTC},
	{"/api/v1/analysis/postpartum", Postpartum}, // also checked in the handler (T-M12-18: it was gated only there)
	{"/api/v1/ivf", IVF},
	{"/api/v1/menopause", Menopause},
	{"/api/v1/teen", Teen},
	{"/api/v1/postpartum", Postpartum},
	{"/api/v1/children", Children},
	{"/api/v1/contraception", Contraception},
	{"/api/v1/conditions", Programs},
	{"/api/v1/pelvic", Programs},
	{"/api/v1/city-services", MotherChild},
	{"/api/v1/learning", Learning},
	{"/api/v1/shop", Shop},
	{"/api/v1/insurance", Insurance},
	{"/api/v1/health-record", Record},
	{"/api/v1/telemed", Doctors},
	{"/api/v1/todo", Todo},
}

// Prefixes lists every gated path prefix with its module (a copy), so tests can walk the router against the gate
// instead of repeating the table (T-M12-18).
func Prefixes() map[string]string {
	out := make(map[string]string, len(routePrefixes))
	for _, r := range routePrefixes {
		out[r.prefix] = r.module
	}
	return out
}

// AlwaysOpen lists the path prefixes under a gated prefix that no switch closes (a copy).
func AlwaysOpen() []string { return slices.Clone(alwaysOpen) }

// alwaysOpen are paths under a gated prefix that stay reachable whatever the switches: the emergency card (and its
// lock-screen view) is safety information, never hidden with the rest of the health record.
var alwaysOpen = []string{"/api/v1/health-record/emergency-card"}

// ModuleOf is the module owning path, or "".
func ModuleOf(path string) string {
	for _, p := range alwaysOpen {
		if path == p || strings.HasPrefix(path, p+"/") {
			return ""
		}
	}
	for _, r := range routePrefixes {
		if path == r.prefix || strings.HasPrefix(path, r.prefix+"/") {
			return r.module
		}
	}
	return ""
}

// Gate answers a disabled module's routes with 403 module_disabled (never 401: clients wipe the session on 401).
// Other paths cost nothing.
func Gate(s *Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		m := ModuleOf(c.Path())
		if m == "" || c.Method() == fiber.MethodOptions || s.Enabled(c.Context(), m) {
			return c.Next()
		}
		return httpx.Fail(fiber.StatusForbidden, "This section is not available.", "error_code", CodeDisabled, "module", m)
	}
}

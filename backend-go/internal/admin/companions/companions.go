// Package companions is the admin «همدم» module (B-N4-07, docs/go-migration/admin-api.md §16) under
// /api/admin/v1/companions/*:
//
//   - /companions/tips — the companion panel's «امروز چه کار کنی؟» copy per partner phase and language: one note
//     and up to three tips. Stored as the fixed message_contents slots of group companion_tip (registry
//     CompanionTipGroup: `<phase>_note` {body}, `<phase>_tip_<n>` {title, body}, `{name}` placeholder) that
//     internal/companion/home reads; no migration. A save writes every slot of the phase for each sent language
//     (live: active + approved), so the language no longer falls back to the default language or the built-in copy;
//     a reset deletes them again.
//   - /companions/links — a read-only overview of companion links: counts by status and type and a list with masked
//     names and mobiles. Never an invite code (or its hash), never what is shared, never the owner's health data.
//
// Every route: any active admin (kit.Admin — tips are content like /messages; the overview is masked metadata like
// the subscriptions list). Mutations pass the CSRF check of the admin chain and write an audit line.
package companions

import (
	"database/sql"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
)

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Handlers serve /companions/*.
type Handlers struct {
	db     *sql.DB
	q      *store.Queries
	logger *slog.Logger
}

// New wires the module.
func New(sqlDB *sql.DB, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{db: sqlDB, q: store.New(sqlDB), logger: logger}
}

// Routes mounts the module (any active admin; static paths before params).
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	a := kit.Admin
	route(fiber.MethodGet, "/companions/tips", a(h.ListTips))
	route(fiber.MethodGet, "/companions/tips/:phase", a(h.ShowTips))
	route(fiber.MethodPut, "/companions/tips/:phase", a(h.UpdateTips))
	route(fiber.MethodDelete, "/companions/tips/:phase", a(h.ResetTips))
	route(fiber.MethodGet, "/companions/links", a(h.ListLinks))
}

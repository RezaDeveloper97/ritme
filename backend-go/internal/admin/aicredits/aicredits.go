// Package aicredits is the admin side of the AI credits system (goose 00125, docs/ai-credits.md,
// docs/go-migration/admin-api.md §29):
//
//   - /ai/models          the model catalogue (ai_models) + a sync from a provider gateway's model list;
//   - /ai/features        the feature slots (ai_features): model, fallback model, credit cost, on/off;
//   - /ai/credit-packs    the purchasable credit packs (ai_credit_packs);
//   - /users/:id/ai-credits  a user's balance, buckets and recent uses + a manual gift / deduction.
//
// Every catalogue write publishes its table (httpadmin.Publishes) so the running API's credits.Catalog reloads.
// Credit writes go through internal/ai/credits.Service only; this package reads the ledger, never writes it.
// API keys are env-only (internal/ai) and never part of this API.
//
// Roles: reads are open to any admin whose role holds the route's capability (`ai/*` → config, `ai/credit-packs` and
// `users/:id/ai-credits` → billing); every write needs a super admin.
package aicredits

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Error codes of this module (besides the generic ones of admin-api.md §2).
const (
	CodeInUse                 = "ai_model_in_use"
	CodeProviderNotConfigured = "ai_provider_not_configured"
	CodeProviderUnavailable   = "ai_provider_unavailable"
	CodeModelLacksCapability  = "ai_model_lacks_capability"
	CodeModelInactive         = "ai_model_inactive"
	CodeFallbackSameAsModel   = "ai_fallback_same_as_model"
	CodeCapabilityStillNeeded = "ai_capability_in_use"
)

// Gateway is the slice of *ai.Client the catalogue sync needs.
type Gateway interface {
	ConfiguredProviders() []string
	ProviderModels(ctx context.Context, provider string) ([]string, error)
}

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Handlers serve the module.
type Handlers struct {
	db      *sql.DB
	q       *store.Queries
	credits *credits.Service
	gateway Gateway // nil = no provider configured (sync answers ai_provider_not_configured)
	logger  *slog.Logger
}

// New wires the module. gateway may be nil.
func New(sqlDB *sql.DB, gateway Gateway, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{db: sqlDB, q: store.New(sqlDB), credits: credits.NewService(sqlDB), gateway: gateway, logger: logger}
}

// Routes mounts the module. Reads: kit.Admin (capability check). Writes: super admin. Static paths before :id.
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	a, s := kit.Admin, kit.Super
	get, post, put, del := fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete
	models := func(fn fiber.Handler) fiber.Handler { return httpadmin.Publishes(fn, credits.ModelsTable) }
	features := func(fn fiber.Handler) fiber.Handler { return httpadmin.Publishes(fn, credits.FeaturesTable) }

	route(get, "/ai/models", a(h.ListModels))
	route(post, "/ai/models", s(models(h.StoreModel)))
	route(post, "/ai/models/sync", s(models(h.SyncModels)))
	route(get, "/ai/models/:id", a(h.ShowModel))
	route(put, "/ai/models/:id", s(models(h.UpdateModel)))
	route(del, "/ai/models/:id", s(models(h.DestroyModel)))

	route(get, "/ai/features", a(h.ListFeatures))
	route(put, "/ai/features/:id", s(features(h.UpdateFeature)))

	route(get, "/ai/credit-packs", a(h.ListPacks))
	route(post, "/ai/credit-packs", s(h.StorePack))
	route(get, "/ai/credit-packs/:id", a(h.ShowPack))
	route(put, "/ai/credit-packs/:id", s(h.UpdatePack))
	route(del, "/ai/credit-packs/:id", s(h.DestroyPack))

	route(get, "/users/:id/ai-credits", a(h.ShowUserCredits))
	route(post, "/users/:id/ai-credits/adjust", s(h.AdjustUserCredits))
}

// ---------------------------------------------------------------------------
// Shared helpers

// now is the request time as the rows store it: Tehran wall-clock, whole seconds.
func now(c fiber.Ctx) time.Time {
	return httpadmin.Now(c).In(civildate.Tehran).Truncate(time.Second)
}

func nullU64(v uint64) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(v), Valid: true} //nolint:gosec // G115: ids fit in int64
}

func nullID(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

// ledger is one plus_admin_actions row (the durable money / entitlement trail billing writes too).
type ledger struct {
	name, targetType string
	targetID, userID uint64
	amount           uint64
	note             string
	details          any
}

// record writes the ledger row through q (joining the caller's transaction when there is one) and the audit line
// `plus.<name>`, like internal/admin/billing.
func (h *Handlers) record(c fiber.Ctx, q *store.Queries, l ledger, attrs ...slog.Attr) error {
	p := store.InsertPlusAdminActionParams{
		Action: l.name, TargetType: l.targetType, TargetID: l.targetID,
		Note: sql.NullString{String: l.note, Valid: l.note != ""},
		Now:  httpadmin.DBTime(httpadmin.Now(c)),
	}
	if adm := httpadmin.CurrentAdmin(c); adm != nil {
		p.AdminID = nullU64(adm.ID)
	}
	if l.userID > 0 {
		p.UserID = nullU64(l.userID)
	}
	if l.amount > 0 {
		p.AmountRials = nullU64(l.amount)
	}
	if l.details != nil {
		p.Details = db.NullRawJSON{V: form.JSON(l.details), Valid: true}
	}
	if err := q.InsertPlusAdminAction(c.Context(), p); err != nil {
		return err
	}
	if l.userID > 0 {
		attrs = append(attrs, slog.Uint64("user_id", l.userID))
	}
	httpadmin.Audit(c, h.logger, "plus."+l.name, l.targetType, l.targetID, attrs...)
	return nil
}

// inTx runs fn in a READ COMMITTED transaction.
func (h *Handlers) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := h.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	if err := fn(h.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

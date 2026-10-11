// Package admin is the internal-shop admin (CB-SHOP-05) under /api/admin/v1/shop/*, next to bloom's link-out
// categories / products CRUD in package shop (B-N10-03), which it extends additively (no second shop section):
// sellers, sub-category parents, the commerce fields and images of a product, variants and stock, delivery slots,
// the shipment queue (confirm → ship → deliver, cancel), returns, review moderation, discount codes and the layette
// checklist template.
//
// Capabilities (httpadmin.capabilityOf): the catalog, reviews, codes and template are «content»; the shipment queue,
// order detail and returns carry the recipient's name, phone and address, so they are «support» only. No handler
// logs order or address data: audit lines carry ids and statuses only.
package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/shop/order"
	"github.com/ritme/backend-go/internal/shop/store"
)

// Error codes of the 422 refusals.
const (
	CodeInUse             = "in_use"
	CodeInvalidTransition = "invalid_transition"
	CodeNotPending        = "not_pending"
	CodeSlotBooked        = "slot_booked"
)

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Admin serves the CB-SHOP-05 endpoints.
type Admin struct {
	db     *sql.DB
	q      *store.Queries
	orders *order.Service
	disk   *media.Disk
	opt    media.Optimizer
	appURL string
	logger *slog.Logger
}

// New builds the handlers.
func New(sqlDB *sql.DB, disk *media.Disk, appURL string, logger *slog.Logger) *Admin {
	if logger == nil {
		logger = slog.Default()
	}
	if disk == nil {
		disk = media.NewDisk("")
	}
	return &Admin{db: sqlDB, q: store.New(sqlDB), orders: order.NewService(sqlDB), disk: disk,
		opt: media.DefaultOptimizer, appURL: appURL, logger: logger}
}

// Routes registers the endpoints (static paths before :id). bloom's /shop/categories|products routes stay in
// package shop; these add sub-paths and new resources only.
func (a *Admin) Routes(route Route, kit *httpadmin.Kit) {
	k := kit.Admin
	get, post, put, del := fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete

	// Catalog («content»).
	route(get, "/shop/commerce-options", k(a.Options))
	route(get, "/shop/sellers", k(a.ListSellers))
	route(post, "/shop/sellers", k(a.StoreSeller))
	route(get, "/shop/sellers/:id", k(a.ShowSeller))
	route(put, "/shop/sellers/:id", k(a.UpdateSeller))
	route(del, "/shop/sellers/:id", k(a.DestroySeller))
	route(put, "/shop/categories/:id/parent", k(a.SetCategoryParent))
	route(get, "/shop/products/:id/commerce", k(a.ShowCommerce))
	route(put, "/shop/products/:id/commerce", k(a.UpdateCommerce))
	route(post, "/shop/products/:id/images", k(a.UploadImage))
	route(get, "/shop/products/:id/variants", k(a.ListVariants))
	route(post, "/shop/products/:id/variants", k(a.StoreVariant))
	route(put, "/shop/variants/:id", k(a.UpdateVariant))
	route(del, "/shop/variants/:id", k(a.DestroyVariant))
	route(post, "/shop/variants/:id/stock", k(a.SetStock))
	route(get, "/shop/stock", k(a.ListStock))
	route(get, "/shop/slots", k(a.ListSlots))
	route(post, "/shop/slots/generate", k(a.GenerateSlots))
	route(put, "/shop/slots/:id", k(a.UpdateSlot))
	route(del, "/shop/slots/:id", k(a.DestroySlot))
	route(get, "/shop/reviews", k(a.ListReviews))
	route(post, "/shop/reviews/:id/moderate", k(a.ModerateReview))
	route(del, "/shop/reviews/:id", k(a.DestroyReview))
	route(get, "/shop/discount-codes", k(a.ListDiscountCodes))
	route(post, "/shop/discount-codes", k(a.StoreDiscountCode))
	route(get, "/shop/discount-codes/:id", k(a.ShowDiscountCode))
	route(put, "/shop/discount-codes/:id", k(a.UpdateDiscountCode))
	route(del, "/shop/discount-codes/:id", k(a.DestroyDiscountCode))
	route(get, "/shop/layette-items", k(a.ListLayetteItems))
	route(post, "/shop/layette-items", k(a.StoreLayetteItem))
	route(get, "/shop/layette-items/:id", k(a.ShowLayetteItem))
	route(put, "/shop/layette-items/:id", k(a.UpdateLayetteItem))
	route(del, "/shop/layette-items/:id", k(a.DestroyLayetteItem))

	// Orders («support»: recipient PII).
	route(get, "/shop/shipments", k(a.ListShipments))
	route(post, "/shop/shipments/:id/move", k(a.MoveShipment))
	route(get, "/shop/orders/:id", k(a.ShowOrder))
	route(post, "/shop/orders/:id/cancel", k(a.CancelOrder))
	route(get, "/shop/returns", k(a.ListReturns))
	route(post, "/shop/returns/:id/approve", k(a.ApproveReturn))
	route(post, "/shop/returns/:id/reject", k(a.RejectReturn))
}

// ─── helpers ───────────────────────────────────────────────────────────────────────────────────────────────────────

func (a *Admin) now(c fiber.Ctx) sql.NullTime { return httpadmin.DBTime(httpadmin.Now(c)) }

func (a *Admin) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(a.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// trans decodes a translatable JSON text column for output.
func trans(s string) any { return form.Raw(json.RawMessage(s)) }

// ntrans is trans for a nullable column.
func ntrans(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return trans(s.String)
}

// nraw converts a cleaned translatable value to the text column sqlc expects.
func nraw(v db.NullRawJSON) sql.NullString {
	if !v.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: string(v.V), Valid: true}
}

func trimmed(data phpval.Map, key string) sql.NullString {
	s := form.Str(data, key)
	if !s.Valid {
		return s
	}
	s.String = strings.TrimSpace(s.String)
	if s.String == "" {
		return sql.NullString{}
	}
	return s
}

func nullInt64(data phpval.Map, key string) sql.NullInt64 {
	if !form.Has(data, key) {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: form.Int(data, key, 0), Valid: true}
}

func uint64Of(data phpval.Map, key string) uint64 {
	n := form.Int(data, key, 0)
	if n < 0 {
		return 0
	}
	return uint64(n)
}

func nullInt(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func parseID(s string) (uint64, bool) {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n, err == nil && n > 0
}

func idOf(n uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(n), Valid: n > 0} } //nolint:gosec // ids fit

// exists is a validation check that an optional id field names a row.
func exists(c fiber.Ctx, field string, find func(id uint64) error) form.Check {
	return func(in phpval.Map, add form.Add) error {
		v, ok := in.Get(field)
		if !ok || v == nil || phpval.ToString(v) == "" {
			return nil
		}
		id, ok := parseID(phpval.ToString(v))
		if !ok {
			add(field, form.Msg(c, "validation.exists", field))
			return nil
		}
		err := find(id)
		if errors.Is(err, sql.ErrNoRows) {
			add(field, form.Msg(c, "validation.exists", field))
			return nil
		}
		return err
	}
}

func (a *Admin) imageURL(path string) string { return content.PublicURL(a.appURL, path) }

// orderFail renders an order.Error refusal with the order package's copy.
func orderFail(c fiber.Ctx, err error) error {
	var oe *order.Error
	if errors.As(err, &oe) {
		if oe.Status == fiber.StatusNotFound {
			return httpadmin.NotFound("Order")
		}
		return httpadmin.Fail(oe.Status, oe.Code, order.T("errors."+oe.Code, i18n.Locale(c)))
	}
	return err
}

func notFound(err error, what string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound(what)
	}
	return err
}

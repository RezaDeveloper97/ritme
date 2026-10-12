package admin

import (
	"database/sql"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/shop/order"
	"github.com/ritme/backend-go/internal/shop/store"
)

// Return statuses this admin writes (the order package files them as requested).
const (
	ReturnRequested = "requested"
	ReturnApproved  = "approved"
	ReturnRejected  = "rejected"

	eventReturnApproved = "return_approved"
	eventReturnRejected = "return_rejected"
)

// NextShipmentStatuses is the shipment machine of the admin queue: awaiting_seller → confirmed → shipped →
// delivered (confirmed may go straight to delivered for a seller who delivers in person). Cancelling is per order
// (POST /shop/orders/:id/cancel) while nothing has shipped.
func NextShipmentStatuses(status string) []string {
	switch status {
	case order.ShipmentAwaiting:
		return []string{order.ShipmentConfirmed}
	case order.ShipmentConfirmed:
		return []string{order.ShipmentShipped, order.StatusDelivered}
	case order.ShipmentShipped:
		return []string{order.StatusDelivered}
	}
	return []string{}
}

func slotOf(date interface{ String() string }, valid bool, start, end sql.NullString) any {
	if !valid {
		return nil
	}
	return jsonx.Obj("date", date.String(), "starts_at", hhmm(start.String), "ends_at", hhmm(end.String))
}

// ListShipments is GET /shop/shipments?status=&seller_id=&q=<order code>&page=: the queue, one row per seller
// shipment, newest first. Rows carry the recipient's name and city (support capability only).
func (a *Admin) ListShipments(c fiber.Ctx) error {
	sellerID, _ := parseID(c.Query("seller_id"))
	f := store.AdminListShipmentsParams{Status: strings.TrimSpace(c.Query("status")), SellerID: sellerID,
		Code: strings.TrimSpace(c.Query("q"))}
	if f.Status == "all" {
		f.Status = ""
	}
	p := httpadmin.PageOf(c)
	f.Limit, f.Offset = int32(p.PerPage), int32(p.Offset()) //nolint:gosec // bounded paging
	rows, err := a.q.AdminListShipments(c, f)
	if err != nil {
		return err
	}
	total, err := a.q.AdminCountShipments(c, store.AdminCountShipmentsParams{Status: f.Status, SellerID: f.SellerID, Code: f.Code})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		name, err := a.orders.OpenRecipientName(r.OrderUserID, r.PiiEnc, r.RecipientName)
		if err != nil {
			return piiUnavailable()
		}
		items = append(items, jsonx.Obj("id", r.ID, "order_id", r.OrderID, "order_code", r.OrderCode,
			"order_status", r.OrderStatus, "seller", jsonx.Obj("id", r.SellerID, "name", trans(r.SellerName)),
			"status", r.Status, "allowed_statuses", NextShipmentStatuses(r.Status),
			"slot", slotOf(r.SlotDate.Date, r.SlotDate.Valid, r.SlotStartsAt, r.SlotEndsAt),
			"items_total", r.ItemsTotal, "shipping_fee", r.ShippingFee, "payable_total", r.PayableTotal,
			"discreet", r.Discreet, "tracking_code", httpadmin.NullString(r.TrackingCode),
			"recipient_name", name, "city", r.City,
			"delivered_at", httpadmin.Time(r.DeliveredAt), "created_at", httpadmin.Time(r.CreatedAt)))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// MoveShipment is POST /shop/shipments/:id/move {status, tracking_code?}. delivered goes through
// order.Service.MarkShipmentDelivered (timeline, order delivered when all are, checklist «دارم»).
func (a *Admin) MoveShipment(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Shipment")
	}
	sh, err := a.q.GetShipment(c, id)
	if err != nil {
		return notFound(err, "Shipment")
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("status", "required|string|in:"+order.ShipmentConfirmed+","+order.ShipmentShipped+","+order.StatusDelivered),
		validation.F("tracking_code", "nullable|string|max:64"),
	})
	if err != nil {
		return err
	}
	to := httpadmin.String(data, "status")
	allowed := false
	for _, s := range NextShipmentStatuses(sh.Status) {
		allowed = allowed || s == to
	}
	if !allowed {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInvalidTransition,
			"This shipment cannot move to that status.", "status", sh.Status)
	}
	now := httpadmin.Now(c)
	if to == order.StatusDelivered {
		if err := a.orders.MarkShipmentDelivered(c, sh.ID, now); err != nil {
			return orderFail(c, err)
		}
	} else {
		err := a.inTx(c, func(q *store.Queries) error {
			n, err := q.AdminMoveShipment(c, store.AdminMoveShipmentParams{ToStatus: to, TrackingCode: trimmed(data, "tracking_code"),
				UpdatedAt: httpadmin.DBTime(now), ID: sh.ID, FromStatus: sh.Status})
			if err != nil {
				return err
			}
			if n == 0 {
				return httpadmin.Fail(fiber.StatusConflict, CodeInvalidTransition, "This shipment changed meanwhile. Reload and try again.")
			}
			_, err = q.CreateOrderEvent(c, store.CreateOrderEventParams{OrderID: sh.OrderID, ShipmentID: idOf(sh.ID),
				Status: to, CreatedAt: httpadmin.DBTime(now)})
			return err
		})
		if err != nil {
			return err
		}
	}
	httpadmin.Audit(c, a.logger, "shop_shipment.move", "shop_shipment", sh.ID,
		slog.String("from", sh.Status), slog.String("to", to), slog.Uint64("order_id", sh.OrderID))
	return a.orderOut(c, sh.OrderID, "Shipment updated.")
}

// ShowOrder is GET /shop/orders/:id: the order with recipient, address, shipments, items, timeline and returns.
func (a *Admin) ShowOrder(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Order")
	}
	return a.orderOut(c, id)
}

func (a *Admin) orderOut(c fiber.Ctx, id uint64, msg ...string) error {
	o, err := a.q.AdminGetOrder(c, id)
	if err != nil {
		return notFound(err, "Order")
	}
	if o, err = a.orders.OpenOrder(o); err != nil {
		return piiUnavailable()
	}
	ships, err := a.q.ListShipmentsByOrder(c, o.ID)
	if err != nil {
		return err
	}
	items, err := a.q.ListOrderItemsByOrder(c, o.ID)
	if err != nil {
		return err
	}
	events, err := a.q.ListOrderEvents(c, o.ID)
	if err != nil {
		return err
	}
	returns, err := a.q.AdminListReturnsByOrder(c, o.ID)
	if err != nil {
		return err
	}
	sellerIDs := []uint64{}
	for _, s := range ships {
		sellerIDs = append(sellerIDs, s.SellerID)
	}
	names := map[uint64]string{}
	if len(sellerIDs) > 0 {
		sellers, err := a.q.ListSellersByIDs(c, sellerIDs)
		if err != nil {
			return err
		}
		for _, s := range sellers {
			names[s.ID] = s.Name
		}
	}
	d := &order.Detail{Order: o, Shipments: ships}
	shipsOut := make([]*jsonx.OrderedMap, 0, len(ships))
	for _, s := range ships {
		lines := []*jsonx.OrderedMap{}
		for _, it := range items {
			if it.ShipmentID == s.ID {
				lines = append(lines, jsonx.Obj("id", it.ID, "product_id", nullInt(it.ProductID), "variant_id", nullInt(it.VariantID),
					"title", trans(it.Title), "variant_label", ntrans(it.VariantLabel), "unit_price", it.UnitPrice,
					"quantity", it.Quantity, "line_total", it.LineTotal))
			}
		}
		var seller any
		if n, ok := names[s.SellerID]; ok {
			seller = trans(n)
		}
		shipsOut = append(shipsOut, jsonx.Obj("id", s.ID, "seller", jsonx.Obj("id", s.SellerID, "name", seller),
			"status", s.Status, "allowed_statuses", NextShipmentStatuses(s.Status),
			"slot", slotOf(s.SlotDate.Date, s.SlotDate.Valid, s.SlotStartsAt, s.SlotEndsAt),
			"items_total", s.ItemsTotal, "shipping_fee", s.ShippingFee, "tracking_code", httpadmin.NullString(s.TrackingCode),
			"delivered_at", httpadmin.Time(s.DeliveredAt), "items", lines))
	}
	evOut := make([]*jsonx.OrderedMap, 0, len(events))
	for _, e := range events {
		evOut = append(evOut, jsonx.Obj("id", e.ID, "shipment_id", nullInt(e.ShipmentID), "status", e.Status,
			"created_at", httpadmin.Time(e.CreatedAt)))
	}
	retOut := make([]*jsonx.OrderedMap, 0, len(returns))
	for _, r := range returns {
		retOut = append(retOut, jsonx.Obj("id", r.ID, "order_item_id", r.OrderItemID, "quantity", r.Quantity,
			"reason", r.Reason, "note", httpadmin.NullString(r.Note), "status", r.Status, "created_at", httpadmin.Time(r.CreatedAt)))
	}
	return httpadmin.OK(c, jsonx.Obj("order", jsonx.Obj(
		"id", o.ID, "code", o.Code, "status", o.Status, "payment_method", o.PaymentMethod, "payment_status", o.PaymentStatus,
		"discreet", o.Discreet, "recipient_name", o.RecipientName, "recipient_phone", o.RecipientPhone, "city", o.City,
		"address_text", o.AddressText, "postal_code", httpadmin.NullString(o.PostalCode),
		"items_total", o.ItemsTotal, "shipping_total", o.ShippingTotal, "discount_total", o.DiscountTotal,
		"payable_total", o.PayableTotal, "discount_code_id", nullInt(o.DiscountCodeID),
		"placed_at", httpadmin.Time(o.PlacedAt), "cancelled_at", httpadmin.Time(o.CancelledAt),
		"can_cancel", d.CanAdminCancel(), "shipments", shipsOut, "events", evOut, "returns", retOut)), msg...)
}

// CancelOrder is POST /shop/orders/:id/cancel: while placed and nothing has shipped (order.Service.CancelByAdmin
// gives stock, slots, the code use and checklist rows back).
func (a *Admin) CancelOrder(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Order")
	}
	if err := a.orders.CancelByAdmin(c, id, httpadmin.Now(c)); err != nil {
		return orderFail(c, err)
	}
	httpadmin.Audit(c, a.logger, "shop_order.cancel", "shop_order", id)
	return a.orderOut(c, id, "Order cancelled.")
}

// ─── returns ───────────────────────────────────────────────────────────────────────────────────────────────────────

// ListReturns is GET /shop/returns?status=requested|approved|rejected|all&page=.
func (a *Admin) ListReturns(c fiber.Ctx) error {
	status := strings.TrimSpace(c.Query("status"))
	if status == "all" {
		status = ""
	}
	p := httpadmin.PageOf(c)
	rows, err := a.q.AdminListReturns(c, store.AdminListReturnsParams{Status: status, Limit: int32(p.PerPage), Offset: int32(p.Offset())}) //nolint:gosec // bounded
	if err != nil {
		return err
	}
	total, err := a.q.AdminCountReturns(c, store.AdminCountReturnsParams{Status: status})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonx.Obj("id", r.ID, "order_id", r.OrderID, "order_code", r.OrderCode,
			"order_item_id", r.OrderItemID, "item_title", trans(r.ItemTitle), "variant_label", ntrans(r.VariantLabel),
			"unit_price", r.UnitPrice, "quantity", r.Quantity, "reason", r.Reason, "note", httpadmin.NullString(r.Note),
			"status", r.Status, "can_restock", r.VariantID.Valid,
			"created_at", httpadmin.Time(r.CreatedAt), "updated_at", httpadmin.Time(r.UpdatedAt)))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// ApproveReturn is POST /shop/returns/:id/approve {restock?}: the return is accepted (cash is settled offline: COD);
// restock puts the quantity back on the variant once the item is back.
func (a *Admin) ApproveReturn(c fiber.Ctx) error { return a.decideReturn(c, ReturnApproved) }

// RejectReturn is POST /shop/returns/:id/reject.
func (a *Admin) RejectReturn(c fiber.Ctx) error { return a.decideReturn(c, ReturnRejected) }

func (a *Admin) decideReturn(c fiber.Ctx, status string) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Return")
	}
	r, err := a.q.AdminGetReturn(c, id)
	if err != nil {
		return notFound(err, "Return")
	}
	data, err := form.Validate(c, validation.Rules{validation.F("restock", "nullable|boolean")})
	if err != nil {
		return err
	}
	restock := status == ReturnApproved && httpadmin.Bool(data, "restock") && r.VariantID.Valid
	now := a.now(c)
	event := eventReturnRejected
	if status == ReturnApproved {
		event = eventReturnApproved
	}
	err = a.inTx(c, func(q *store.Queries) error {
		n, err := q.AdminDecideReturn(c, store.AdminDecideReturnParams{Status: status, UpdatedAt: now, ID: r.ID})
		if err != nil {
			return err
		}
		if n == 0 {
			return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeNotPending, "This return was already decided.")
		}
		if restock {
			if err := q.RestockVariant(c, store.RestockVariantParams{Stock: uint32(r.Quantity), UpdatedAt: now,
				ID: uint64(r.VariantID.Int64)}); err != nil { //nolint:gosec // ids fit
				return err
			}
		}
		_, err = q.CreateOrderEvent(c, store.CreateOrderEventParams{OrderID: r.OrderID, Status: event, CreatedAt: now})
		return err
	})
	if err != nil {
		return err
	}
	httpadmin.Audit(c, a.logger, "shop_return."+status, "shop_return", r.ID, slog.Bool("restock", restock))
	return httpadmin.OK(c, jsonx.Obj("id", r.ID, "status", status), "Return updated.")
}

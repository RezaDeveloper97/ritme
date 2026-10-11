// Package order is the internal shop's cart, pay-on-delivery checkout, orders and returns (CB-SHOP-03,
// docs/canvas-build/shop.md). MVP = COD only (DECISIONS #13): there is no gateway and no payment adapter here.
//
// Concurrency: a checkout runs in one transaction that first locks the user's cart row (serialising one user's
// checkouts, which makes the idempotency key check race-free), then takes stock with guarded UPDATEs in variant-id
// order (`stock >= qty`), then the discount code use and delivery-slot capacity (`booked < capacity`) the same way.
// A guard that matches no row aborts the whole order, so stock and slots are never oversold.
package order

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/shop/store"
)

// Limits.
const (
	MaxLineQuantity = 10  // one variant in a cart
	MaxCartLines    = 30  // distinct variants in a cart
	MaxAddresses    = 20  // saved addresses per user
	MaxOpenOrders   = 5   // placed orders not yet shipped, per user (security audit: stock / slot hoarding)
	OrdersListed    = 50  // GET /shop/orders
	codeAttempts    = 5   // order-code collisions retried
	codeSpace       = 1e8 // 8-digit order codes
)

// Order and shipment statuses this package writes (CB-SHOP-05's admin moves shipments through
// confirmed → shipped → delivered).
const (
	StatusPlaced         = "placed"
	StatusCancelled      = "cancelled"
	StatusDelivered      = "delivered"
	ShipmentAwaiting     = "awaiting_seller"
	ShipmentConfirmed    = "confirmed" // CB-SHOP-05 admin: the seller accepted the shipment
	ShipmentShipped      = "shipped"   // CB-SHOP-05 admin: handed to the courier
	EventReturnRequested = "return_requested"
)

// ReturnReasons are the accepted return reasons.
var ReturnReasons = []string{"wrong_size", "damaged", "not_as_described", "changed_mind", "other"}

// Error is a business refusal the handlers render as {success:false, message, error_code[, errors]}.
type Error struct {
	Status     int
	Code       string
	Field      string   // set → a 422-style errors bag on this field
	VariantIDs []uint64 // out_of_stock / cart_unavailable: the lines concerned
}

func (e *Error) Error() string { return "order: " + e.Code }

func fail(status int, code, field string) *Error {
	return &Error{Status: status, Code: code, Field: field}
}

// Service runs the order use cases.
type Service struct {
	db *sql.DB
	q  *store.Queries
}

// NewService wraps db.
func NewService(db *sql.DB) *Service { return &Service{db: db, q: store.New(db)} }

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("order: begin: %w", err)
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		if isDeadlock(err) {
			return fail(409, "busy", "")
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		if isDeadlock(err) {
			return fail(409, "busy", "")
		}
		return fmt.Errorf("order: commit: %w", err)
	}
	return nil
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func isDeadlock(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && (me.Number == 1213 || me.Number == 1205)
}

func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func id64(n sql.NullInt64) uint64 {
	if !n.Valid || n.Int64 < 0 {
		return 0
	}
	return uint64(n.Int64)
}

// ─── cart ───────────────────────────────────────────────────────────────────────────────────────────────────────────

// Cart is the user's priced cart (an empty one when she never added anything).
func (s *Service) Cart(ctx context.Context, userID uint64, now time.Time) (*Cart, error) {
	c, err := s.q.GetCartByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return &Cart{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("order: cart: %w", err)
	}
	return s.price(ctx, s.q, userID, c, now)
}

// price loads and prices a cart row, including its discount code (re-evaluated on every read).
func (s *Service) price(ctx context.Context, q *store.Queries, userID uint64, c store.ShopCart, now time.Time) (*Cart, error) {
	rows, err := q.ListCartLines(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("order: cart lines: %w", err)
	}
	var ids []uint64
	for _, r := range rows {
		if r.SellerID.Valid && !slices.Contains(ids, id64(r.SellerID)) {
			ids = append(ids, id64(r.SellerID))
		}
	}
	sellers := map[uint64]*store.ShopSeller{}
	if len(ids) > 0 {
		list, err := q.ListSellersByIDs(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("order: sellers: %w", err)
		}
		for i := range list {
			sellers[list[i].ID] = &list[i]
		}
	}
	cart := buildCart(c.ID, rows, sellers)
	if c.DiscountCodeID.Valid {
		d, err := q.GetDiscountCode(ctx, id64(c.DiscountCodeID))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order: discount: %w", err)
		}
		if err == nil {
			cart.Discount = &d
			uses, err := q.CountUserOrdersWithDiscount(ctx, store.CountUserOrdersWithDiscountParams{UserID: userID, DiscountCodeID: c.DiscountCodeID})
			if err != nil {
				return nil, fmt.Errorf("order: discount uses: %w", err)
			}
			cart.discountUses = uses
			cart.totals(now)
		}
	}
	return cart, nil
}

// cartTx runs fn in a transaction holding the user's cart row lock. The row is created BEFORE the transaction
// (autocommit on the pool): inside it, a FOR UPDATE that finds no row leaves a gap lock that would block the INSERT,
// and INSERT IGNORE's shared lock on an existing row deadlocks two concurrent FOR UPDATEs.
func (s *Service) cartTx(ctx context.Context, userID uint64, now time.Time, fn func(q *store.Queries, c store.ShopCart) error) error {
	if _, err := s.q.GetCartByUser(ctx, userID); errors.Is(err, sql.ErrNoRows) {
		if err := s.q.EnsureCart(ctx, store.EnsureCartParams{UserID: userID, CreatedAt: nt(now), UpdatedAt: nt(now)}); err != nil {
			return fmt.Errorf("order: ensure cart: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("order: cart: %w", err)
	}
	return s.inTx(ctx, func(q *store.Queries) error {
		c, err := lockCart(ctx, q, userID)
		if err != nil {
			return err
		}
		return fn(q, c)
	})
}

func lockCart(ctx context.Context, q *store.Queries, userID uint64) (store.ShopCart, error) {
	c, err := q.LockCartByUser(ctx, userID)
	if err != nil {
		return c, fmt.Errorf("order: lock cart: %w", err)
	}
	return c, nil
}

// purchasable checks a variant can go into a cart and returns its stock.
func purchasable(ctx context.Context, q *store.Queries, variantID uint64) (uint32, error) {
	v, err := q.GetVariantForCart(ctx, variantID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fail(422, "variant_unavailable", "variant_id")
	}
	if err != nil {
		return 0, fmt.Errorf("order: variant: %w", err)
	}
	if !v.VariantActive || !v.ProductActive || !v.CategoryActive || !v.SellerID.Valid {
		return 0, fail(422, "variant_unavailable", "variant_id")
	}
	sellers, err := q.ListSellersByIDs(ctx, []uint64{id64(v.SellerID)})
	if err != nil {
		return 0, fmt.Errorf("order: seller: %w", err)
	}
	if len(sellers) == 0 || !sellers[0].IsActive {
		return 0, fail(422, "variant_unavailable", "variant_id")
	}
	return v.Stock, nil
}

// SetItem puts a variant in the cart: add=true adds qty to what is there, else qty replaces it (the line must exist).
func (s *Service) SetItem(ctx context.Context, userID, variantID uint64, qty int, add bool, now time.Time) error {
	return s.cartTx(ctx, userID, now, func(q *store.Queries, c store.ShopCart) error {
		existing, err := q.GetCartItemQuantity(ctx, store.GetCartItemQuantityParams{CartID: c.ID, VariantID: variantID})
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if !add {
				return fail(404, "not_found", "")
			}
			items, err := q.ListCartItems(ctx, c.ID)
			if err != nil {
				return fmt.Errorf("order: cart items: %w", err)
			}
			if len(items) >= MaxCartLines {
				return fail(422, "cart_full", "variant_id")
			}
		case err != nil:
			return fmt.Errorf("order: cart item: %w", err)
		}
		stock, err := purchasable(ctx, q, variantID)
		if err != nil {
			return err
		}
		n := qty
		if add {
			n += int(existing)
		}
		if n > MaxLineQuantity {
			return fail(422, "quantity_limit", "quantity")
		}
		if uint32(n) > stock { //nolint:gosec // G115: 1 ≤ n ≤ MaxLineQuantity
			return &Error{Status: 422, Code: "out_of_stock", Field: "quantity", VariantIDs: []uint64{variantID}}
		}
		return q.UpsertCartItem(ctx, store.UpsertCartItemParams{CartID: c.ID, VariantID: variantID, Quantity: uint16(n), //nolint:gosec // G115
			CreatedAt: nt(now), UpdatedAt: nt(now)})
	})
}

// RemoveItem deletes a cart line (404 when it is not in her cart).
func (s *Service) RemoveItem(ctx context.Context, userID, variantID uint64) error {
	c, err := s.q.GetCartByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return fail(404, "not_found", "")
	}
	if err != nil {
		return fmt.Errorf("order: cart: %w", err)
	}
	if _, err := s.q.GetCartItemQuantity(ctx, store.GetCartItemQuantityParams{CartID: c.ID, VariantID: variantID}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fail(404, "not_found", "")
		}
		return fmt.Errorf("order: cart item: %w", err)
	}
	return s.q.DeleteCartItem(ctx, store.DeleteCartItemParams{CartID: c.ID, VariantID: variantID})
}

// ApplyDiscount validates code against the current cart and stores it on the cart.
func (s *Service) ApplyDiscount(ctx context.Context, userID uint64, code string, now time.Time) error {
	return s.cartTx(ctx, userID, now, func(q *store.Queries, c store.ShopCart) error {
		d, err := q.GetDiscountCodeByCode(ctx, code)
		if errors.Is(err, sql.ErrNoRows) {
			return fail(422, "discount_invalid", "code")
		}
		if err != nil {
			return fmt.Errorf("order: discount: %w", err)
		}
		c.DiscountCodeID = sql.NullInt64{}
		cart, err := s.price(ctx, q, userID, c, now)
		if err != nil {
			return err
		}
		uses, err := q.CountUserOrdersWithDiscount(ctx, store.CountUserOrdersWithDiscountParams{UserID: userID,
			DiscountCodeID: sql.NullInt64{Int64: int64(d.ID), Valid: true}}) //nolint:gosec // G115
		if err != nil {
			return fmt.Errorf("order: discount uses: %w", err)
		}
		// One refusal for every reason (unknown, inactive, exhausted, used, below the minimum): applying a code must
		// not tell a guesser which codes exist (security audit). Once applied, the cart names the exact reason.
		if _, why := evaluateDiscount(&d, cart.ItemsTotal, now, uses); why != "" {
			return fail(422, "discount_invalid", "code")
		}
		return q.SetCartDiscountCode(ctx, store.SetCartDiscountCodeParams{
			DiscountCodeID: sql.NullInt64{Int64: int64(d.ID), Valid: true}, UpdatedAt: nt(now), ID: c.ID}) //nolint:gosec // G115
	})
}

// RemoveDiscount clears the cart's code (no-op without a cart).
func (s *Service) RemoveDiscount(ctx context.Context, userID uint64, now time.Time) error {
	c, err := s.q.GetCartByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("order: cart: %w", err)
	}
	return s.q.SetCartDiscountCode(ctx, store.SetCartDiscountCodeParams{UpdatedAt: nt(now), ID: c.ID})
}

// ─── addresses ──────────────────────────────────────────────────────────────────────────────────────────────────────

// AddressInput is a validated address body.
type AddressInput struct {
	Label, RecipientName, RecipientPhone, Province, City, AddressLine, PostalCode string
	IsDefault                                                                     bool
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// Addresses lists her addresses, default first.
func (s *Service) Addresses(ctx context.Context, userID uint64) ([]store.ShopAddress, error) {
	return s.q.ListAddressesByUser(ctx, userID)
}

// SaveAddress creates (id 0) or updates one of her addresses; the first address is the default. 404 for a foreign id.
func (s *Service) SaveAddress(ctx context.Context, userID, id uint64, in AddressInput, now time.Time) (store.ShopAddress, error) {
	var out store.ShopAddress
	err := s.cartTx(ctx, userID, now, func(q *store.Queries, _ store.ShopCart) error { // the cart lock serialises the default flag
		if id == 0 {
			n, err := q.CountAddressesByUser(ctx, userID)
			if err != nil {
				return fmt.Errorf("order: addresses: %w", err)
			}
			if n >= MaxAddresses {
				return fail(422, "address_limit", "address_line")
			}
			last, err := q.CreateAddress(ctx, store.CreateAddressParams{UserID: userID, Label: ns(in.Label), RecipientName: in.RecipientName,
				RecipientPhone: in.RecipientPhone, Province: ns(in.Province), City: in.City, AddressLine: in.AddressLine,
				PostalCode: ns(in.PostalCode), IsDefault: in.IsDefault || n == 0, CreatedAt: nt(now), UpdatedAt: nt(now)})
			if err != nil {
				return fmt.Errorf("order: create address: %w", err)
			}
			id = uint64(last) //nolint:gosec // G115: auto-increment id
		} else {
			cur, err := q.GetAddressForUser(ctx, store.GetAddressForUserParams{ID: id, UserID: userID})
			if errors.Is(err, sql.ErrNoRows) {
				return fail(404, "not_found", "")
			}
			if err != nil {
				return fmt.Errorf("order: address: %w", err)
			}
			if _, err := q.UpdateAddressForUser(ctx, store.UpdateAddressForUserParams{Label: ns(in.Label), RecipientName: in.RecipientName,
				RecipientPhone: in.RecipientPhone, Province: ns(in.Province), City: in.City, AddressLine: in.AddressLine,
				PostalCode: ns(in.PostalCode), IsDefault: in.IsDefault || cur.IsDefault, UpdatedAt: nt(now), ID: id, UserID: userID}); err != nil {
				return fmt.Errorf("order: update address: %w", err)
			}
		}
		a, err := q.GetAddressForUser(ctx, store.GetAddressForUserParams{ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("order: address: %w", err)
		}
		if a.IsDefault {
			if err := q.ClearDefaultAddress(ctx, store.ClearDefaultAddressParams{UserID: userID, ID: id}); err != nil {
				return fmt.Errorf("order: default address: %w", err)
			}
		}
		out = a
		return nil
	})
	return out, err
}

// DeleteAddress removes one of her addresses (orders keep their snapshot). 404 for a foreign id.
func (s *Service) DeleteAddress(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteAddressForUser(ctx, store.DeleteAddressForUserParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("order: delete address: %w", err)
	}
	if n == 0 {
		return fail(404, "not_found", "")
	}
	return nil
}

// ─── checkout ───────────────────────────────────────────────────────────────────────────────────────────────────────

// Checkout is the review step: priced cart, addresses, open delivery slots per seller, COD availability.
type Checkout struct {
	Cart      *Cart
	Addresses []store.ShopAddress
	Slots     map[uint64][]store.ShopDeliverySlot
	COD       bool
	Blockers  []string
}

// Payable is items − discount + shipping.
func (c *Cart) Payable() uint64 { return c.ItemsTotal - c.DiscountAmount + c.ShippingTotal() }

// Checkout builds the review step. Slots start tomorrow (Tehran).
func (s *Service) Checkout(ctx context.Context, userID uint64, now time.Time) (*Checkout, error) {
	cart, err := s.Cart(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	addrs, err := s.q.ListAddressesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("order: addresses: %w", err)
	}
	co := &Checkout{Cart: cart, Addresses: addrs, Slots: map[uint64][]store.ShopDeliverySlot{}, COD: true}
	from := civildate.FromTime(now).AddDays(1)
	for _, g := range cart.Groups {
		slots, err := s.q.ListOpenDeliverySlots(ctx, store.ListOpenDeliverySlotsParams{SellerID: g.Seller.ID, FromDate: from})
		if err != nil {
			return nil, fmt.Errorf("order: slots: %w", err)
		}
		co.Slots[g.Seller.ID] = slots
		if !codAllowed(g) {
			co.COD = false
		}
	}
	if cart.ItemCount == 0 {
		co.Blockers = append(co.Blockers, "cart_empty")
	}
	if !cart.Purchasable() {
		co.Blockers = append(co.Blockers, "cart_unavailable")
	}
	if cart.DiscountError != "" {
		co.Blockers = append(co.Blockers, cart.DiscountError)
	}
	if len(addrs) == 0 {
		co.Blockers = append(co.Blockers, "address_invalid")
	}
	if !co.COD {
		co.Blockers = append(co.Blockers, "cod_limit_exceeded")
	}
	return co, nil
}

// PlaceInput is a validated POST /shop/orders body. Slots maps seller id → delivery slot id (optional per seller).
type PlaceInput struct {
	Key       string
	AddressID uint64
	Discreet  bool
	Slots     map[uint64]uint64
}

// Placed is the result of Place.
type Placed struct {
	OrderID  uint64
	Replay   bool  // the idempotency key matched an earlier order: nothing new was placed
	OnTheWay int64 // checklist rows moved to «در راه»
	Notice   *Notice
}

// Place turns the cart into a COD order (see the package doc for the locking order).
func (s *Service) Place(ctx context.Context, userID uint64, in PlaceInput, defaultLang string, now time.Time) (*Placed, error) {
	if o, err := s.q.GetOrderByIdempotencyKey(ctx, store.GetOrderByIdempotencyKeyParams{UserID: userID, IdempotencyKey: ns(in.Key)}); err == nil {
		return &Placed{OrderID: o.ID, Replay: true}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("order: idempotency: %w", err)
	}
	var out *Placed
	err := s.cartTx(ctx, userID, now, func(q *store.Queries, c store.ShopCart) error {
		// After the lock: a concurrent twin of this request has committed by now.
		if o, err := q.GetOrderByIdempotencyKey(ctx, store.GetOrderByIdempotencyKeyParams{UserID: userID, IdempotencyKey: ns(in.Key)}); err == nil {
			out = &Placed{OrderID: o.ID, Replay: true}
			return nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("order: idempotency: %w", err)
		}
		// The cap is read under the cart lock, so two parallel checkouts of one user are serialized here.
		if open, err := q.CountOpenOrdersByUser(ctx, userID); err != nil {
			return fmt.Errorf("order: open orders: %w", err)
		} else if open >= MaxOpenOrders {
			return fail(422, "too_many_open_orders", "")
		}
		addr, err := q.GetAddressForUser(ctx, store.GetAddressForUserParams{ID: in.AddressID, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return fail(422, "address_invalid", "address_id")
		}
		if err != nil {
			return fmt.Errorf("order: address: %w", err)
		}
		cart, err := s.price(ctx, q, userID, c, now)
		if err != nil {
			return err
		}
		if len(cart.Groups) == 0 && len(cart.Unavailable) == 0 {
			return fail(422, "cart_empty", "")
		}
		if len(cart.Unavailable) > 0 {
			e := fail(409, "cart_unavailable", "")
			for _, l := range cart.Unavailable {
				e.VariantIDs = append(e.VariantIDs, l.VariantID)
			}
			return e
		}
		if cart.DiscountError != "" {
			return fail(409, cart.DiscountError, "")
		}
		for sellerID := range in.Slots {
			if !slices.ContainsFunc(cart.Groups, func(g *Group) bool { return g.Seller.ID == sellerID }) {
				return fail(422, "slot_invalid", "slots")
			}
		}
		for _, g := range cart.Groups {
			if !codAllowed(g) {
				return fail(422, "cod_limit_exceeded", "payment_method")
			}
		}

		// 1. Stock, in variant-id order.
		lines := cart.Lines()
		slices.SortFunc(lines, func(a, b *Line) int { return cmpU(a.VariantID, b.VariantID) })
		var short []uint64
		for _, l := range lines {
			n, err := q.ReserveVariantStock(ctx, store.ReserveVariantStockParams{Qty: uint32(l.Quantity), UpdatedAt: nt(now), ID: l.VariantID})
			if err != nil {
				return fmt.Errorf("order: reserve stock: %w", err)
			}
			if n == 0 {
				short = append(short, l.VariantID)
			}
		}
		if len(short) > 0 {
			return &Error{Status: 409, Code: "out_of_stock", VariantIDs: short}
		}
		// Every line's stock is taken: the snapshot read may have seen a stale stock, so settle the totals on all lines.
		cart.settle(now)

		// 2. Discount code use.
		if cart.Discount != nil {
			n, err := q.UseDiscountCode(ctx, store.UseDiscountCodeParams{UpdatedAt: nt(now), ID: cart.Discount.ID})
			if err != nil {
				return fmt.Errorf("order: use discount: %w", err)
			}
			if n == 0 {
				return fail(409, "discount_exhausted", "")
			}
		}

		// 3. Delivery slots, in seller-id order.
		groups := slices.Clone(cart.Groups)
		slices.SortFunc(groups, func(a, b *Group) int { return cmpU(a.Seller.ID, b.Seller.ID) })
		slots := map[uint64]store.ShopDeliverySlot{}
		from := civildate.FromTime(now).AddDays(1)
		for _, g := range groups {
			slotID, ok := in.Slots[g.Seller.ID]
			if !ok {
				continue
			}
			sl, err := q.GetDeliverySlot(ctx, slotID)
			if errors.Is(err, sql.ErrNoRows) || (err == nil && (sl.SellerID != g.Seller.ID || !sl.IsActive || sl.SlotDate.Before(from))) {
				return fail(422, "slot_invalid", "slots")
			}
			if err != nil {
				return fmt.Errorf("order: slot: %w", err)
			}
			n, err := q.BookDeliverySlot(ctx, store.BookDeliverySlotParams{UpdatedAt: nt(now), ID: slotID})
			if err != nil {
				return fmt.Errorf("order: book slot: %w", err)
			}
			if n == 0 {
				return fail(409, "slot_full", "slots")
			}
			slots[g.Seller.ID] = sl
		}

		// 4. The order, shipments, items, timeline.
		orderID, code, err := createOrder(ctx, q, store.CreateOrderParams{UserID: userID,
			AddressID:     sql.NullInt64{Int64: int64(addr.ID), Valid: true}, //nolint:gosec // G115
			RecipientName: addr.RecipientName, RecipientPhone: addr.RecipientPhone, City: addr.City,
			AddressText: addressText(addr), PostalCode: addr.PostalCode, Discreet: in.Discreet,
			ItemsTotal: cart.ItemsTotal, ShippingTotal: cart.ShippingTotal(), DiscountTotal: cart.DiscountAmount,
			PayableTotal: cart.Payable(), PlacedAt: nt(now), DiscountCodeID: cart.discountID(), IdempotencyKey: ns(in.Key),
			CreatedAt: nt(now), UpdatedAt: nt(now)})
		if err != nil {
			return err
		}
		if _, err := q.CreateOrderEvent(ctx, store.CreateOrderEventParams{OrderID: orderID, Status: StatusPlaced, CreatedAt: nt(now)}); err != nil {
			return fmt.Errorf("order: event: %w", err)
		}
		categories := []sql.NullInt64{}
		for _, g := range cart.Groups {
			sp := store.CreateShipmentParams{OrderID: orderID, SellerID: g.Seller.ID, ItemsTotal: g.ItemsTotal, ShippingFee: g.ShippingFee,
				CreatedAt: nt(now), UpdatedAt: nt(now)}
			if sl, ok := slots[g.Seller.ID]; ok {
				sp.DeliverySlotID = sql.NullInt64{Int64: int64(sl.ID), Valid: true} //nolint:gosec // G115
				sp.SlotDate = civildate.NullDate{Date: sl.SlotDate, Valid: true}
				sp.SlotStartsAt, sp.SlotEndsAt = ns(sl.StartsAt), ns(sl.EndsAt)
			}
			last, err := q.CreateShipment(ctx, sp)
			if err != nil {
				return fmt.Errorf("order: shipment: %w", err)
			}
			shipmentID := uint64(last) //nolint:gosec // G115
			if _, err := q.CreateOrderEvent(ctx, store.CreateOrderEventParams{OrderID: orderID,
				ShipmentID: sql.NullInt64{Int64: last, Valid: true}, Status: ShipmentAwaiting, CreatedAt: nt(now)}); err != nil {
				return fmt.Errorf("order: event: %w", err)
			}
			for _, l := range g.Lines {
				label, ok := variantLabelSnapshot(defaultLang, variantParts(l.Size.String, l.Color.String, l.Volume.String))
				if _, err := q.CreateOrderItem(ctx, store.CreateOrderItemParams{OrderID: orderID, ShipmentID: shipmentID,
					ProductID: sql.NullInt64{Int64: int64(l.ProductID), Valid: true}, //nolint:gosec // G115
					VariantID: sql.NullInt64{Int64: int64(l.VariantID), Valid: true}, //nolint:gosec // G115
					Title:     l.Title, VariantLabel: sql.NullString{String: label, Valid: ok}, UnitPrice: l.Price, Quantity: l.Quantity,
					LineTotal: l.LineTotal, CreatedAt: nt(now), UpdatedAt: nt(now)}); err != nil {
					return fmt.Errorf("order: item: %w", err)
				}
				if err := q.AddProductPurchases(ctx, store.AddProductPurchasesParams{PurchaseCount: uint32(l.Quantity), ID: l.ProductID}); err != nil {
					return fmt.Errorf("order: purchases: %w", err)
				}
				categories = appendCategory(categories, l.CategoryID, l.CategoryParentID)
			}
		}
		if err := q.ClearCart(ctx, c.ID); err != nil {
			return fmt.Errorf("order: clear cart: %w", err)
		}
		if err := q.SetCartDiscountCode(ctx, store.SetCartDiscountCodeParams{UpdatedAt: nt(now), ID: c.ID}); err != nil {
			return fmt.Errorf("order: clear discount: %w", err)
		}
		moved, err := q.MarkChecklistOnTheWay(ctx, store.MarkChecklistOnTheWayParams{UpdatedAt: nt(now), UserID: userID, CategoryIds: categories})
		if err != nil {
			return fmt.Errorf("order: checklist: %w", err)
		}
		if len(categories) > 0 { // CB-SHOP-12 F-6: rows she never touched are created «در راه» too
			added, err := q.InsertChecklistOnTheWay(ctx, store.InsertChecklistOnTheWayParams{UserID: userID, Now: nt(now), CategoryIds: categories})
			if err != nil {
				return fmt.Errorf("order: checklist insert: %w", err)
			}
			moved += added
		}
		out = &Placed{OrderID: orderID, OnTheWay: moved, Notice: &Notice{UserID: userID, OrderID: orderID, Code: code}}
		return nil
	})
	if err != nil {
		var e *Error
		if !errors.As(err, &e) && isDuplicate(err) { // a twin request on another connection won the unique key
			if o, gerr := s.q.GetOrderByIdempotencyKey(ctx, store.GetOrderByIdempotencyKeyParams{UserID: userID, IdempotencyKey: ns(in.Key)}); gerr == nil {
				return &Placed{OrderID: o.ID, Replay: true}, nil
			}
		}
		return nil, err
	}
	return out, nil
}

func cmpU(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func appendCategory(list []sql.NullInt64, id uint64, parent sql.NullInt64) []sql.NullInt64 {
	for _, c := range []sql.NullInt64{{Int64: int64(id), Valid: true}, parent} { //nolint:gosec // G115
		if c.Valid && !slices.Contains(list, c) {
			list = append(list, c)
		}
	}
	return list
}

func addressText(a store.ShopAddress) string {
	s := a.AddressLine
	if a.Province.Valid && a.Province.String != "" {
		s = a.Province.String + "، " + s
	}
	if len(s) > 600 {
		s = s[:600]
	}
	return s
}

// createOrder inserts the order with a fresh random 8-digit code (collisions retried).
func createOrder(ctx context.Context, q *store.Queries, p store.CreateOrderParams) (uint64, string, error) {
	for range codeAttempts {
		n, err := rand.Int(rand.Reader, big.NewInt(codeSpace))
		if err != nil {
			return 0, "", fmt.Errorf("order: code: %w", err)
		}
		p.Code = fmt.Sprintf("%08d", n.Int64())
		id, err := q.CreateOrder(ctx, p)
		if err == nil {
			return uint64(id), p.Code, nil //nolint:gosec // G115
		}
		var me *mysql.MySQLError
		if !errors.As(err, &me) || me.Number != 1062 || !containsKey(me.Message, "shop_orders_code_unique") {
			return 0, "", fmt.Errorf("order: create: %w", err)
		}
	}
	return 0, "", fail(409, "busy", "")
}

func containsKey(msg, key string) bool {
	return len(msg) >= len(key) && slices.Contains(splitQuoted(msg), key)
}

// splitQuoted returns the single-quoted words of a MySQL error message ("Duplicate entry 'x' for key 'k'").
func splitQuoted(msg string) []string {
	var out []string
	start := -1
	for i, r := range msg {
		if r != '\'' {
			continue
		}
		if start < 0 {
			start = i + 1
			continue
		}
		out = append(out, msg[start:i])
		start = -1
	}
	return out
}

// ─── orders ─────────────────────────────────────────────────────────────────────────────────────────────────────────

// Detail is one order with everything the order screen shows.
type Detail struct {
	Order     store.ShopOrder
	Shipments []store.ShopOrderShipment
	Items     []store.ShopOrderItem
	Events    []store.ShopOrderEvent
	Returns   []store.ShopReturn
	Sellers   map[uint64]*store.ShopSeller
	Images    map[uint64]string
	OnTheWay  int64
	Reviews   ReviewStatus // CB-SHOP-12 F-5: her review status per order item id
}

// CanCancel: placed and no seller has acted yet.
func (d *Detail) CanCancel() bool {
	if d.Order.Status != StatusPlaced {
		return false
	}
	for _, s := range d.Shipments {
		if s.Status != ShipmentAwaiting {
			return false
		}
	}
	return true
}

// Order loads one of her orders (404 for anyone else's).
func (s *Service) Order(ctx context.Context, userID, id uint64) (*Detail, error) {
	o, err := s.q.GetOrderForUser(ctx, store.GetOrderForUserParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fail(404, "not_found", "")
	}
	if err != nil {
		return nil, fmt.Errorf("order: get: %w", err)
	}
	d := &Detail{Order: o}
	if d.Shipments, err = s.q.ListShipmentsByOrder(ctx, o.ID); err != nil {
		return nil, fmt.Errorf("order: shipments: %w", err)
	}
	if d.Items, err = s.q.ListOrderItemsByOrder(ctx, o.ID); err != nil {
		return nil, fmt.Errorf("order: items: %w", err)
	}
	if d.Events, err = s.q.ListOrderEvents(ctx, o.ID); err != nil {
		return nil, fmt.Errorf("order: events: %w", err)
	}
	if d.Returns, err = s.q.ListReturnsByOrderForUser(ctx, store.ListReturnsByOrderForUserParams{OrderID: o.ID, UserID: userID}); err != nil {
		return nil, fmt.Errorf("order: returns: %w", err)
	}
	if d.Sellers, d.Images, err = s.lookups(ctx, d.Shipments, d.Items); err != nil {
		return nil, err
	}
	if d.Reviews, err = s.reviewStatuses(ctx, userID, o.ID); err != nil {
		return nil, err
	}
	if o.Status != StatusCancelled {
		cats, err := s.q.ListOrderCategories(ctx, o.ID)
		if err != nil {
			return nil, fmt.Errorf("order: categories: %w", err)
		}
		if ids := categoryIDs(cats); len(ids) > 0 {
			if d.OnTheWay, err = s.q.CountChecklistOnTheWay(ctx, store.CountChecklistOnTheWayParams{UserID: userID, CategoryIds: ids}); err != nil {
				return nil, fmt.Errorf("order: checklist: %w", err)
			}
		}
	}
	return d, nil
}

func otherRows(rows []store.ListOtherOpenOrderCategoriesRow) []store.ListOrderCategoriesRow {
	out := make([]store.ListOrderCategoriesRow, len(rows))
	for i, r := range rows {
		out[i] = store.ListOrderCategoriesRow(r)
	}
	return out
}

func categoryIDs(rows []store.ListOrderCategoriesRow) []sql.NullInt64 {
	out := []sql.NullInt64{}
	for _, r := range rows {
		out = appendCategory(out, r.CategoryID, r.ParentID)
	}
	return out
}

func (s *Service) lookups(ctx context.Context, ships []store.ShopOrderShipment, items []store.ShopOrderItem) (map[uint64]*store.ShopSeller, map[uint64]string, error) {
	var sellerIDs, productIDs []uint64
	for _, sh := range ships {
		if !slices.Contains(sellerIDs, sh.SellerID) {
			sellerIDs = append(sellerIDs, sh.SellerID)
		}
	}
	for _, it := range items {
		if p := id64(it.ProductID); p > 0 && !slices.Contains(productIDs, p) {
			productIDs = append(productIDs, p)
		}
	}
	sellers, images := map[uint64]*store.ShopSeller{}, map[uint64]string{}
	if len(sellerIDs) > 0 {
		list, err := s.q.ListSellersByIDs(ctx, sellerIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("order: sellers: %w", err)
		}
		for i := range list {
			sellers[list[i].ID] = &list[i]
		}
	}
	if len(productIDs) > 0 {
		list, err := s.q.ListProductImages(ctx, productIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("order: images: %w", err)
		}
		for _, p := range list {
			images[p.ID] = coverPath(p.ImagePath, p.Images)
		}
	}
	return sellers, images, nil
}

// Summary is an order card of «سفارش‌ها».
type Summary struct {
	Order     store.ShopOrder
	Shipments []store.ShopOrderShipment
	Items     []store.ShopOrderItem
}

// Orders lists her newest orders with their shipments and items.
func (s *Service) Orders(ctx context.Context, userID uint64) ([]Summary, map[uint64]*store.ShopSeller, map[uint64]string, error) {
	orders, err := s.q.ListOrdersByUser(ctx, store.ListOrdersByUserParams{UserID: userID, Limit: OrdersListed})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("order: list: %w", err)
	}
	if len(orders) == 0 {
		return []Summary{}, nil, nil, nil
	}
	ships, err := s.q.ListShipmentsByUser(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("order: shipments: %w", err)
	}
	items, err := s.q.ListOrderItemsByUser(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("order: items: %w", err)
	}
	out := make([]Summary, len(orders))
	idx := map[uint64]int{}
	for i, o := range orders {
		out[i].Order, idx[o.ID] = o, i
	}
	var keptShips []store.ShopOrderShipment
	var keptItems []store.ShopOrderItem
	for _, sh := range ships {
		if i, ok := idx[sh.OrderID]; ok {
			out[i].Shipments = append(out[i].Shipments, sh)
			keptShips = append(keptShips, sh)
		}
	}
	for _, it := range items {
		if i, ok := idx[it.OrderID]; ok {
			out[i].Items = append(out[i].Items, it)
			keptItems = append(keptItems, it)
		}
	}
	sellers, images, err := s.lookups(ctx, keptShips, keptItems)
	return out, sellers, images, err
}

// Cancel cancels one of her orders while every shipment still awaits the seller: stock, slots and the discount use
// are given back, checklist rows set «در راه» by this order go back to «todo».
func (s *Service) Cancel(ctx context.Context, userID, id uint64, now time.Time) error {
	return s.cancel(ctx, userID, id, now, (*Detail).CanCancel)
}

// CancelByAdmin cancels an order from the admin queue (CB-SHOP-05): allowed while it is placed and no shipment has
// left the seller (awaiting or confirmed); stock, slots, the code use and checklist rows are given back like Cancel.
func (s *Service) CancelByAdmin(ctx context.Context, id uint64, now time.Time) error {
	o, err := s.q.AdminGetOrder(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return fail(404, "not_found", "")
	}
	if err != nil {
		return fmt.Errorf("order: get: %w", err)
	}
	return s.cancel(ctx, o.UserID, id, now, (*Detail).CanAdminCancel)
}

// CanAdminCancel: placed and no shipment shipped or delivered yet (CB-SHOP-05).
func (d *Detail) CanAdminCancel() bool {
	if d.Order.Status != StatusPlaced {
		return false
	}
	for _, s := range d.Shipments {
		if s.Status != ShipmentAwaiting && s.Status != ShipmentConfirmed {
			return false
		}
	}
	return true
}

func (s *Service) cancel(ctx context.Context, userID, id uint64, now time.Time, allowed func(*Detail) bool) error {
	return s.inTx(ctx, func(q *store.Queries) error {
		o, err := q.LockOrderForUser(ctx, store.LockOrderForUserParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return fail(404, "not_found", "")
		}
		if err != nil {
			return fmt.Errorf("order: lock: %w", err)
		}
		ships, err := q.ListShipmentsByOrder(ctx, o.ID)
		if err != nil {
			return fmt.Errorf("order: shipments: %w", err)
		}
		d := &Detail{Order: o, Shipments: ships}
		if !allowed(d) {
			return fail(422, "not_cancellable", "")
		}
		if n, err := q.CancelOrderForUser(ctx, store.CancelOrderForUserParams{CancelledAt: nt(now), UpdatedAt: nt(now), ID: o.ID, UserID: userID}); err != nil {
			return fmt.Errorf("order: cancel: %w", err)
		} else if n == 0 {
			return fail(422, "not_cancellable", "")
		}
		if err := q.CancelOrderShipments(ctx, store.CancelOrderShipmentsParams{UpdatedAt: nt(now), OrderID: o.ID}); err != nil {
			return fmt.Errorf("order: cancel shipments: %w", err)
		}
		items, err := q.ListOrderItemsByOrder(ctx, o.ID)
		if err != nil {
			return fmt.Errorf("order: items: %w", err)
		}
		slices.SortFunc(items, func(a, b store.ShopOrderItem) int { return cmpU(id64(a.VariantID), id64(b.VariantID)) })
		for _, it := range items {
			if it.VariantID.Valid {
				if err := q.RestockVariant(ctx, store.RestockVariantParams{Stock: uint32(it.Quantity), UpdatedAt: nt(now), ID: id64(it.VariantID)}); err != nil {
					return fmt.Errorf("order: restock: %w", err)
				}
			}
			if it.ProductID.Valid {
				if err := q.SubtractProductPurchases(ctx, store.SubtractProductPurchasesParams{Qty: uint32(it.Quantity), ID: id64(it.ProductID)}); err != nil {
					return fmt.Errorf("order: purchases: %w", err)
				}
			}
		}
		if o.DiscountCodeID.Valid {
			if err := q.ReleaseDiscountCode(ctx, store.ReleaseDiscountCodeParams{UpdatedAt: nt(now), ID: id64(o.DiscountCodeID)}); err != nil {
				return fmt.Errorf("order: release discount: %w", err)
			}
		}
		for _, sh := range ships {
			if sh.DeliverySlotID.Valid {
				if err := q.ReleaseDeliverySlot(ctx, store.ReleaseDeliverySlotParams{UpdatedAt: nt(now), ID: id64(sh.DeliverySlotID)}); err != nil {
					return fmt.Errorf("order: release slot: %w", err)
				}
			}
		}
		if _, err := q.CreateOrderEvent(ctx, store.CreateOrderEventParams{OrderID: o.ID, Status: StatusCancelled, CreatedAt: nt(now)}); err != nil {
			return fmt.Errorf("order: event: %w", err)
		}
		cats, err := q.ListOrderCategories(ctx, o.ID)
		if err != nil {
			return fmt.Errorf("order: categories: %w", err)
		}
		// Rows «در راه» carry no order id, so only categories none of her other placed orders still carries go back
		// to «باید بخرم» (a row she marked «در راه» by hand in such a category is still reverted — not traceable).
		others, err := q.ListOtherOpenOrderCategories(ctx, store.ListOtherOpenOrderCategoriesParams{UserID: userID, ID: o.ID})
		if err != nil {
			return fmt.Errorf("order: open categories: %w", err)
		}
		keep := categoryIDs(otherRows(others))
		ids := slices.DeleteFunc(categoryIDs(cats), func(c sql.NullInt64) bool { return slices.Contains(keep, c) })
		if len(ids) > 0 {
			if _, err := q.RevertChecklistOnTheWay(ctx, store.RevertChecklistOnTheWayParams{UpdatedAt: nt(now), UserID: userID, CategoryIds: ids}); err != nil {
				return fmt.Errorf("order: checklist: %w", err)
			}
		}
		return nil
	})
}

// MarkShipmentDelivered records a delivery (CB-SHOP-05's admin calls it): shipment delivered + timeline, the order
// becomes delivered once every live shipment is, and the user's checklist rows «در راه» for these categories become
// «دارم». Idempotent: an already delivered or cancelled shipment is left alone.
func (s *Service) MarkShipmentDelivered(ctx context.Context, shipmentID uint64, now time.Time) error {
	return s.inTx(ctx, func(q *store.Queries) error {
		sh, err := q.GetShipment(ctx, shipmentID)
		if errors.Is(err, sql.ErrNoRows) {
			return fail(404, "not_found", "")
		}
		if err != nil {
			return fmt.Errorf("order: shipment: %w", err)
		}
		o, err := q.LockOrder(ctx, sh.OrderID)
		if err != nil {
			return fmt.Errorf("order: lock: %w", err)
		}
		n, err := q.MarkShipmentDelivered(ctx, store.MarkShipmentDeliveredParams{DeliveredAt: nt(now), UpdatedAt: nt(now), ID: shipmentID})
		if err != nil {
			return fmt.Errorf("order: deliver: %w", err)
		}
		if n == 0 {
			return nil
		}
		if _, err := q.CreateOrderEvent(ctx, store.CreateOrderEventParams{OrderID: o.ID,
			ShipmentID: sql.NullInt64{Int64: int64(shipmentID), Valid: true}, Status: StatusDelivered, CreatedAt: nt(now)}); err != nil { //nolint:gosec // G115
			return fmt.Errorf("order: event: %w", err)
		}
		left, err := q.CountUndeliveredShipments(ctx, o.ID)
		if err != nil {
			return fmt.Errorf("order: shipments: %w", err)
		}
		if left == 0 && o.Status != StatusCancelled {
			if err := q.SetOrderStatus(ctx, store.SetOrderStatusParams{Status: StatusDelivered, UpdatedAt: nt(now), ID: o.ID}); err != nil {
				return fmt.Errorf("order: status: %w", err)
			}
			if _, err := q.CreateOrderEvent(ctx, store.CreateOrderEventParams{OrderID: o.ID, Status: StatusDelivered, CreatedAt: nt(now)}); err != nil {
				return fmt.Errorf("order: event: %w", err)
			}
		}
		rows, err := q.ListShipmentCategories(ctx, shipmentID)
		if err != nil {
			return fmt.Errorf("order: categories: %w", err)
		}
		ids := []sql.NullInt64{}
		for _, r := range rows {
			ids = appendCategory(ids, r.CategoryID, r.ParentID)
		}
		if len(ids) > 0 {
			if _, err := q.MarkChecklistHave(ctx, store.MarkChecklistHaveParams{UpdatedAt: nt(now), UserID: o.UserID, CategoryIds: ids}); err != nil {
				return fmt.Errorf("order: checklist: %w", err)
			}
		}
		return nil
	})
}

// ─── returns ────────────────────────────────────────────────────────────────────────────────────────────────────────

// ReturnInput is a validated return request.
type ReturnInput struct {
	OrderItemID uint64
	Quantity    int
	Reason      string
	Note        string
}

// ReturnUntil is the last moment a delivered shipment's items can be returned (seller's return_days, default 7).
func ReturnUntil(deliveredAt time.Time, seller *store.ShopSeller) time.Time {
	days := 7
	if seller != nil {
		days = int(seller.ReturnDays)
	}
	return deliveredAt.AddDate(0, 0, days)
}

// RequestReturn files a return for an item of one of her delivered shipments within the seller's window.
func (s *Service) RequestReturn(ctx context.Context, userID, orderID uint64, in ReturnInput, now time.Time) (store.ShopReturn, error) {
	var out store.ShopReturn
	err := s.inTx(ctx, func(q *store.Queries) error {
		if _, err := q.LockOrderForUser(ctx, store.LockOrderForUserParams{ID: orderID, UserID: userID}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fail(404, "not_found", "")
			}
			return fmt.Errorf("order: lock: %w", err)
		}
		it, err := q.GetOrderItemForReturn(ctx, store.GetOrderItemForReturnParams{ItemID: in.OrderItemID, OrderID: orderID, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return fail(422, "not_returnable", "order_item_id")
		}
		if err != nil {
			return fmt.Errorf("order: item: %w", err)
		}
		if it.ShipmentStatus != StatusDelivered || !it.DeliveredAt.Valid {
			return fail(422, "not_returnable", "order_item_id")
		}
		sellers, err := q.ListSellersByIDs(ctx, []uint64{it.SellerID})
		if err != nil {
			return fmt.Errorf("order: seller: %w", err)
		}
		var seller *store.ShopSeller
		if len(sellers) > 0 {
			seller = &sellers[0]
		}
		if now.After(ReturnUntil(it.DeliveredAt.Time, seller)) {
			return fail(422, "return_window_closed", "order_item_id")
		}
		open, err := q.SumOpenReturnQuantity(ctx, it.ID)
		if err != nil {
			return fmt.Errorf("order: returns: %w", err)
		}
		if open+int64(in.Quantity) > int64(it.Quantity) {
			return fail(422, "return_quantity", "quantity")
		}
		last, err := q.CreateReturn(ctx, store.CreateReturnParams{OrderID: orderID, OrderItemID: it.ID, UserID: userID,
			Quantity: uint16(in.Quantity), Reason: in.Reason, Note: ns(in.Note), CreatedAt: nt(now), UpdatedAt: nt(now)}) //nolint:gosec // G115
		if err != nil {
			return fmt.Errorf("order: create return: %w", err)
		}
		if _, err := q.CreateOrderEvent(ctx, store.CreateOrderEventParams{OrderID: orderID, Status: EventReturnRequested, CreatedAt: nt(now)}); err != nil {
			return fmt.Errorf("order: event: %w", err)
		}
		out, err = q.GetReturnForUser(ctx, store.GetReturnForUserParams{ID: uint64(last), UserID: userID}) //nolint:gosec // G115
		if err != nil {
			return fmt.Errorf("order: return: %w", err)
		}
		return nil
	})
	return out, err
}

// Returns lists her return requests, newest first.
func (s *Service) Returns(ctx context.Context, userID uint64) ([]store.ShopReturn, error) {
	return s.q.ListReturnsByUser(ctx, userID)
}

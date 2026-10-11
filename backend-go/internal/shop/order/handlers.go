package order

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/shop/catalog"
	"github.com/ritme/backend-go/internal/shop/store"
)

// PaymentCOD is the only payment method of the MVP (DECISIONS #13): any other value is a 422.
const PaymentCOD = "cod"

// Handlers serve /api/v1/shop/{cart,addresses,checkout,orders,returns} (auth:api, owner-scoped; teen → 403).
type Handlers struct {
	svc      *Service
	policy   catalog.CommercialPolicy
	notifier Notifier
	appURL   string
	clock    clock.Clock
	logger   *slog.Logger
}

// NewHandlers wires the handlers; notifier and logger may be nil.
func NewHandlers(svc *Service, policy catalog.CommercialPolicy, notifier Notifier, appURL string, clk clock.Clock, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, policy: policy, notifier: notifier, appURL: appURL, clock: clk, logger: logger}
}

// gate resolves the user and refuses a teen-mode account (403 teen_commercial_blocked, as the catalog does).
func (h *Handlers) gate(c fiber.Ctx) (uint64, error) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	if h.policy != nil {
		allowed, err := h.policy.AllowsCommercial(c, uid)
		if err != nil {
			return 0, fmt.Errorf("order: policy: %w", err)
		}
		if !allowed {
			return 0, httpx.Fail(fiber.StatusForbidden, catalog.T("errors.teen_commercial_blocked", i18n.Locale(c)),
				"error_code", catalog.ErrorCodeTeenCommercialBlocked)
		}
	}
	return uid, nil
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func (h *Handlers) locale(c fiber.Ctx) catalog.Locale {
	return catalog.Locale{Code: i18n.ResolveLocale(c, ""), Default: i18n.LanguagesOf(c).DefaultCode()}
}

// render maps a service *Error to {success:false, message, error_code[, errors][, variant_ids]}.
func render(c fiber.Ctx, err error) error {
	var e *Error
	if !errors.As(err, &e) {
		return err
	}
	locale := i18n.Locale(c)
	text := T("errors."+e.Code, locale)
	extras := []any{"error_code", e.Code}
	if e.Field != "" {
		extras = append(extras, "errors", jsonx.Obj(e.Field, []string{text}))
	}
	if len(e.VariantIDs) > 0 {
		extras = append(extras, "variant_ids", e.VariantIDs)
	}
	return httpx.Fail(e.Status, text, extras...)
}

func notFound(c fiber.Ctx) error { return render(c, fail(fiber.StatusNotFound, "not_found", "")) }

func pathID(c fiber.Ctx, name string) (uint64, bool) {
	s := c.Params(name)
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil && n > 0 && len(s) < 20
}

// validate runs rules over the allow-listed keys of the body; a failure is the framework-style 422 plus success:false.
func validate(c fiber.Ctx, now time.Time, rules validation.Rules, keys ...string) (phpval.Map, error) {
	return validateInput(c, validation.Input(c), now, rules, keys...)
}

func validateInput(c fiber.Ctx, in phpval.Map, now time.Time, rules validation.Rules, keys ...string) (phpval.Map, error) {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := in.Get(k); ok {
			data.Set(k, v)
		}
	}
	locale := i18n.Locale(c)
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributeNames(locale, keys...)...))
	if v.Fails() {
		return nil, httpx.Fail(fiber.StatusUnprocessableEntity, T("errors.validation_failed", locale), "errors", v.ErrorBag())
	}
	return data, nil
}

func str(data any, key string) string {
	v, _ := phpval.Get(data, key)
	if v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

func num(data any, key string) uint64 {
	n, _ := strconv.ParseUint(str(data, key), 10, 64)
	return n
}

// ─── JSON ───────────────────────────────────────────────────────────────────────────────────────────────────────────

func (h *Handlers) url(path string) any {
	if path == "" {
		return nil
	}
	return content.PublicURL(h.appURL, path)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func iso(t time.Time) string { return t.In(civildate.Tehran).Format("2006-01-02T15:04:05-07:00") }

func isoNull(valid bool, t time.Time) any {
	if !valid {
		return nil
	}
	return iso(t)
}

func hm(s string) string {
	if len(s) >= 5 {
		return s[:5]
	}
	return s
}

func (h *Handlers) sellerJSON(l catalog.Locale, s *store.ShopSeller) *jsonx.OrderedMap {
	return jsonx.Obj("id", s.ID, "slug", s.Slug, "name", l.Pick(s.Name), "logo_url", h.url(s.LogoPath.String),
		"is_verified", s.IsVerified, "ship_min_days", s.ShipMinDays, "ship_max_days", s.ShipMaxDays,
		"discreet_packaging", s.DiscreetPackaging, "shipping_fee", s.ShippingFee, "free_shipping_over", nullInt(s.FreeShippingOver),
		"cod_max_total", nullInt(s.CodMaxTotal), "return_days", s.ReturnDays)
}

func (h *Handlers) lineJSON(l catalog.Locale, x *Line) *jsonx.OrderedMap {
	return jsonx.Obj("variant_id", x.VariantID, "product_id", x.ProductID, "title", l.Pick(x.Title),
		"image_url", h.url(x.ImagePath.String),
		"variant_label", nullable(variantLabel(l, variantParts(x.Size.String, x.Color.String, x.Volume.String))),
		"unit_price", x.Price, "compare_at_price", nullInt(x.CompareAtPrice), "quantity", x.Quantity, "line_total", x.LineTotal,
		"stock", x.Stock, "status", x.Status, "discreet_packaging", x.DiscreetPackaging)
}

func (h *Handlers) cartJSON(l catalog.Locale, cart *Cart) *jsonx.OrderedMap {
	groups := make([]*jsonx.OrderedMap, 0, len(cart.Groups))
	for _, g := range cart.Groups {
		items := make([]*jsonx.OrderedMap, 0, len(g.Lines))
		for _, x := range g.Lines {
			items = append(items, h.lineJSON(l, x))
		}
		groups = append(groups, jsonx.Obj("seller", h.sellerJSON(l, g.Seller), "items_count", g.ItemCount,
			"items_total", g.ItemsTotal, "shipping_fee", g.ShippingFee, "free_shipping_remaining", freeShippingRemaining(g),
			"items", items))
	}
	unavailable := make([]*jsonx.OrderedMap, 0, len(cart.Unavailable))
	for _, x := range cart.Unavailable {
		unavailable = append(unavailable, h.lineJSON(l, x))
	}
	var discount any
	if cart.Discount != nil {
		discount = jsonx.Obj("code", cart.Discount.Code, "amount", cart.DiscountAmount, "error_code", nullable(cart.DiscountError))
	}
	return jsonx.Obj("items_count", cart.ItemCount, "sellers_count", len(cart.Groups), "groups", groups,
		"unavailable_items", unavailable, "discount", discount, "items_total", cart.ItemsTotal,
		"discount_total", cart.DiscountAmount, "payable_total", cart.ItemsTotal-cart.DiscountAmount)
}

func addressJSON(a store.ShopAddress) *jsonx.OrderedMap {
	return jsonx.Obj("id", a.ID, "label", nullable(a.Label.String), "recipient_name", a.RecipientName,
		"recipient_phone", a.RecipientPhone, "province", nullable(a.Province.String), "city", a.City,
		"address_line", a.AddressLine, "postal_code", nullable(a.PostalCode.String), "is_default", a.IsDefault)
}

func slotJSON(id uint64, date civildate.Date, starts, ends string) *jsonx.OrderedMap {
	return jsonx.Obj("id", id, "date", date.String(), "starts_at", hm(starts), "ends_at", hm(ends))
}

func eventJSON(e store.ShopOrderEvent) *jsonx.OrderedMap {
	return jsonx.Obj("status", e.Status, "at", isoNull(e.CreatedAt.Valid, e.CreatedAt.Time))
}

func returnJSON(r store.ShopReturn) *jsonx.OrderedMap {
	return jsonx.Obj("id", r.ID, "order_id", r.OrderID, "order_item_id", r.OrderItemID, "quantity", r.Quantity,
		"reason", r.Reason, "note", nullable(r.Note.String), "status", r.Status,
		"created_at", isoNull(r.CreatedAt.Valid, r.CreatedAt.Time))
}

func (h *Handlers) itemJSON(l catalog.Locale, it store.ShopOrderItem, images map[uint64]string) *jsonx.OrderedMap {
	return jsonx.Obj("id", it.ID, "product_id", nullInt(it.ProductID), "variant_id", nullInt(it.VariantID),
		"title", l.Pick(it.Title), "variant_label", nullable(l.Pick(it.VariantLabel.String)),
		"image_url", h.url(images[id64(it.ProductID)]), "unit_price", it.UnitPrice, "quantity", it.Quantity,
		"line_total", it.LineTotal)
}

func (h *Handlers) shipmentSellerJSON(l catalog.Locale, sellers map[uint64]*store.ShopSeller, id uint64) any {
	s, ok := sellers[id]
	if !ok {
		return nil
	}
	return jsonx.Obj("id", s.ID, "slug", s.Slug, "name", l.Pick(s.Name), "logo_url", h.url(s.LogoPath.String))
}

func shipmentSlot(s store.ShopOrderShipment) any {
	if !s.SlotDate.Valid {
		return nil
	}
	return jsonx.Obj("date", s.SlotDate.Date.String(), "starts_at", hm(s.SlotStartsAt.String), "ends_at", hm(s.SlotEndsAt.String))
}

func (h *Handlers) orderJSON(l catalog.Locale, d *Detail, now time.Time) *jsonx.OrderedMap {
	o := d.Order
	shipments := make([]*jsonx.OrderedMap, 0, len(d.Shipments))
	for _, s := range d.Shipments {
		var until any
		returnable := false
		if s.Status == StatusDelivered && s.DeliveredAt.Valid {
			u := ReturnUntil(s.DeliveredAt.Time, d.Sellers[s.SellerID])
			until, returnable = iso(u), !now.After(u)
		}
		items := []*jsonx.OrderedMap{}
		for _, it := range d.Items {
			if it.ShipmentID == s.ID {
				items = append(items, h.itemJSON(l, it, d.Images).Set("returnable", returnable))
			}
		}
		timeline := []*jsonx.OrderedMap{}
		for _, e := range d.Events {
			if id64(e.ShipmentID) == s.ID {
				timeline = append(timeline, eventJSON(e))
			}
		}
		shipments = append(shipments, jsonx.Obj("id", s.ID, "seller", h.shipmentSellerJSON(l, d.Sellers, s.SellerID),
			"status", s.Status, "slot", shipmentSlot(s), "items_total", s.ItemsTotal, "shipping_fee", s.ShippingFee,
			"tracking_code", nullable(s.TrackingCode.String), "delivered_at", isoNull(s.DeliveredAt.Valid, s.DeliveredAt.Time),
			"return_until", until, "items", items, "timeline", timeline))
	}
	timeline := []*jsonx.OrderedMap{}
	for _, e := range d.Events {
		if !e.ShipmentID.Valid {
			timeline = append(timeline, eventJSON(e))
		}
	}
	returns := make([]*jsonx.OrderedMap, 0, len(d.Returns))
	for _, r := range d.Returns {
		returns = append(returns, returnJSON(r))
	}
	return jsonx.Obj("id", o.ID, "code", o.Code, "status", o.Status, "payment_method", o.PaymentMethod,
		"payment_status", o.PaymentStatus, "discreet", o.Discreet, "can_cancel", d.CanCancel(),
		"recipient", jsonx.Obj("name", o.RecipientName, "phone", o.RecipientPhone, "city", o.City, "address", o.AddressText,
			"postal_code", nullable(o.PostalCode.String)),
		"items_total", o.ItemsTotal, "shipping_total", o.ShippingTotal, "discount_total", o.DiscountTotal,
		"payable_total", o.PayableTotal, "placed_at", isoNull(o.PlacedAt.Valid, o.PlacedAt.Time),
		"cancelled_at", isoNull(o.CancelledAt.Valid, o.CancelledAt.Time), "shipments", shipments, "timeline", timeline,
		"returns", returns, "checklist_on_the_way", d.OnTheWay)
}

// ─── cart ───────────────────────────────────────────────────────────────────────────────────────────────────────────

func (h *Handlers) cartReply(c fiber.Ctx, uid uint64) error {
	cart, err := h.svc.Cart(c, uid, h.now(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("cart", h.cartJSON(h.locale(c), cart)))
}

// Cart is GET /shop/cart: lines grouped per seller (one shipment each), unavailable lines apart, the applied discount.
func (h *Handlers) Cart(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	return h.cartReply(c, uid)
}

var maxQty = "max:" + strconv.Itoa(MaxLineQuantity)

// AddItem is POST /shop/cart/items {variant_id, quantity=1}: adds to the line (≤ stock, ≤ MaxLineQuantity).
func (h *Handlers) AddItem(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	data, err := validate(c, now, validation.Rules{
		validation.F("variant_id", "required", "integer", "min:1"),
		validation.F("quantity", "nullable", "integer", "min:1", maxQty),
	}, "variant_id", "quantity")
	if err != nil {
		return err
	}
	qty := max(int(num(data, "quantity")), 1) //nolint:gosec // G115: validated ≤ MaxLineQuantity
	if err := h.svc.SetItem(c, uid, num(data, "variant_id"), qty, true, now); err != nil {
		return render(c, err)
	}
	return h.cartReply(c, uid)
}

// UpdateItem is PUT /shop/cart/items/:variant_id {quantity}: sets the line's quantity (404 when not in her cart).
func (h *Handlers) UpdateItem(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	vid, ok := pathID(c, "variant_id")
	if !ok {
		return notFound(c)
	}
	now := h.now(c)
	data, err := validate(c, now, validation.Rules{validation.F("quantity", "required", "integer", "min:1", maxQty)}, "quantity")
	if err != nil {
		return err
	}
	if err := h.svc.SetItem(c, uid, vid, int(num(data, "quantity")), false, now); err != nil { //nolint:gosec // G115: validated
		return render(c, err)
	}
	return h.cartReply(c, uid)
}

// RemoveItem is DELETE /shop/cart/items/:variant_id.
func (h *Handlers) RemoveItem(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	vid, ok := pathID(c, "variant_id")
	if !ok {
		return notFound(c)
	}
	if err := h.svc.RemoveItem(c, uid, vid); err != nil {
		return render(c, err)
	}
	return h.cartReply(c, uid)
}

// ApplyDiscount is PUT /shop/cart/discount {code}: «کد تخفیف داری؟». 422 with the reason when it does not apply.
func (h *Handlers) ApplyDiscount(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	data, err := validate(c, now, validation.Rules{validation.F("code", "required", "string", "max:32")}, "code")
	if err != nil {
		return err
	}
	if err := h.svc.ApplyDiscount(c, uid, str(data, "code"), now); err != nil {
		return render(c, err)
	}
	return h.cartReply(c, uid)
}

// RemoveDiscount is DELETE /shop/cart/discount.
func (h *Handlers) RemoveDiscount(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	if err := h.svc.RemoveDiscount(c, uid, h.now(c)); err != nil {
		return err
	}
	return h.cartReply(c, uid)
}

// ─── addresses ──────────────────────────────────────────────────────────────────────────────────────────────────────

// Persian / Arabic digits → ASCII and separators out; phone and postal code are validated on the folded value.
var digitFold = strings.NewReplacer("۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	" ", "", "-", "")

var addressKeys = []string{"label", "recipient_name", "recipient_phone", "province", "city", "address_line", "postal_code", "is_default"}

func addressRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("label", "nullable", "string", "max:40"),
		F("recipient_name", "required", "string", "max:120"),
		F("recipient_phone", "required", "string", "regex:/^09[0-9]{9}$/"),
		F("province", "nullable", "string", "max:60"),
		F("city", "required", "string", "max:60"),
		F("address_line", "required", "string", "max:500"),
		F("postal_code", "nullable", "string", "regex:/^[0-9]{10}$/"),
		F("is_default", "nullable", "boolean"),
	}
}

func (h *Handlers) addressInput(c fiber.Ctx, now time.Time) (AddressInput, error) {
	in := validation.Input(c)
	for _, k := range []string{"recipient_phone", "postal_code"} {
		if v, ok := in.Get(k); ok {
			if s, isStr := v.(string); isStr {
				s = digitFold.Replace(s)
				if k == "recipient_phone" && strings.HasPrefix(s, "+98") {
					s = "0" + s[3:]
				}
				in.Set(k, s)
			}
		}
	}
	data, err := validateInput(c, in, now, addressRules(), addressKeys...)
	if err != nil {
		return AddressInput{}, err
	}
	def, _ := phpval.Get(data, "is_default")
	return AddressInput{Label: str(data, "label"), RecipientName: str(data, "recipient_name"),
		RecipientPhone: str(data, "recipient_phone"), Province: str(data, "province"), City: str(data, "city"),
		AddressLine: str(data, "address_line"), PostalCode: str(data, "postal_code"), IsDefault: phpval.Truthy(def)}, nil
}

// Addresses is GET /shop/addresses (her own, default first).
func (h *Handlers) Addresses(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	list, err := h.svc.Addresses(c, uid)
	if err != nil {
		return fmt.Errorf("order: addresses: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(list))
	for _, a := range list {
		out = append(out, addressJSON(a))
	}
	return httpx.OK(c, jsonx.Obj("addresses", out))
}

// StoreAddress is POST /shop/addresses → 201.
func (h *Handlers) StoreAddress(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	in, err := h.addressInput(c, now)
	if err != nil {
		return err
	}
	a, err := h.svc.SaveAddress(c, uid, 0, in, now)
	if err != nil {
		return render(c, err)
	}
	return httpx.Created(c, jsonx.Obj("address", addressJSON(a)), T("messages.address_saved", i18n.Locale(c)))
}

// UpdateAddress is PUT /shop/addresses/:id (404 for anyone else's).
func (h *Handlers) UpdateAddress(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	id, ok := pathID(c, "id")
	if !ok {
		return notFound(c)
	}
	now := h.now(c)
	in, err := h.addressInput(c, now)
	if err != nil {
		return err
	}
	a, err := h.svc.SaveAddress(c, uid, id, in, now)
	if err != nil {
		return render(c, err)
	}
	return httpx.OK(c, jsonx.Obj("address", addressJSON(a)), T("messages.address_saved", i18n.Locale(c)))
}

// DeleteAddress is DELETE /shop/addresses/:id (404 for anyone else's). Orders keep their address snapshot.
func (h *Handlers) DeleteAddress(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	id, ok := pathID(c, "id")
	if !ok {
		return notFound(c)
	}
	if err := h.svc.DeleteAddress(c, uid, id); err != nil {
		return render(c, err)
	}
	return httpx.OK(c, nil, T("messages.address_deleted", i18n.Locale(c)))
}

// ─── checkout ───────────────────────────────────────────────────────────────────────────────────────────────────────

// Checkout is GET /shop/checkout: the review step (addresses, per-seller shipment with fee and open slots, plain
// packaging, COD — the only payment method — with its availability, totals, blockers).
func (h *Handlers) Checkout(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	co, err := h.svc.Checkout(c, uid, h.now(c))
	if err != nil {
		return err
	}
	l := h.locale(c)
	addrs := make([]*jsonx.OrderedMap, 0, len(co.Addresses))
	var def any
	for _, a := range co.Addresses {
		addrs = append(addrs, addressJSON(a))
		if def == nil && a.IsDefault {
			def = a.ID
		}
	}
	if def == nil && len(co.Addresses) > 0 {
		def = co.Addresses[0].ID
	}
	discreet := false
	shipments := make([]*jsonx.OrderedMap, 0, len(co.Cart.Groups))
	for _, g := range co.Cart.Groups {
		slots := make([]*jsonx.OrderedMap, 0, len(co.Slots[g.Seller.ID]))
		for _, s := range co.Slots[g.Seller.ID] {
			slots = append(slots, slotJSON(s.ID, s.SlotDate, s.StartsAt, s.EndsAt))
		}
		discreet = discreet || g.Seller.DiscreetPackaging
		shipments = append(shipments, jsonx.Obj("seller", h.sellerJSON(l, g.Seller), "items_count", g.ItemCount,
			"items_total", g.ItemsTotal, "shipping_fee", g.ShippingFee, "cod_available", codAllowed(g), "slots", slots))
	}
	cart := co.Cart
	blockers := co.Blockers
	if blockers == nil {
		blockers = []string{}
	}
	return httpx.OK(c, jsonx.Obj("cart", h.cartJSON(l, cart), "addresses", addrs, "default_address_id", def,
		"shipments", shipments, "discreet_packaging_available", discreet,
		"payment_methods", []*jsonx.OrderedMap{jsonx.Obj("method", PaymentCOD, "available", co.COD)},
		"items_total", cart.ItemsTotal, "discount_total", cart.DiscountAmount, "shipping_total", cart.ShippingTotal(),
		"payable_total", cart.Payable(), "can_place", len(blockers) == 0, "blockers", blockers))
}

// ─── orders ─────────────────────────────────────────────────────────────────────────────────────────────────────────

// Orders is GET /shop/orders: her newest orders as cards.
func (h *Handlers) Orders(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	list, sellers, images, err := h.svc.Orders(c, uid)
	if err != nil {
		return err
	}
	l := h.locale(c)
	out := make([]*jsonx.OrderedMap, 0, len(list))
	for _, s := range list {
		ships := make([]*jsonx.OrderedMap, 0, len(s.Shipments))
		for _, sh := range s.Shipments {
			ships = append(ships, jsonx.Obj("id", sh.ID, "seller", h.shipmentSellerJSON(l, sellers, sh.SellerID),
				"status", sh.Status, "slot", shipmentSlot(sh)))
		}
		items := make([]*jsonx.OrderedMap, 0, len(s.Items))
		count := 0
		for _, it := range s.Items {
			items = append(items, h.itemJSON(l, it, images))
			count += int(it.Quantity)
		}
		o := s.Order
		out = append(out, jsonx.Obj("id", o.ID, "code", o.Code, "status", o.Status, "payment_method", o.PaymentMethod,
			"payable_total", o.PayableTotal, "items_count", count, "placed_at", isoNull(o.PlacedAt.Valid, o.PlacedAt.Time),
			"shipments", ships, "items", items))
	}
	return httpx.OK(c, jsonx.Obj("orders", out))
}

// Order is GET /shop/orders/:id (404 for anyone else's).
func (h *Handlers) Order(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	id, ok := pathID(c, "id")
	if !ok {
		return notFound(c)
	}
	d, err := h.svc.Order(c, uid, id)
	if err != nil {
		return render(c, err)
	}
	return httpx.OK(c, jsonx.Obj("order", h.orderJSON(h.locale(c), d, h.now(c))))
}

// PlaceOrder is POST /shop/orders {idempotency_key, address_id, payment_method: "cod", discreet, slots[]} → 201 with
// the order; the same idempotency_key again answers 200 with that first order (nothing new is placed).
func (h *Handlers) PlaceOrder(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	F := validation.F
	data, err := validate(c, now, validation.Rules{
		F("idempotency_key", "required", "string", "min:8", "max:64", "regex:/^[A-Za-z0-9_-]+$/"),
		F("address_id", "required", "integer", "min:1"),
		F("payment_method", "required", validation.In(PaymentCOD)),
		F("discreet", "nullable", "boolean"),
		F("slots", "nullable", "array", "max:20"),
		F("slots.*.seller_id", "required", "integer", "min:1"),
		F("slots.*.slot_id", "required", "integer", "min:1"),
	}, "idempotency_key", "address_id", "payment_method", "discreet", "slots")
	if err != nil {
		return err
	}
	in := PlaceInput{Key: str(data, "idempotency_key"), AddressID: num(data, "address_id"), Slots: map[uint64]uint64{}}
	disc, _ := phpval.Get(data, "discreet")
	in.Discreet = phpval.Truthy(disc)
	slots, _ := phpval.Get(data, "slots")
	_, entries := phpval.Entries(slots)
	for _, e := range entries {
		in.Slots[num(e, "seller_id")] = num(e, "slot_id")
	}
	locale := i18n.Locale(c)
	placed, err := h.svc.Place(c, uid, in, i18n.LanguagesOf(c).DefaultCode(), now)
	if err != nil {
		return render(c, err)
	}
	if placed.Notice != nil && h.notifier != nil {
		n := *placed.Notice
		n.Text = T("sms.order_placed", locale, "code", n.Code)
		if err := h.notifier.OrderPlaced(c, n); err != nil && h.logger != nil {
			h.logger.WarnContext(c, "shop: order sms failed", slog.Uint64("order_id", n.OrderID)) // the order stands
		}
	}
	d, err := h.svc.Order(c, uid, placed.OrderID)
	if err != nil {
		return render(c, err)
	}
	body := jsonx.Obj("order", h.orderJSON(h.locale(c), d, now), "replayed", placed.Replay)
	if placed.Replay {
		return httpx.OK(c, body, T("messages.order_placed", locale))
	}
	return httpx.Created(c, body, T("messages.order_placed", locale))
}

// CancelOrder is POST /shop/orders/:id/cancel: only while every shipment awaits the seller.
func (h *Handlers) CancelOrder(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	id, ok := pathID(c, "id")
	if !ok {
		return notFound(c)
	}
	now := h.now(c)
	if err := h.svc.Cancel(c, uid, id, now); err != nil {
		return render(c, err)
	}
	d, err := h.svc.Order(c, uid, id)
	if err != nil {
		return render(c, err)
	}
	return httpx.OK(c, jsonx.Obj("order", h.orderJSON(h.locale(c), d, now)), T("messages.order_cancelled", i18n.Locale(c)))
}

// RequestReturn is POST /shop/orders/:id/returns {order_item_id, quantity, reason, note} → 201: «درخواست بازگشت»,
// within the seller's return window (7 days by default) after the shipment's delivery.
func (h *Handlers) RequestReturn(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	id, ok := pathID(c, "id")
	if !ok {
		return notFound(c)
	}
	now := h.now(c)
	F := validation.F
	data, err := validate(c, now, validation.Rules{
		F("order_item_id", "required", "integer", "min:1"),
		F("quantity", "nullable", "integer", "min:1", maxQty),
		F("reason", "required", validation.In(ReturnReasons...)),
		F("note", "nullable", "string", "max:1000"),
	}, "order_item_id", "quantity", "reason", "note")
	if err != nil {
		return err
	}
	r, err := h.svc.RequestReturn(c, uid, id, ReturnInput{OrderItemID: num(data, "order_item_id"),
		Quantity: max(int(num(data, "quantity")), 1), Reason: str(data, "reason"), Note: str(data, "note")}, now) //nolint:gosec // G115
	if err != nil {
		return render(c, err)
	}
	return httpx.Created(c, jsonx.Obj("return", returnJSON(r)), T("messages.return_requested", i18n.Locale(c)))
}

// Returns is GET /shop/returns: her return requests, newest first.
func (h *Handlers) Returns(c fiber.Ctx) error {
	uid, err := h.gate(c)
	if err != nil {
		return err
	}
	list, err := h.svc.Returns(c, uid)
	if err != nil {
		return fmt.Errorf("order: returns: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(list))
	for _, r := range list {
		out = append(out, returnJSON(r))
	}
	return httpx.OK(c, jsonx.Obj("returns", out))
}

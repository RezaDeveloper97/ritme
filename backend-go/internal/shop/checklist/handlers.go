package checklist

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/shop"
)

// ForUserField names the owner whose list a companion reads (query of GET /shop/checklist).
const ForUserField = "for_user_id"

// ErrorCodeCompanionForbidden is the 403's error_code when the caller may not read that owner's list.
const ErrorCodeCompanionForbidden = "companion_forbidden"

// ShareAccess decides (and audits) a companion's read of ownerID's list — family share, read only. The routes back it
// with the companion `pregnancy` grant.
type ShareAccess interface {
	CanView(ctx context.Context, ownerID, viewerID uint64) (bool, error)
}

// Handlers serve /api/v1/shop/checklist* and /api/v1/shop/supply-reminder (auth:api).
type Handlers struct {
	svc    *Service
	policy shop.CommercialPolicy
	share  ShareAccess
	clock  clock.Clock
}

// NewHandlers wires the handlers; policy and share may be nil (everyone allowed / no family share).
func NewHandlers(svc *Service, policy shop.CommercialPolicy, share ShareAccess, base clock.Clock) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	return &Handlers{svc: svc, policy: policy, share: share, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// user is the caller, after the commercial policy (a teen-mode account has no shop: 404).
func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	if h.policy != nil {
		allowed, err := h.policy.AllowsCommercial(c, uid)
		if err != nil {
			return 0, fmt.Errorf("shop checklist: policy: %w", err)
		}
		if !allowed {
			return 0, httpx.NotFound()
		}
	}
	return uid, nil
}

func companionForbidden(locale string) error {
	return httpx.Fail(fiber.StatusForbidden, T("messages.companion_forbidden", locale), "error_code", ErrorCodeCompanionForbidden)
}

// owner is the list to read: the caller's, or the one named by ForUserField when the share grant allows it. A
// malformed id is a 422; a missing grant or unknown account the same 403.
func (h *Handlers) owner(c fiber.Ctx, uid uint64, locale string) (uint64, error) {
	raw, ok := validation.Query(c).Get(ForUserField)
	if !ok || raw == nil || phpval.ToString(raw) == "" {
		return uid, nil
	}
	id, err := strconv.ParseUint(phpval.ToString(raw), 10, 64)
	if err != nil || id == 0 {
		return 0, fieldFail(locale, ForUserField, "for_user_invalid")
	}
	if id == uid {
		return uid, nil
	}
	if h.share == nil {
		return 0, companionForbidden(locale)
	}
	allowed, err := h.share.CanView(c, id, uid)
	if err != nil {
		return 0, fmt.Errorf("shop checklist: share: %w", err)
	}
	if !allowed {
		return 0, companionForbidden(locale)
	}
	return id, nil
}

// noDelegation refuses a write naming another account (the family share is read only).
func noDelegation(c fiber.Ctx, uid uint64, locale string) error {
	raw, ok := validation.Input(c).Get(ForUserField)
	if !ok || raw == nil || phpval.ToString(raw) == "" || phpval.ToString(raw) == strconv.FormatUint(uid, 10) {
		return nil
	}
	return companionForbidden(locale)
}

func dateOrNil(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

func nonZero(n int) any {
	if n <= 0 {
		return nil
	}
	return n
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func itemJSON(it *Item) *jsonx.OrderedMap {
	var id, tpl any
	if it.ID > 0 {
		id = it.ID
	}
	if it.LayetteItemID > 0 {
		tpl = it.LayetteItemID
	}
	return jsonx.Obj("id", id, "layette_item_id", tpl, "is_custom", it.Custom(), "title", it.Title,
		"note", strOrNil(it.Note), "suggested_qty", nonZero(it.SuggestedQty), "is_essential", it.IsEssential,
		"is_first_days", it.IsFirstDays, "category_slug", strOrNil(it.CategorySlug), "state", it.State,
		"quantity", nonZero(it.Quantity))
}

// Show is GET /shop/checklist[?for_user_id=] → {due_date, weeks_left, read_only, summary, groups:[…]}. A companion
// with a live pregnancy grant reads the owner's list (read_only true, audited).
func (h *Handlers) Show(c fiber.Ctx) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	owner, err := h.owner(c, uid, locale)
	if err != nil {
		return err
	}
	resolved, def := i18n.ResolveLocale(c, ""), i18n.LanguagesOf(c).DefaultCode()
	pick := func(raw string) string { return i18n.PickString([]byte(raw), resolved, def) }
	now := h.now(c)
	v, err := h.svc.Load(c, owner, civildate.InTehran(now), pick)
	if err != nil {
		return err
	}
	groups := make([]*jsonx.OrderedMap, 0, len(v.Groups))
	for _, g := range v.Groups {
		items := make([]*jsonx.OrderedMap, 0, len(g.Items))
		for i := range g.Items {
			items = append(items, itemJSON(&g.Items[i]))
		}
		groups = append(groups, jsonx.Obj("key", g.Key, "title", GroupTitle(g.Key, locale), "ready", g.Ready,
			"total", g.Total, "items", items))
	}
	var weeks any
	if v.WeeksLeft >= 0 {
		weeks = v.WeeksLeft
	}
	return httpx.OK(c, jsonx.Obj(
		"due_date", dateOrNil(v.DueDate), "weeks_left", weeks, "read_only", owner != uid,
		"summary", jsonx.Obj("ready", v.Summary.Ready, "total", v.Summary.Total, "not_needed", v.Summary.NotNeeded,
			"percent", v.Summary.Percent),
		"groups", groups,
	))
}

func pathID(c fiber.Ctx) (uint64, bool) {
	s := c.Params("id")
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil && n > 0 && len(s) < 20
}

func notFound(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.item_not_found", locale))
}

// SetState is PUT /shop/checklist/items/:id {state, quantity?} on a template item → {layette_item_id, state, quantity}.
func (h *Handlers) SetState(c fiber.Ctx) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := noDelegation(c, uid, locale); err != nil {
		return err
	}
	id, ok := pathID(c)
	if !ok {
		return notFound(locale)
	}
	now := h.now(c)
	in, err := ValidateState(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.SetState(c, uid, id, in.State, in.Quantity, now); err != nil {
		if errors.Is(err, ErrNotFound) {
			return notFound(locale)
		}
		return err
	}
	return httpx.OK(c, jsonx.Obj("layette_item_id", id, "state", in.State, "quantity", nonZero(in.Quantity)),
		T("messages.saved", locale))
}

func customJSON(id uint64, in CustomInput) *jsonx.OrderedMap {
	return itemJSON(&Item{ID: id, Group: in.Group, Title: in.Title, State: in.State, Quantity: in.Quantity})
}

// StoreCustom is POST /shop/checklist/custom {title, group, state?, quantity?} → 201 {item}.
func (h *Handlers) StoreCustom(c fiber.Ctx) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := noDelegation(c, uid, locale); err != nil {
		return err
	}
	now := h.now(c)
	in, err := ValidateCustom(validation.Input(c), nil, locale, now)
	if err != nil {
		return err
	}
	id, err := h.svc.AddCustom(c, uid, in, now)
	if errors.Is(err, ErrTooMany) {
		return fieldFail(locale, "title", "too_many_items")
	}
	if err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("item", customJSON(id, in)), T("messages.item_added", locale))
}

// UpdateCustom is PATCH /shop/checklist/custom/:id (partial) → {item}.
func (h *Handlers) UpdateCustom(c fiber.Ctx) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := noDelegation(c, uid, locale); err != nil {
		return err
	}
	id, ok := pathID(c)
	if !ok {
		return notFound(locale)
	}
	cur, err := h.svc.Custom(c, uid, id)
	if errors.Is(err, ErrNotFound) {
		return notFound(locale)
	}
	if err != nil {
		return err
	}
	now := h.now(c)
	in, err := ValidateCustom(validation.Input(c), &cur, locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UpdateCustom(c, uid, id, in, now); err != nil {
		if errors.Is(err, ErrNotFound) {
			return notFound(locale)
		}
		return err
	}
	return httpx.OK(c, jsonx.Obj("item", customJSON(id, in)), T("messages.saved", locale))
}

// DestroyCustom is DELETE /shop/checklist/custom/:id.
func (h *Handlers) DestroyCustom(c fiber.Ctx) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := pathID(c)
	if !ok {
		return notFound(locale)
	}
	if err := h.svc.DeleteCustom(c, uid, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return notFound(locale)
		}
		return err
	}
	return httpx.Message(c, T("messages.deleted", locale))
}

func supplyJSON(st SupplyState) *jsonx.OrderedMap {
	var product any
	if st.ProductID > 0 {
		product = st.ProductID
	}
	return jsonx.Obj("enabled", st.Enabled, "lead_days", st.LeadDays, "product_id", product,
		"remind_on", dateOrNil(st.RemindOn))
}

// ShowSupply is GET /shop/supply-reminder → {enabled, lead_days, product_id, remind_on}. When on, it re-syncs the care
// reminder with the current prediction first (rolls it to the next cycle); when off it never creates one.
func (h *Handlers) ShowSupply(c fiber.Ctx) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	sup, err := h.svc.Supply(c, uid)
	if err != nil {
		return err
	}
	st := SupplyState{Supply: sup}
	if sup.Enabled {
		if st, err = h.svc.SyncSupply(c, uid, civildate.InTehran(now), now, i18n.Locale(c)); err != nil {
			return err
		}
	}
	return httpx.OK(c, supplyJSON(st))
}

// UpdateSupply is PUT /shop/supply-reminder {enabled, lead_days?, product_id?} → the saved opt-in. Off deletes the
// care reminder it created.
func (h *Handlers) UpdateSupply(c fiber.Ctx) error {
	uid, err := h.user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	now := h.now(c)
	cur, err := h.svc.Supply(c, uid)
	if err != nil {
		return err
	}
	in, err := ValidateSupply(validation.Input(c), cur, locale, now)
	if err != nil {
		return err
	}
	if in.ProductID > 0 && in.ProductID != cur.ProductID {
		ok, err := h.svc.ProductAvailable(c, in.ProductID)
		if err != nil {
			return err
		}
		if !ok {
			return fieldFail(locale, "product_id", "product_invalid")
		}
	}
	st, err := h.svc.SaveSupply(c, uid, in, civildate.InTehran(now), now, locale)
	if err != nil {
		return err
	}
	msg := "messages.supply_off"
	if st.Enabled {
		msg = "messages.supply_on"
	}
	return httpx.OK(c, supplyJSON(st), T(msg, locale))
}

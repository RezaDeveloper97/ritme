package plus

import (
	"context"
	"errors"
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
)

// ErrorCodeTeenCommercialBlocked is the 403 of a trial start / checkout by a teen-mode account (CB-TEEN-01).
const ErrorCodeTeenCommercialBlocked = "teen_commercial_blocked"

// CommercialPolicy decides whether an account may be sold to (internal/teen.Policy: never a teen-mode account).
type CommercialPolicy interface {
	AllowsCommercial(ctx context.Context, userID uint64) (bool, error)
}

// Handlers are the /api/v1/plus actions. Mount them behind the locale middleware; everything but /plans needs
// auth RequireUser.
type Handlers struct {
	svc    *Service
	clock  clock.Clock
	policy CommercialPolicy
}

// NewHandlers wires the handlers; base is the fallback clock (X-Test-Now pins it per request in tests).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

// WithCommercial sets the commercial policy (CB-TEEN-01): for an account it refuses, the trial offer is never shown
// (status / trial sheet) and starting a trial or checking out answers 403 teen_commercial_blocked. nil = everyone.
func (h *Handlers) WithCommercial(p CommercialPolicy) *Handlers {
	h.policy = p
	return h
}

// commercialAllowed reports whether userID may see offers and buy.
func (h *Handlers) commercialAllowed(c fiber.Ctx, userID uint64) (bool, error) {
	if h.policy == nil {
		return true, nil
	}
	return h.policy.AllowsCommercial(c, userID)
}

// refuseBlocked is the 403 for an account that may not buy (nil when it may).
func (h *Handlers) refuseBlocked(c fiber.Ctx, userID uint64) error {
	ok, err := h.commercialAllowed(c, userID)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return httpx.Fail(fiber.StatusForbidden, T("errors.teen_commercial_blocked", i18n.Locale(c)),
		"error_code", ErrorCodeTeenCommercialBlocked)
}

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// now is the request time, Tehran wall-clock, whole seconds.
func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func localizer(c fiber.Ctx) Localizer {
	return Localizer{Locale: i18n.Locale(c), Default: i18n.LanguagesOf(c).DefaultCode()}
}

func (h *Handlers) configured(c fiber.Ctx) (Configured, error) {
	vat, err := h.svc.VATRateBps(c)
	if err != nil {
		return Configured{}, err
	}
	return Configured{VATRateBps: vat, TrialDays: h.svc.Config().TrialDays}, nil
}

// fail maps a domain error to its response (unknown errors fall through to the 500 handler).
func fail(err error, locale string) error {
	code := func(status int, key, errorCode string) error {
		return httpx.Fail(status, T("errors."+key, locale), "error_code", errorCode)
	}
	var de *DiscountError
	switch {
	case errors.As(err, &de):
		return discountInvalid(locale, de.Reason)
	case errors.Is(err, ErrPlanNotFound):
		return planInvalid(locale)
	case errors.Is(err, ErrPackNotFound):
		return failValidation(locale, jsonx.Obj("pack_id", []string{T("errors.pack_invalid", locale)}))
	case errors.Is(err, ErrTrialUsed):
		return code(fiber.StatusUnprocessableEntity, "trial_used", "trial_used")
	case errors.Is(err, ErrTrialUnavailable):
		return code(fiber.StatusUnprocessableEntity, "trial_unavailable", "trial_unavailable")
	case errors.Is(err, ErrNoSubscription):
		return code(fiber.StatusUnprocessableEntity, "no_subscription", "no_subscription")
	case errors.Is(err, ErrInvoiceNotFound):
		return code(fiber.StatusNotFound, "invoice_not_found", "invoice_not_found")
	case errors.Is(err, ErrInvoiceClosed):
		return code(fiber.StatusUnprocessableEntity, "invoice_closed", "invoice_closed")
	case errors.Is(err, ErrAuthorityMismatch):
		return code(fiber.StatusUnprocessableEntity, "authority_mismatch", "authority_mismatch")
	case errors.Is(err, ErrPaymentFailed):
		return code(fiber.StatusUnprocessableEntity, "payment_failed", "payment_failed")
	case errors.Is(err, ErrAmountMismatch):
		return code(fiber.StatusUnprocessableEntity, "amount_mismatch", "amount_mismatch")
	case errors.Is(err, ErrPaymentUnavailable):
		return code(fiber.StatusServiceUnavailable, "payment_unavailable", "payment_unavailable")
	case errors.Is(err, ErrStoreSignature), errors.Is(err, ErrStorePurchaseBad):
		return code(fiber.StatusUnprocessableEntity, "store_purchase_invalid", "store_purchase_invalid")
	case errors.Is(err, ErrStoreProduct):
		return code(fiber.StatusUnprocessableEntity, "store_product_unknown", "store_product_unknown")
	case errors.Is(err, ErrStorePayload):
		return code(fiber.StatusConflict, "store_purchase_owned", "store_purchase_owned")
	}
	return err
}

// status renders the user's status, optionally with a message.
func (h *Handlers) status(c fiber.Ctx, userID uint64, now time.Time, msg ...string) error {
	st, err := h.svc.Status(c, userID, now)
	if err != nil {
		return err
	}
	ok, err := h.commercialAllowed(c, userID)
	if err != nil {
		return err
	}
	if !ok {
		st.Offer = nil // no trial offer for a teen-mode account (CB-TEEN-01)
	}
	return httpx.OK(c, StatusJSON(st, localizer(c)), msg...)
}

// Plans is GET /plus/plans (public): the active plans with per-month price and savings, the VAT rate and the
// trial length.
func (h *Handlers) Plans(c fiber.Ctx) error {
	plans, err := h.svc.Plans(c)
	if err != nil {
		return err
	}
	cfg, err := h.configured(c)
	if err != nil {
		return err
	}
	skus, trialSKU, err := h.svc.StoreProducts(c)
	if err != nil {
		return err
	}
	for i := range plans {
		for st, byPlan := range skus {
			if sku, ok := byPlan[plans[i].ID]; ok {
				if plans[i].StoreSKUs == nil {
					plans[i].StoreSKUs = map[string]string{}
				}
				plans[i].StoreSKUs[st] = sku
			}
		}
	}
	cfg.StoreChannels, cfg.StoreTrialSKU = h.svc.StoreChannels(), trialSKU
	return httpx.OK(c, PlansJSON(plans, cfg, localizer(c)))
}

// Status is GET /plus/status: tier, subscription, trial and this month's entitlements.
func (h *Handlers) Status(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	return h.status(c, userID, h.now(c))
}

// Usage is GET /plus/usage: this month's entitlements with counters.
func (h *Handlers) Usage(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	st, err := h.svc.Status(c, userID, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, UsageJSON(st.Usage, st.Tier))
}

// StartTrial is POST /plus/trial/start: the one free trial, then the status.
func (h *Handlers) StartTrial(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if err := h.refuseBlocked(c, userID); err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	if err := h.svc.StartTrial(c, userID, now); err != nil {
		return fail(err, locale)
	}
	return h.status(c, userID, now, T("messages.trial_started", locale))
}

// Checkout is POST /plus/checkout {plan_id, discount_code?, preview?}. preview=true prices without creating
// anything ({quote}); otherwise a pending invoice and the gateway redirect ({invoice, payment}). A 100 % discount
// is settled at once (payment null, invoice paid).
func (h *Handlers) Checkout(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if err := h.refuseBlocked(c, userID); err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateCheckout(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	l := localizer(c)
	if in.Preview {
		q, err := h.svc.Preview(c, userID, in.PlanID, in.DiscountCode, now)
		if err != nil {
			return fail(err, locale)
		}
		return httpx.OK(c, jsonx.Obj("quote", QuoteJSON(q, l)))
	}
	out, err := h.svc.Checkout(c, userID, in.PlanID, in.DiscountCode, now)
	if err != nil {
		return fail(err, locale)
	}
	gateway := ""
	if out.Invoice.Gateway.Valid {
		gateway = out.Invoice.Gateway.String
	}
	msg := T("messages.checkout_created", locale)
	if out.Payment == nil {
		msg = T("messages.checkout_paid", locale)
	}
	return httpx.OK(c, jsonx.Obj(
		"invoice", InvoiceJSON(out.Invoice, &out.Plan, nil, l),
		"payment", PaymentJSON(out.Payment, gateway),
	), msg)
}

// Packs is GET /plus/credit-packs (public): the purchasable packs of extra AI credits.
func (h *Handlers) Packs(c fiber.Ctx) error {
	packs, err := h.svc.Packs(c)
	if err != nil {
		return err
	}
	vat, err := h.svc.VATRateBps(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, PacksJSON(packs, vat, localizer(c)))
}

// CheckoutPack is POST /plus/credit-packs/checkout {pack_id, preview?}: the checkout of a credit pack. The answer
// has the shape of POST /plus/checkout ({quote} for a preview, else {invoice, payment}); the payment is settled by
// POST /plus/verify like any other invoice, which then grants the credits.
func (h *Handlers) CheckoutPack(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if err := h.refuseBlocked(c, userID); err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	F := validation.F
	data, err := validate(validation.Input(c), []string{"pack_id", "preview"}, validation.Rules{
		F("pack_id", "required", "integer", "min:1"),
		F("preview", "nullable", "boolean"),
	}, locale, now)
	if err != nil {
		return err
	}
	packID, _ := strconv.ParseUint(str(data, "pack_id"), 10, 64)
	preview, _ := data.Get("preview")
	l := localizer(c)
	if phpval.Truthy(preview) {
		q, err := h.svc.PreviewPack(c, packID)
		if err != nil {
			return fail(err, locale)
		}
		return httpx.OK(c, jsonx.Obj("quote", PackQuoteJSON(q, l)))
	}
	out, err := h.svc.CheckoutPack(c, userID, packID, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj(
		"invoice", PackInvoiceJSON(out.Invoice, nil, out.Pack, nil, l),
		"payment", PaymentJSON(out.Payment, out.Invoice.Gateway.String),
	), T("messages.checkout_created", locale))
}

// Verify is POST /plus/verify {reference, authority, status?}: settles the user's invoice with the gateway.
// Idempotent — a replay of a verified invoice answers 200 with the same invoice and adds no time.
func (h *Handlers) Verify(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateVerify(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	var callback map[string]string
	if in.Status != "" {
		callback = map[string]string{"status": in.Status}
	}
	res, err := h.svc.Verify(c, userID, in.Reference, in.Authority, callback, now)
	if err != nil {
		return fail(err, locale)
	}
	l := localizer(c)
	plan, err := h.svc.Plan(c, res.Invoice.PlanID)
	if err != nil {
		return err
	}
	receipt, err := h.svc.receipt(c, res.Invoice, userID)
	if err != nil {
		return err
	}
	st, err := h.svc.Status(c, userID, now)
	if err != nil {
		return err
	}
	pack, err := h.svc.Pack(c, res.Invoice.CreditPackID)
	if err != nil {
		return err
	}
	msg := T("messages.payment_verified", locale)
	if res.Invoice.Kind == KindCreditPack {
		msg = T("messages.credits_added", locale)
	}
	if res.AlreadyPaid {
		msg = T("messages.payment_already_verified", locale)
	}
	return httpx.OK(c, jsonx.Obj(
		"invoice", PackInvoiceJSON(res.Invoice, plan, pack, receipt, l),
		"already_verified", res.AlreadyPaid,
		"status", StatusJSON(st, l),
	), msg)
}

// Cancel is POST /plus/cancel: auto-renew off; Plus stays until the period ends. Then the status.
func (h *Handlers) Cancel(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	if err := h.svc.Cancel(c, userID, now); err != nil {
		return fail(err, locale)
	}
	return h.status(c, userID, now, T("messages.subscription_canceled", locale))
}

// Restore is POST /plus/restore: re-checks pending checkouts with the gateway, then the status (+ restored count).
func (h *Handlers) Restore(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	n, err := h.svc.Restore(c, userID, now)
	if err != nil {
		return err
	}
	st, err := h.svc.Status(c, userID, now)
	if err != nil {
		return err
	}
	msg := T("messages.restored", locale)
	if n == 0 {
		msg = T("messages.nothing_to_restore", locale)
	}
	return httpx.OK(c, jsonx.Obj("restored", n, "status", StatusJSON(st, localizer(c))), msg)
}

// History is GET /plus/history: the latest 50 invoices, newest first, with receipts.
func (h *Handlers) History(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	entries, err := h.svc.History(c, userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, HistoryJSON(entries, localizer(c)))
}

// TrialSheet is GET /plus/trial: the trial sheet (nbl_Prem_TrialSheet) — trial countdown, the running trial
// offer, every plan with its offer price and the Plus features used since the trial began.
func (h *Handlers) TrialSheet(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	sh, err := h.svc.TrialSheet(c, userID, h.now(c))
	if err != nil {
		return err
	}
	ok, err := h.commercialAllowed(c, userID)
	if err != nil {
		return err
	}
	if !ok {
		sh.Offer, sh.State.Offer = nil, nil // no trial offer (or offer prices) for a teen-mode account
	}
	return httpx.OK(c, TrialSheetJSON(sh, localizer(c)))
}

// StoreIntent is POST /plus/store/intent {store, sku}: checks the SKU is sellable to the user on that store and
// returns the developer payload the app passes to the store's purchase call.
func (h *Handlers) StoreIntent(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if err := h.refuseBlocked(c, userID); err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateStoreIntent(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	it, err := h.svc.Intent(c, userID, in.Store, in.SKU, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("store", it.Store, "sku", it.SKU, "kind", it.Kind, "payload", it.Payload))
}

// StoreVerify is POST /plus/store/verify {store, signed_data, signature}: redeems one store purchase (a plan period
// or the trial), then the status. Idempotent: a replay answers 200 with already_verified true and adds no time.
func (h *Handlers) StoreVerify(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if err := h.refuseBlocked(c, userID); err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateStoreVerify(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	p := in.Purchases[0]
	res, err := h.svc.StoreVerify(c, userID, in.Store, p.SignedData, p.Signature, now)
	if err != nil {
		return fail(err, locale)
	}
	l := localizer(c)
	var invoice any
	if res.Invoice != nil {
		plan, err := h.svc.Plan(c, res.Invoice.PlanID)
		if err != nil {
			return err
		}
		receipt, err := h.svc.receipt(c, *res.Invoice, userID)
		if err != nil {
			return err
		}
		invoice = InvoiceJSON(*res.Invoice, plan, receipt, l)
	}
	st, err := h.svc.Status(c, userID, now)
	if err != nil {
		return err
	}
	msg := T("messages.payment_verified", locale)
	switch {
	case res.AlreadyDone:
		msg = T("messages.payment_already_verified", locale)
	case res.Kind == StoreKindTrial:
		msg = T("messages.trial_started", locale)
	}
	return httpx.OK(c, jsonx.Obj(
		"kind", res.Kind,
		"sku", res.SKU,
		"invoice", invoice,
		"already_verified", res.AlreadyDone,
		"status", StatusJSON(st, l),
	), msg)
}

// StoreSync is POST /plus/store/sync {store, purchases: [{signed_data, signature}]}: the app's active store
// purchases on open (restore after reinstall, subscription renewals). Each purchase is redeemed like StoreVerify;
// rejected ones are skipped (counted), never fail the call. Then the status.
func (h *Handlers) StoreSync(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateStoreSync(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if len(in.Purchases) > 0 && !h.svc.storeConfigured(in.Store) {
		return fail(ErrPaymentUnavailable, locale)
	}
	restored, rejected := 0, 0
	for _, p := range in.Purchases {
		res, err := h.svc.StoreVerify(c, userID, in.Store, p.SignedData, p.Signature, now)
		switch {
		case err == nil && !res.AlreadyDone:
			restored++
		case err == nil:
		case errors.Is(err, ErrStoreSignature), errors.Is(err, ErrStoreProduct), errors.Is(err, ErrStorePayload),
			errors.Is(err, ErrStorePurchaseBad), errors.Is(err, ErrTrialUsed), errors.Is(err, ErrTrialUnavailable):
			rejected++
		default:
			return err
		}
	}
	st, err := h.svc.Status(c, userID, now)
	if err != nil {
		return err
	}
	msg := T("messages.restored", locale)
	if restored == 0 {
		msg = T("messages.nothing_to_restore", locale)
	}
	return httpx.OK(c, jsonx.Obj("restored", restored, "rejected", rejected, "status", StatusJSON(st, localizer(c))), msg)
}

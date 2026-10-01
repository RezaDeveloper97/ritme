package plus

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers are the /api/v1/plus actions. Mount them behind the locale middleware; everything but /plans needs
// auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (X-Test-Now pins it per request in tests).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

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

func (h *Handlers) configured() Configured {
	cfg := h.svc.Config()
	return Configured{VATRateBps: cfg.VATRateBps, TrialDays: cfg.TrialDays}
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
	}
	return err
}

// status renders the user's status, optionally with a message.
func (h *Handlers) status(c fiber.Ctx, userID uint64, now time.Time, msg ...string) error {
	st, err := h.svc.Status(c, userID, now)
	if err != nil {
		return err
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
	return httpx.OK(c, PlansJSON(plans, h.configured(), localizer(c)))
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
	msg := T("messages.payment_verified", locale)
	if res.AlreadyPaid {
		msg = T("messages.payment_already_verified", locale)
	}
	return httpx.OK(c, jsonx.Obj(
		"invoice", InvoiceJSON(res.Invoice, plan, receipt, l),
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

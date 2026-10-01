package plus

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/plus/store"
)

// Domain errors (the handlers map them to responses).
var (
	ErrPlanNotFound       = errors.New("plus: plan not found")
	ErrTrialUsed          = errors.New("plus: trial already used")
	ErrTrialUnavailable   = errors.New("plus: trial unavailable after a subscription")
	ErrNoSubscription     = errors.New("plus: no active subscription")
	ErrInvoiceNotFound    = errors.New("plus: invoice not found")
	ErrInvoiceClosed      = errors.New("plus: invoice is not pending")
	ErrAuthorityMismatch  = errors.New("plus: authority does not match the invoice")
	ErrPaymentFailed      = errors.New("plus: payment not completed")
	ErrAmountMismatch     = errors.New("plus: settled amount differs from the invoice")
	ErrPaymentUnavailable = errors.New("plus: payment gateway unavailable")
	ErrUnknownFeature     = errors.New("plus: unknown entitlement key")
	ErrNotEntitled        = errors.New("plus: feature requires Plus")
	ErrQuotaExceeded      = errors.New("plus: monthly quota reached")
)

// Invoice statuses.
const (
	InvoicePending  = "pending"
	InvoicePaid     = "paid"
	InvoiceFailed   = "failed"
	InvoiceExpired  = "expired"
	InvoiceRefunded = "refunded"
)

// Service is the subscription domain. It takes the clock's "now" on every call (handlers pass the request clock,
// so X-Test-Now drives trials, periods and quotas in tests).
type Service struct {
	db      *sql.DB
	q       *store.Queries
	cfg     config.Plus
	gateway Gateway // nil = no payment provider (checkout/verify answer 503)
	logger  *slog.Logger
}

// NewService wires the domain. gateway may be nil (payments unavailable).
func NewService(db *sql.DB, cfg config.Plus, gateway Gateway, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{db: db, q: store.New(db), cfg: cfg, gateway: gateway, logger: logger}
}

// Config is the checkout configuration (env VAT rate — see VATRateBps for the effective one —, trial length).
func (s *Service) Config() config.Plus { return s.cfg }

// ---------------------------------------------------------------------------
// Plans

// Plan is an active plan with its derived display figures.
type Plan struct {
	store.PlusPlan
	Monthly uint64 // rials per month
	Savings int    // percent cheaper per month than the base plan
}

// Plans lists the active plans in display order.
func (s *Service) Plans(ctx context.Context) ([]Plan, error) {
	rows, err := s.q.ListActivePlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("plus: plans: %w", err)
	}
	base := BaseMonthly(rows)
	out := make([]Plan, len(rows))
	for i, p := range rows {
		out[i] = Plan{PlusPlan: p, Monthly: MonthlyPrice(p), Savings: SavingsPercent(p, base)}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Status, entitlements, usage

// State is what the user has right now.
type State struct {
	Now              time.Time
	Tier             Tier
	Subscription     *store.PlusSubscription // the period covering now
	SubscriptionPlan *store.PlusPlan         // its plan (nil when deleted)
	Trial            *store.PlusTrial        // the trial row, running or not
	TrialAvailable   bool                    // never trialled and never subscribed
	Usage            Usage
	Offer            *Offer // the running trial offer (nil when none)
}

// Usage is this month's resolved entitlements.
type Usage struct {
	PeriodStart  civildate.Date // first day of the month (Tehran)
	ResetsAt     time.Time      // first instant of next month (Tehran)
	Entitlements []Entitlement
}

// PeriodStart is the quota month of now: the first day of its calendar month in Tehran.
func PeriodStart(now time.Time) civildate.Date {
	t := now.In(civildate.Tehran)
	return civildate.New(t.Year(), t.Month(), 1)
}

func resetsAt(now time.Time) time.Time {
	t := now.In(civildate.Tehran)
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, civildate.Tehran)
}

// tierOf reads the current subscription and trial.
func tierOf(ctx context.Context, q *store.Queries, userID uint64, now time.Time) (Tier, *store.PlusSubscription, *store.PlusTrial, error) {
	var sub *store.PlusSubscription
	row, err := q.CurrentSubscription(ctx, store.CurrentSubscriptionParams{UserID: userID, Now: now})
	switch {
	case err == nil:
		sub = &row
	case !errors.Is(err, sql.ErrNoRows):
		return "", nil, nil, fmt.Errorf("plus: subscription: %w", err)
	}
	var trial *store.PlusTrial
	tr, err := q.GetTrial(ctx, userID)
	switch {
	case err == nil:
		trial = &tr
	case !errors.Is(err, sql.ErrNoRows):
		return "", nil, nil, fmt.Errorf("plus: trial: %w", err)
	}
	switch {
	case sub != nil:
		return TierPlus, sub, trial, nil
	case trial != nil && trialRunning(*trial, now):
		return TierTrial, nil, trial, nil
	}
	return TierFree, nil, trial, nil
}

func trialRunning(t store.PlusTrial, now time.Time) bool {
	return !now.Before(t.StartedAt) && now.Before(t.EndsAt)
}

func (s *Service) usage(ctx context.Context, userID uint64, tier Tier, now time.Time) (Usage, error) {
	period := PeriodStart(now)
	rows, err := s.q.ListUsage(ctx, store.ListUsageParams{UserID: userID, PeriodStart: period})
	if err != nil {
		return Usage{}, fmt.Errorf("plus: usage: %w", err)
	}
	used := make(map[Key]int, len(rows))
	for _, r := range rows {
		used[Key(r.Feature)] = int(r.Used)
	}
	return Usage{PeriodStart: period, ResetsAt: resetsAt(now), Entitlements: Resolve(tier, used)}, nil
}

// Status is the user's tier, subscription, trial and entitlements at now.
func (s *Service) Status(ctx context.Context, userID uint64, now time.Time) (State, error) {
	tier, sub, trial, err := tierOf(ctx, s.q, userID, now)
	if err != nil {
		return State{}, err
	}
	st := State{Now: now, Tier: tier, Subscription: sub, Trial: trial}
	if sub != nil && sub.PlanID.Valid {
		p, err := s.q.GetPlan(ctx, uint64(sub.PlanID.Int64)) //nolint:gosec // G115: ids are positive
		switch {
		case err == nil:
			st.SubscriptionPlan = &p
		case !errors.Is(err, sql.ErrNoRows):
			return State{}, fmt.Errorf("plus: plan: %w", err)
		}
	}
	if trial == nil {
		had, err := s.q.HasAnySubscription(ctx, userID)
		if err != nil {
			return State{}, fmt.Errorf("plus: subscriptions: %w", err)
		}
		st.TrialAvailable = !had
	}
	if st.Usage, err = s.usage(ctx, userID, tier, now); err != nil {
		return State{}, err
	}
	if tier == TierTrial {
		pct, err := s.TrialOfferPercent(ctx)
		if err != nil {
			return State{}, err
		}
		if pct > 0 {
			if st.Offer, err = s.offer(ctx, pct, trial.EndsAt, now); err != nil {
				return State{}, err
			}
		}
	}
	return st, nil
}

// Usage is this month's entitlements and counters.
func (s *Service) Usage(ctx context.Context, userID uint64, now time.Time) (Usage, error) {
	tier, _, _, err := tierOf(ctx, s.q, userID, now)
	if err != nil {
		return Usage{}, err
	}
	return s.usage(ctx, userID, tier, now)
}

// Entitlement resolves one feature for the user without counting a use (gating checks, B-N2-06).
func (s *Service) Entitlement(ctx context.Context, userID uint64, key Key, now time.Time) (Entitlement, error) {
	e, _, err := s.entitlement(ctx, userID, key, now)
	return e, err
}

// entitlement is Entitlement plus the tier it was resolved for.
func (s *Service) entitlement(ctx context.Context, userID uint64, key Key, now time.Time) (Entitlement, Tier, error) {
	def, ok := Lookup(key)
	if !ok {
		return Entitlement{}, "", ErrUnknownFeature
	}
	tier, _, _, err := tierOf(ctx, s.q, userID, now)
	if err != nil {
		return Entitlement{}, "", err
	}
	u, err := s.usage(ctx, userID, tier, now)
	if err != nil {
		return Entitlement{}, "", err
	}
	for _, e := range u.Entitlements {
		if e.Key == def.Key {
			return e, tier, nil
		}
	}
	return Entitlement{}, "", ErrUnknownFeature
}

// Consume counts one use of key for the user at now and returns the entitlement after it. A locked feature is
// ErrNotEntitled, a spent monthly quota ErrQuotaExceeded (nothing counted). The increment is a single atomic
// statement, so concurrent uses can never exceed the quota. Unlimited features are still counted (the trial
// sheet's «۸ پیام» lines).
func (s *Service) Consume(ctx context.Context, userID uint64, key Key, now time.Time) (Entitlement, error) {
	def, ok := Lookup(key)
	if !ok {
		return Entitlement{}, ErrUnknownFeature
	}
	tier, _, _, err := tierOf(ctx, s.q, userID, now)
	if err != nil {
		return Entitlement{}, err
	}
	quota := def.QuotaFor(tier)
	period := PeriodStart(now)
	stamp := sql.NullTime{Time: now, Valid: true}
	switch {
	case !quota.Enabled:
		err = ErrNotEntitled
	case quota.Limit == 0:
		err = s.q.IncrementUsage(ctx, store.IncrementUsageParams{UserID: userID, Feature: string(key), PeriodStart: period, Now: stamp})
	default:
		var n int64
		n, err = s.q.IncrementUsageWithin(ctx, store.IncrementUsageWithinParams{
			UserID: userID, Feature: string(key), PeriodStart: period, Now: stamp, Quota: uint32(quota.Limit), //nolint:gosec // G115: small positive quota
		})
		if err == nil && n == 0 {
			err = ErrQuotaExceeded
		}
	}
	if err != nil && !errors.Is(err, ErrNotEntitled) && !errors.Is(err, ErrQuotaExceeded) {
		return Entitlement{}, fmt.Errorf("plus: consume %s: %w", key, err)
	}
	u, uerr := s.usage(ctx, userID, tier, now)
	if uerr != nil {
		return Entitlement{}, uerr
	}
	for _, e := range u.Entitlements {
		if e.Key == key {
			return e, err
		}
	}
	return Entitlement{}, ErrUnknownFeature
}

// ---------------------------------------------------------------------------
// Trial

// StartTrial starts the one 7-day trial of the user (config PLUS_TRIAL_DAYS). A user who already trialled gets
// ErrTrialUsed — enforced by UNIQUE(user_id), so concurrent starts cannot both win — and one who has ever
// subscribed ErrTrialUnavailable («۷ روز رایگان برای اولین اشتراک»).
func (s *Service) StartTrial(ctx context.Context, userID uint64, now time.Time) error {
	if _, err := s.q.GetTrial(ctx, userID); err == nil {
		return ErrTrialUsed
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("plus: trial: %w", err)
	}
	had, err := s.q.HasAnySubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("plus: subscriptions: %w", err)
	}
	if had {
		return ErrTrialUnavailable
	}
	stamp := sql.NullTime{Time: now, Valid: true}
	err = s.q.InsertTrial(ctx, store.InsertTrialParams{
		UserID: userID, StartedAt: now, EndsAt: now.AddDate(0, 0, s.cfg.TrialDays), CreatedAt: stamp, UpdatedAt: stamp,
	})
	if isDuplicate(err) {
		return ErrTrialUsed
	}
	if err != nil {
		return fmt.Errorf("plus: start trial: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Checkout

// Quote is a priced plan with an optional discount: a discount code or the running trial offer, whichever is
// larger (they never stack; on a tie the offer wins and the code is not redeemed).
type Quote struct {
	Plan           store.PlusPlan
	Amounts        Amounts
	DiscountCodeID sql.NullInt64
	DiscountCode   string
	TrialOffer     *AppliedOffer // the trial offer priced this quote (nil when a code or nothing did)
}

// AppliedOffer is the trial offer a quote used.
type AppliedOffer struct {
	Percent int
	EndsAt  time.Time
}

// quote prices planID with code through q. The code row is read FOR UPDATE: inside a transaction this serialises
// concurrent checkouts on one code until the invoice is written.
func (s *Service) quote(ctx context.Context, q *store.Queries, userID, planID uint64, code string, now time.Time) (Quote, error) {
	plan, err := q.GetActivePlan(ctx, planID)
	if errors.Is(err, sql.ErrNoRows) {
		return Quote{}, ErrPlanNotFound
	}
	if err != nil {
		return Quote{}, fmt.Errorf("plus: plan: %w", err)
	}
	rate, err := vatRateBps(ctx, q, s) // admin override (B-N2-09), else PLUS_VAT_RATE_BPS; the invoice snapshots it
	if err != nil {
		return Quote{}, err
	}
	vat := uint32(rate) //nolint:gosec // G115: validated 0–10000 (config / parseBps)
	base := Quote{Plan: plan, Amounts: Price(plan.PriceRials, "", 0, vat)}
	pct, trial, err := offerPercent(ctx, q, s, userID, now)
	if err != nil {
		return Quote{}, err
	}
	if pct > 0 {
		base.Amounts = Price(plan.PriceRials, DiscountPercent, uint64(pct), vat) //nolint:gosec // G115: 1–100
		base.TrialOffer = &AppliedOffer{Percent: pct, EndsAt: trial.EndsAt}
	}
	if code == "" {
		return base, nil
	}
	dc, err := q.LockDiscountCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return Quote{}, &DiscountError{Reason: DiscountInvalid}
	}
	if err != nil {
		return Quote{}, fmt.Errorf("plus: discount: %w", err)
	}
	id := sql.NullInt64{Int64: int64(dc.ID), Valid: true} //nolint:gosec // G115: ids are positive
	total, err := q.CountDiscountRedemptions(ctx, store.CountDiscountRedemptionsParams{DiscountCodeID: id, Now: now})
	if err != nil {
		return Quote{}, fmt.Errorf("plus: discount redemptions: %w", err)
	}
	mine, err := q.CountUserDiscountRedemptions(ctx, store.CountUserDiscountRedemptionsParams{DiscountCodeID: id, UserID: userID, Now: now})
	if err != nil {
		return Quote{}, fmt.Errorf("plus: discount redemptions: %w", err)
	}
	if err := CheckDiscount(dc, planID, now, total, mine); err != nil {
		return Quote{}, err
	}
	withCode := Quote{Plan: plan, Amounts: Price(plan.PriceRials, dc.Kind, dc.Value, vat), DiscountCodeID: id, DiscountCode: dc.Code}
	if base.TrialOffer != nil && base.Amounts.Discount >= withCode.Amounts.Discount {
		return base, nil
	}
	return withCode, nil
}

// Preview prices a checkout without creating anything (the «اعمال شد» discount line).
func (s *Service) Preview(ctx context.Context, userID, planID uint64, code string, now time.Time) (Quote, error) {
	return s.quote(ctx, s.q, userID, planID, code, now)
}

// Checkout is a created invoice and, unless it was free, where to pay it.
type Checkout struct {
	Invoice store.PlusInvoice
	Plan    store.PlusPlan
	Payment *PaymentSession // nil when the invoice was settled at once (100 % discount)
}

// Checkout prices planID (the client never sends an amount), writes a pending invoice holding the discount for
// PLUS_INVOICE_TTL_MINUTES, and registers the payment with the gateway.
func (s *Service) Checkout(ctx context.Context, userID, planID uint64, code string, now time.Time) (Checkout, error) {
	if s.gateway == nil {
		return Checkout{}, ErrPaymentUnavailable
	}
	ref, err := newReference()
	if err != nil {
		return Checkout{}, err
	}
	var qt Quote
	var invoiceID int64
	err = s.inTx(ctx, func(q *store.Queries) error {
		var err error
		if qt, err = s.quote(ctx, q, userID, planID, code, now); err != nil {
			return err
		}
		stamp := sql.NullTime{Time: now, Valid: true}
		invoiceID, err = q.InsertInvoice(ctx, store.InsertInvoiceParams{
			Reference: ref, UserID: userID, PlanID: sql.NullInt64{Int64: int64(qt.Plan.ID), Valid: true}, //nolint:gosec // G115: ids are positive
			DurationMonths: qt.Plan.DurationMonths,
			SubtotalRials:  qt.Amounts.Subtotal, DiscountRials: qt.Amounts.Discount,
			VatRateBps: qt.Amounts.VATRateBps, VatRials: qt.Amounts.VAT, TotalRials: qt.Amounts.Total,
			DiscountCodeID: qt.DiscountCodeID, DiscountCode: sql.NullString{String: qt.DiscountCode, Valid: qt.DiscountCode != ""},
			Gateway:   sql.NullString{String: s.gateway.Name(), Valid: true},
			ExpiresAt: now.Add(s.cfg.InvoiceTTL), CreatedAt: stamp, UpdatedAt: stamp,
		})
		if err != nil {
			return fmt.Errorf("plus: invoice: %w", err)
		}
		if qt.Amounts.Total > 0 {
			return nil
		}
		// Nothing to charge: settle now, no gateway round trip.
		if err := q.MarkInvoicePaidFree(ctx, store.MarkInvoicePaidFreeParams{Now: stamp, ID: uint64(invoiceID)}); err != nil { //nolint:gosec // G115: ids are positive
			return fmt.Errorf("plus: invoice: %w", err)
		}
		return activate(ctx, q, userID, qt.Plan.ID, uint64(invoiceID), int(qt.Plan.DurationMonths), now) //nolint:gosec // G115: ids are positive
	})
	if err != nil {
		return Checkout{}, err
	}
	out := Checkout{Plan: qt.Plan}
	if qt.Amounts.Total > 0 {
		sess, err := s.gateway.Create(ctx, PaymentRequest{
			Reference: ref, AmountRials: qt.Amounts.Total, Description: "Ritme Plus " + qt.Plan.Code, CallbackURL: s.cfg.CallbackURL,
		})
		if err != nil {
			s.logger.WarnContext(ctx, "plus: gateway create failed", "gateway", s.gateway.Name(), "reference", ref, "error", err.Error())
			_ = s.q.MarkInvoiceFailed(ctx, store.MarkInvoiceFailedParams{Now: sql.NullTime{Time: now, Valid: true}, ID: uint64(invoiceID)}) //nolint:gosec // G115
			return Checkout{}, fmt.Errorf("%w: %w", ErrPaymentUnavailable, err)
		}
		if err := s.q.SetInvoiceAuthority(ctx, store.SetInvoiceAuthorityParams{
			Authority: sql.NullString{String: sess.Authority, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true}, ID: uint64(invoiceID), //nolint:gosec // G115
		}); err != nil {
			return Checkout{}, fmt.Errorf("plus: invoice authority: %w", err)
		}
		out.Payment = &sess
	}
	if out.Invoice, err = s.q.GetUserInvoiceByReference(ctx, store.GetUserInvoiceByReferenceParams{Reference: ref, UserID: userID}); err != nil {
		return Checkout{}, fmt.Errorf("plus: invoice: %w", err)
	}
	return out, nil
}

// activate adds a subscription period for a paid invoice: it starts now, or — when a period is still running or
// queued — right after the last one ends (buying again extends, never overlaps).
func activate(ctx context.Context, q *store.Queries, userID, planID, invoiceID uint64, months int, now time.Time) error {
	start := now
	last, err := q.LatestValidSubscription(ctx, store.LatestValidSubscriptionParams{UserID: userID, Now: now})
	switch {
	case err == nil && last.EndsAt.After(now):
		start = last.EndsAt
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("plus: latest subscription: %w", err)
	}
	stamp := sql.NullTime{Time: now, Valid: true}
	_, err = q.InsertSubscription(ctx, store.InsertSubscriptionParams{
		UserID: userID, PlanID: sql.NullInt64{Int64: int64(planID), Valid: planID > 0}, //nolint:gosec // G115: ids are positive
		InvoiceID: sql.NullInt64{Int64: int64(invoiceID), Valid: true}, //nolint:gosec // G115: ids are positive
		Source:    "purchase", StartsAt: start, EndsAt: AddMonths(start, months), CreatedAt: stamp, UpdatedAt: stamp,
	})
	if err != nil {
		return fmt.Errorf("plus: subscription: %w", err)
	}
	return nil
}

// Settled is the outcome of a verify.
type Settled struct {
	Invoice     store.PlusInvoice
	AlreadyPaid bool // the invoice had been verified before (idempotent replay)
}

// Verify settles the user's invoice reference after the gateway sent the user back. It is idempotent: a second
// verify of a paid invoice returns it unchanged (AlreadyPaid) without asking the gateway again or adding time.
// The invoice row is locked for the whole settlement, so concurrent verifies run one after the other.
func (s *Service) Verify(ctx context.Context, userID uint64, reference, authority string, callback map[string]string, now time.Time) (Settled, error) {
	return s.settle(ctx, userID, reference, authority, callback, now, true)
}

func (s *Service) settle(ctx context.Context, userID uint64, reference, authority string, callback map[string]string, now time.Time, failOnDecline bool) (Settled, error) {
	var out Settled
	var outcome error // a business failure that still commits (invoice marked failed)
	err := s.inTx(ctx, func(q *store.Queries) error {
		inv, err := q.LockUserInvoiceByReference(ctx, store.LockUserInvoiceByReferenceParams{Reference: reference, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvoiceNotFound
		}
		if err != nil {
			return fmt.Errorf("plus: invoice: %w", err)
		}
		out.Invoice = inv
		switch {
		case inv.Status == InvoicePaid:
			out.AlreadyPaid = true
			return nil
		case inv.Status != InvoicePending:
			return ErrInvoiceClosed
		case !inv.Authority.Valid || inv.Authority.String != authority:
			return ErrAuthorityMismatch
		case s.gateway == nil || inv.Gateway.String != s.gateway.Name():
			return ErrPaymentUnavailable
		}
		res, err := s.gateway.Verify(ctx, VerifyRequest{Authority: authority, AmountRials: inv.TotalRials, Callback: callback})
		if err != nil {
			s.logger.WarnContext(ctx, "plus: gateway verify failed", "gateway", s.gateway.Name(), "reference", reference, "error", err.Error())
			return fmt.Errorf("%w: %w", ErrPaymentUnavailable, err)
		}
		stamp := sql.NullTime{Time: now, Valid: true}
		fail := func(reason error) error {
			if err := q.MarkInvoiceFailed(ctx, store.MarkInvoiceFailedParams{Now: stamp, ID: inv.ID}); err != nil {
				return fmt.Errorf("plus: invoice: %w", err)
			}
			outcome = reason
			return nil
		}
		switch {
		case !res.Paid && !failOnDecline:
			outcome = ErrPaymentFailed // still pending: re-checked later, not failed early
			return nil
		case !res.Paid:
			return fail(ErrPaymentFailed)
		case res.AmountRials != inv.TotalRials:
			s.logger.ErrorContext(ctx, "plus: settled amount differs from invoice", "gateway", s.gateway.Name(),
				"reference", reference, "invoice_total", inv.TotalRials, "settled", res.AmountRials)
			return fail(ErrAmountMismatch)
		}
		err = q.InsertReceipt(ctx, store.InsertReceiptParams{
			InvoiceID: inv.ID, UserID: userID, Gateway: s.gateway.Name(), RefID: res.RefID,
			CardPan:     sql.NullString{String: res.CardPAN, Valid: res.CardPAN != ""},
			AmountRials: res.AmountRials, PaidAt: now, CreatedAt: stamp, UpdatedAt: stamp,
		})
		if isDuplicate(err) {
			s.logger.ErrorContext(ctx, "plus: bank reference already used by another invoice", "gateway", s.gateway.Name(), "reference", reference)
			return fail(ErrPaymentFailed)
		}
		if err != nil {
			return fmt.Errorf("plus: receipt: %w", err)
		}
		var planID uint64
		if inv.PlanID.Valid {
			planID = uint64(inv.PlanID.Int64) //nolint:gosec // G115: ids are positive
		}
		if err := activate(ctx, q, userID, planID, inv.ID, int(inv.DurationMonths), now); err != nil {
			return err
		}
		if err := q.MarkInvoicePaid(ctx, store.MarkInvoicePaidParams{Now: stamp, ID: inv.ID}); err != nil {
			return fmt.Errorf("plus: invoice: %w", err)
		}
		return nil
	})
	if err != nil {
		return Settled{}, err
	}
	if out.Invoice, err = s.q.GetUserInvoiceByReference(ctx, store.GetUserInvoiceByReferenceParams{Reference: reference, UserID: userID}); err != nil {
		return Settled{}, fmt.Errorf("plus: invoice: %w", err)
	}
	return out, outcome
}

// Restore re-checks the user's pending checkouts with the gateway (the «بازیابی خرید» button: paid, but the
// return page never verified). It returns how many were settled now. A still-unpaid checkout stays pending until
// its reservation expires, then is marked failed.
func (s *Service) Restore(ctx context.Context, userID uint64, now time.Time) (int, error) {
	if s.gateway == nil {
		return 0, nil
	}
	pending, err := s.q.ListUserPendingInvoices(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("plus: pending invoices: %w", err)
	}
	n := 0
	for _, inv := range pending {
		res, err := s.settle(ctx, userID, inv.Reference, inv.Authority.String, nil, now, !now.Before(inv.ExpiresAt))
		switch {
		case err == nil && !res.AlreadyPaid:
			n++
		case err == nil, errors.Is(err, ErrPaymentFailed), errors.Is(err, ErrAmountMismatch),
			errors.Is(err, ErrPaymentUnavailable), errors.Is(err, ErrInvoiceClosed), errors.Is(err, ErrAuthorityMismatch):
		default:
			return n, err
		}
	}
	return n, nil
}

// Cancel turns auto-renew off: the subscription stays valid until it ends. Idempotent while a period is running.
func (s *Service) Cancel(ctx context.Context, userID uint64, now time.Time) error {
	n, err := s.q.CancelSubscriptions(ctx, store.CancelSubscriptionsParams{Stamp: sql.NullTime{Time: now, Valid: true}, UserID: userID, Now: now})
	if err != nil {
		return fmt.Errorf("plus: cancel: %w", err)
	}
	if n > 0 {
		return nil
	}
	if _, err := s.q.CurrentSubscription(ctx, store.CurrentSubscriptionParams{UserID: userID, Now: now}); errors.Is(err, sql.ErrNoRows) {
		return ErrNoSubscription
	} else if err != nil {
		return fmt.Errorf("plus: subscription: %w", err)
	}
	return nil
}

// HistoryEntry is one invoice with its plan and receipt.
type HistoryEntry struct {
	Row  store.ListUserInvoicesRow
	Plan *store.PlusPlan
}

// History is the user's latest invoices, newest first.
func (s *Service) History(ctx context.Context, userID uint64) ([]HistoryEntry, error) {
	rows, err := s.q.ListUserInvoices(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("plus: history: %w", err)
	}
	plans := map[int64]*store.PlusPlan{}
	out := make([]HistoryEntry, len(rows))
	for i, r := range rows {
		out[i].Row = r
		if !r.PlusInvoice.PlanID.Valid {
			continue
		}
		id := r.PlusInvoice.PlanID.Int64
		if p, ok := plans[id]; ok {
			out[i].Plan = p
			continue
		}
		p, err := s.q.GetPlan(ctx, uint64(id)) //nolint:gosec // G115: ids are positive
		switch {
		case err == nil:
			plans[id] = &p
			out[i].Plan = &p
		case !errors.Is(err, sql.ErrNoRows):
			return nil, fmt.Errorf("plus: plan: %w", err)
		}
	}
	return out, nil
}

// Plan returns a plan by id (any state), nil when it is gone.
func (s *Service) Plan(ctx context.Context, id sql.NullInt64) (*store.PlusPlan, error) {
	if !id.Valid {
		return nil, nil
	}
	p, err := s.q.GetPlan(ctx, uint64(id.Int64)) //nolint:gosec // G115: ids are positive
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("plus: plan: %w", err)
	}
	return &p, nil
}

// receipt is the verified payment of the user's invoice (nil when none).
func (s *Service) receipt(ctx context.Context, inv store.PlusInvoice, userID uint64) (*Receipt, error) {
	r, err := s.q.GetInvoiceReceipt(ctx, store.GetInvoiceReceiptParams{InvoiceID: inv.ID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("plus: receipt: %w", err)
	}
	return &Receipt{RefID: r.RefID, CardPAN: r.CardPan, Amount: r.AmountRials, PaidAt: r.PaidAt}, nil
}

// ---------------------------------------------------------------------------

// inTx runs fn in a READ COMMITTED transaction: after a row lock (discount code, invoice) is granted, the plain
// reads that follow must see what the previous lock holder committed — under InnoDB's default REPEATABLE READ
// they would read the snapshot taken before the wait and miss it (the discount-limit race test proves this).
func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("plus: begin: %w", err)
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("plus: commit: %w", err)
	}
	return nil
}

var refEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// newReference is an unguessable public invoice id ("RP" + 16 base32 chars, 80 random bits).
func newReference() (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("plus: reference: %w", err)
	}
	return "RP" + refEncoding.EncodeToString(b), nil
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

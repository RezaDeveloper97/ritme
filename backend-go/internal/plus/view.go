package plus

import (
	"database/sql"
	"encoding/json"
	"math"
	"time"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/plus/store"
)

// Currency is the unit of every amount in the API (integer rials).
const Currency = "IRR"

// Localizer picks translatable columns for the request locale (fallback: the default language).
type Localizer struct {
	Locale  string
	Default string
}

func (l Localizer) pick(raw json.RawMessage) string { return i18n.PickString(raw, l.Locale, l.Default) }

func (l Localizer) pickNull(raw sql.Null[json.RawMessage]) any {
	if !raw.Valid {
		return nil
	}
	if s := l.pick(raw.V); s != "" {
		return s
	}
	return nil
}

func iso(t time.Time) jsonx.ISO8601Tehran { return jsonx.ISO8601(t) }

func isoNull(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return iso(t.Time)
}

// daysLeft is the «۷۶ روز مانده» figure: whole days until end, rounded up; 0 once it has passed.
func daysLeft(now, end time.Time) int {
	if !end.After(now) {
		return 0
	}
	return int(math.Ceil(end.Sub(now).Hours() / 24))
}

// PlanJSON is one plan card (nbl_Prem_Plans).
func PlanJSON(p Plan, l Localizer) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", p.ID,
		"code", p.Code,
		"title", l.pick(p.Title),
		"badge", l.pickNull(p.Badge),
		"duration_months", p.DurationMonths,
		"price", p.PriceRials,
		"monthly_price", p.Monthly,
		"savings_percent", p.Savings,
		"is_highlighted", p.IsHighlighted,
	)
}

// PlansJSON is GET /plus/plans.
func PlansJSON(plans []Plan, cfg Configured, l Localizer) *jsonx.OrderedMap {
	list := make([]any, len(plans))
	for i, p := range plans {
		list[i] = PlanJSON(p, l)
	}
	return jsonx.Obj(
		"currency", Currency,
		"vat_rate_bps", cfg.VATRateBps,
		"trial_days", cfg.TrialDays,
		"plans", list,
	)
}

// Configured is the part of the config the views print.
type Configured struct {
	VATRateBps int
	TrialDays  int
}

// planRef is the short plan object on subscriptions and invoices (null when the plan was deleted).
func planRef(p *store.PlusPlan, l Localizer) any {
	if p == nil {
		return nil
	}
	return jsonx.Obj("id", p.ID, "code", p.Code, "title", l.pick(p.Title), "duration_months", p.DurationMonths)
}

// EntitlementJSON is one feature line.
func EntitlementJSON(e Entitlement) *jsonx.OrderedMap {
	var limit, remaining any
	switch {
	case e.Unlimited:
	case e.Limit > 0:
		limit, remaining = e.Limit, e.Remaining
	default:
		limit, remaining = 0, 0
	}
	return jsonx.Obj(
		"key", string(e.Key),
		"allowed", e.Allowed,
		"unlimited", e.Unlimited,
		"limit", limit,
		"used", e.Used,
		"remaining", remaining,
	)
}

func entitlementsJSON(es []Entitlement) []any {
	out := make([]any, len(es))
	for i, e := range es {
		out[i] = EntitlementJSON(e)
	}
	return out
}

// UsageJSON is GET /plus/usage.
func UsageJSON(u Usage, tier Tier) *jsonx.OrderedMap {
	return jsonx.Obj(
		"tier", string(tier),
		"period_start", u.PeriodStart,
		"resets_at", iso(u.ResetsAt),
		"features", entitlementsJSON(u.Entitlements),
	)
}

// StatusJSON is GET /plus/status (and the body of trial/start, verify, cancel, restore).
func StatusJSON(st State, l Localizer) *jsonx.OrderedMap {
	var sub, trial any
	if s := st.Subscription; s != nil {
		sub = jsonx.Obj(
			"id", s.ID,
			"plan", planRef(st.SubscriptionPlan, l),
			"status", s.Status,
			"source", s.Source,
			"starts_at", iso(s.StartsAt),
			"ends_at", iso(s.EndsAt),
			"days_left", daysLeft(st.Now, s.EndsAt),
			"auto_renew", s.AutoRenew,
			"canceled_at", isoNull(s.CanceledAt),
		)
	}
	if t := st.Trial; t != nil {
		trial = jsonx.Obj(
			"started_at", iso(t.StartedAt),
			"ends_at", iso(t.EndsAt),
			"is_active", trialRunning(*t, st.Now),
			"days_left", daysLeft(st.Now, t.EndsAt),
		)
	}
	return jsonx.Obj(
		"tier", string(st.Tier),
		"is_plus", st.Tier != TierFree,
		"subscription", sub,
		"trial", trial,
		"trial_available", st.TrialAvailable,
		"period_start", st.Usage.PeriodStart,
		"resets_at", iso(st.Usage.ResetsAt),
		"entitlements", entitlementsJSON(st.Usage.Entitlements),
		"trial_offer", TrialOfferJSON(st.Offer, l),
	)
}

// secondsLeft is the countdown in whole seconds (0 once end has passed).
func secondsLeft(now, end time.Time) int64 {
	if !end.After(now) {
		return 0
	}
	return int64(end.Sub(now) / time.Second)
}

// countdownJSON is the banner's «۰۵ روز ۱۴ ساعت ۲۲ دقیقه» split of seconds.
func countdownJSON(secs int64) *jsonx.OrderedMap {
	return jsonx.Obj("days", secs/86400, "hours", secs%86400/3600, "minutes", secs%3600/60)
}

// OfferPlanJSON is a plan card with its trial-offer prices (offer fields null when no offer runs).
func OfferPlanJSON(p Plan, percent int, l Localizer) *jsonx.OrderedMap {
	var price, monthly any
	if percent > 0 {
		price, monthly = OfferPrice(p.PriceRials, percent), OfferPrice(p.Monthly, percent)
	}
	return PlanJSON(p, l).Set("offer_price", price).Set("offer_monthly_price", monthly)
}

// TrialOfferJSON is the trial banner (nbl_Prem_TrialHome) — also /plus/status.trial_offer and the trial sheet's
// offer. null when no offer runs. Money in rials (prices before VAT, like the plan cards).
func TrialOfferJSON(o *Offer, l Localizer) any {
	if o == nil {
		return nil
	}
	secs := secondsLeft(o.Now, o.EndsAt)
	return jsonx.Obj(
		"percent", o.Percent,
		"ends_at", iso(o.EndsAt),
		"seconds_left", secs,
		"days_left", daysLeft(o.Now, o.EndsAt),
		"countdown", countdownJSON(secs),
		"currency", Currency,
		"plan", OfferPlanJSON(o.Featured, o.Percent, l),
	)
}

// TrialSheetJSON is GET /plus/trial (nbl_Prem_TrialSheet): status, the offer, every plan with its offer price and
// what the user used of Plus since the trial began.
func TrialSheetJSON(sh TrialSheet, l Localizer) *jsonx.OrderedMap {
	var trial any
	if t := sh.State.Trial; t != nil {
		trial = jsonx.Obj(
			"started_at", iso(t.StartedAt),
			"ends_at", iso(t.EndsAt),
			"is_active", trialRunning(*t, sh.State.Now),
			"days_left", daysLeft(sh.State.Now, t.EndsAt),
			"seconds_left", secondsLeft(sh.State.Now, t.EndsAt),
		)
	}
	percent := 0
	if sh.Offer != nil {
		percent = sh.Offer.Percent
	}
	plans := make([]any, len(sh.Plans))
	for i, p := range sh.Plans {
		plans[i] = OfferPlanJSON(p, percent, l)
	}
	features := make([]any, len(sh.Used))
	for i, u := range sh.Used {
		q := u.Definition.Plus
		var limit any
		if q.Limit > 0 {
			limit = q.Limit
		}
		features[i] = jsonx.Obj("key", string(u.Definition.Key), "used", u.Used, "unlimited", q.Enabled && q.Limit == 0, "plus_limit", limit)
	}
	return jsonx.Obj(
		"tier", string(sh.State.Tier),
		"trial", trial,
		"trial_available", sh.State.TrialAvailable,
		"offer", TrialOfferJSON(sh.Offer, l),
		"currency", Currency,
		"plans", plans,
		"usage", jsonx.Obj("since", sh.UsageSince, "features", features),
	)
}

// QuoteJSON is a checkout preview.
func QuoteJSON(q Quote, l Localizer) *jsonx.OrderedMap {
	var code any
	if q.DiscountCode != "" {
		code = q.DiscountCode
	}
	p := q.Plan
	return jsonx.Obj(
		"plan", planRef(&p, l),
		"currency", Currency,
		"subtotal", q.Amounts.Subtotal,
		"discount", q.Amounts.Discount,
		"discount_code", code,
		"vat_rate_bps", q.Amounts.VATRateBps,
		"vat", q.Amounts.VAT,
		"total", q.Amounts.Total,
		"discount_source", quoteSource(q),
		"trial_offer", appliedOfferJSON(q.TrialOffer, q.Plan),
	)
}

// Discount sources («discount_source»): a discount code or the trial offer (null = no discount).
const (
	SourceCode       = "code"
	SourceTrialOffer = "trial_offer"
)

func quoteSource(q Quote) any {
	switch {
	case q.DiscountCode != "":
		return SourceCode
	case q.TrialOffer != nil:
		return SourceTrialOffer
	}
	return nil
}

// invoiceSource derives where an invoice's discount came from: the only discounts are codes (kept on the
// invoice) and the trial offer.
func invoiceSource(inv store.PlusInvoice) any {
	switch {
	case inv.DiscountCode.Valid:
		return SourceCode
	case inv.DiscountRials > 0:
		return SourceTrialOffer
	}
	return nil
}

func appliedOfferJSON(o *AppliedOffer, p store.PlusPlan) any {
	if o == nil {
		return nil
	}
	return jsonx.Obj("percent", o.Percent, "ends_at", iso(o.EndsAt), "offer_price", OfferPrice(p.PriceRials, o.Percent))
}

// Receipt is the verified-payment part of an invoice.
type Receipt struct {
	RefID   string
	CardPAN sql.NullString
	Amount  uint64
	PaidAt  time.Time
}

// InvoiceJSON is one invoice (checkout, verify, history).
func InvoiceJSON(inv store.PlusInvoice, plan *store.PlusPlan, rc *Receipt, l Localizer) *jsonx.OrderedMap {
	var receipt any
	if rc != nil {
		var pan any
		if rc.CardPAN.Valid {
			pan = rc.CardPAN.String
		}
		receipt = jsonx.Obj("ref_id", rc.RefID, "card_pan", pan, "amount", rc.Amount, "paid_at", iso(rc.PaidAt))
	}
	var code, gateway any
	if inv.DiscountCode.Valid {
		code = inv.DiscountCode.String
	}
	if inv.Gateway.Valid {
		gateway = inv.Gateway.String
	}
	return jsonx.Obj(
		"reference", inv.Reference,
		"status", inv.Status,
		"plan", planRef(plan, l),
		"duration_months", inv.DurationMonths,
		"currency", inv.Currency,
		"subtotal", inv.SubtotalRials,
		"discount", inv.DiscountRials,
		"discount_code", code,
		"vat_rate_bps", inv.VatRateBps,
		"vat", inv.VatRials,
		"total", inv.TotalRials,
		"gateway", gateway,
		"created_at", isoNull(inv.CreatedAt),
		"expires_at", iso(inv.ExpiresAt),
		"paid_at", isoNull(inv.PaidAt),
		"receipt", receipt,
		"discount_source", invoiceSource(inv),
	)
}

// PaymentJSON is where to send the user (null when nothing is charged).
func PaymentJSON(p *PaymentSession, gateway string) any {
	if p == nil {
		return nil
	}
	return jsonx.Obj("gateway", gateway, "authority", p.Authority, "redirect_url", p.RedirectURL)
}

// HistoryJSON is GET /plus/history.
func HistoryJSON(entries []HistoryEntry, l Localizer) []any {
	out := make([]any, len(entries))
	for i, e := range entries {
		var rc *Receipt
		if r := e.Row; r.ReceiptRefID.Valid {
			rc = &Receipt{RefID: r.ReceiptRefID.String, CardPAN: r.ReceiptCardPan, Amount: uint64(max(r.ReceiptAmountRials.Int64, 0)), PaidAt: r.ReceiptPaidAt.Time}
		}
		out[i] = InvoiceJSON(e.Row.PlusInvoice, e.Plan, rc, l)
	}
	return out
}

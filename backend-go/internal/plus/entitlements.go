// Package plus is the Ritme Plus subscription domain (B-N2-04): plans, the 7-day trial, checkout through a payment
// port, subscriptions, discount codes, invoices/receipts and the entitlement service every Plus-gated feature asks.
//
// Money is integer rials (IRR) everywhere; clients show toman (rials / 10).
package plus

// Key names one Plus entitlement. Keys are stable API identifiers (clients, gating middleware, usage counters);
// a new Plus feature adds a constant and a Definition below — nothing else in the package changes.
type Key string

// The Plus entitlements (bloom N2–N7; roadmap canvas tasks gate their AI features through the same keys).
const (
	DeepAnalysis       Key = "plus.deep_analysis"       // analysis tab: patterns, correlations, mood by phase (B-N3-07)
	VoiceLog           Key = "plus.voice_log"           // voice logging (B-N3-05)
	LabAI              Key = "plus.lab_ai"              // AI lab-result analysis: 10 per month, free 1 per month (B-N6, A-03)
	AssistantUnlimited Key = "plus.assistant_unlimited" // health assistant: Plus unlimited with a fair-use cap (B-N7, A-04)
	PDFShare           Key = "plus.pdf_share"           // doctor report PDF + 7-day share link (B-N6-04)
	VisitDiscount      Key = "plus.visit_discount"      // discount on online doctor visits (B-N7)
	DocAI              Key = "plus.doc_ai"              // AI reading of record documents, 20 per month (CB-REC-02)
	AnalysisLongRange  Key = "plus.analysis_long_range" // analysis ranges of a year and more: 1y, all (T-M10-29, D-178)
)

// Tier is what the user currently has.
type Tier string

// Tiers. A running trial grants the Plus quotas.
const (
	TierFree  Tier = "free"
	TierTrial Tier = "trial"
	TierPlus  Tier = "plus"
)

// Quota is one tier's allowance for a feature: Enabled false = locked; Limit 0 = unlimited, otherwise uses per
// calendar month (Tehran). FairUse is a cap on an unlimited allowance (A-04): the feature is still shown as
// unlimited, but the uses of one month stop at FairUse (0 = no cap).
type Quota struct {
	Enabled bool
	Limit   int
	FairUse int
}

// Cap is the most uses of one month: Limit, else FairUse; 0 = no cap.
func (q Quota) Cap() int {
	if q.Limit > 0 {
		return q.Limit
	}
	return q.FairUse
}

var (
	unlimited = Quota{Enabled: true}
	locked    = Quota{}
)

// Definition is one entitlement and its allowance per tier.
type Definition struct {
	Key  Key
	Plus Quota // Plus subscribers and running trials
	Free Quota
}

// Code defaults of the admin-tunable allowances (plus_settings, settings.go; D-178).
const (
	// DefaultLabAIFreeLimit is the free lab-AI allowance per month (A-03: «۱ بار در ماه»).
	DefaultLabAIFreeLimit = 1
	// DefaultAssistantFairUse is the Plus assistant's fair-use cap per month (A-04).
	DefaultAssistantFairUse = 300
)

// definitions is the single source of the entitlement rules, in display order (the trial sheet's usage lines).
// Free quotas: the assistant (Plus is «نامحدود» with a fair-use cap) and the lab AI (SPEC §4.3) have a free
// allowance; every other Plus row is locked for free users on nbl_Prem_Paywall. The lab-AI free limit and the
// assistant's fair-use cap are the code defaults of admin settings (Service.Definition applies them).
//
// Never add an emergency, safety-alert, medication-reminder, EPDS or core-logging feature here: those are free
// for everyone (SPEC §4.3, internal/http/paywall_guard_int_test.go).
var definitions = []Definition{
	{Key: DeepAnalysis, Plus: unlimited, Free: locked},
	{Key: LabAI, Plus: Quota{Enabled: true, Limit: 10}, Free: Quota{Enabled: true, Limit: DefaultLabAIFreeLimit}},
	{Key: AssistantUnlimited, Plus: Quota{Enabled: true, FairUse: DefaultAssistantFairUse}, Free: Quota{Enabled: true, Limit: 5}},
	{Key: PDFShare, Plus: unlimited, Free: locked},
	{Key: VisitDiscount, Plus: unlimited, Free: locked},
	{Key: VoiceLog, Plus: unlimited, Free: locked},
	{Key: DocAI, Plus: Quota{Enabled: true, Limit: 20}, Free: locked},
	{Key: AnalysisLongRange, Plus: unlimited, Free: locked},
}

// Definitions returns a copy of the entitlement rules in display order.
func Definitions() []Definition { return append([]Definition(nil), definitions...) }

// Lookup returns the definition of key.
func Lookup(key Key) (Definition, bool) {
	for _, d := range definitions {
		if d.Key == key {
			return d, true
		}
	}
	return Definition{}, false
}

// QuotaFor is the allowance of tier.
func (d Definition) QuotaFor(t Tier) Quota {
	if t == TierPlus || t == TierTrial {
		return d.Plus
	}
	return d.Free
}

// Entitlement is one resolved feature for a user this month.
type Entitlement struct {
	Key       Key
	Allowed   bool // may use it now (enabled and, when limited or capped, under the limit)
	Unlimited bool
	Limit     int // 0 when locked or unlimited without a cap; the fair-use cap when FairUse
	Used      int
	Remaining int // 0 when locked or unlimited without a cap
	// FairUse: an unlimited allowance with a fair-use cap (Limit). The clients still show «نامحدود»; at the cap the
	// feature answers politely instead of with the paywall (the assistant's in-chat message, A-04).
	FairUse bool
}

// AtFairUseCap reports whether e is an unlimited allowance whose fair-use cap is spent this month.
func (e Entitlement) AtFairUseCap() bool { return e.FairUse && !e.Allowed }

// ResolveOne applies tier's allowance to this month's usage of one feature.
func ResolveOne(t Tier, d Definition, used int) Entitlement {
	q := d.QuotaFor(t)
	e := Entitlement{Key: d.Key, Used: used}
	switch {
	case !q.Enabled:
	case q.Limit == 0 && q.FairUse > 0:
		e.Unlimited, e.FairUse, e.Limit = true, true, q.FairUse
		e.Remaining = max(0, q.FairUse-used)
		e.Allowed = e.Remaining > 0
	case q.Limit == 0:
		e.Allowed, e.Unlimited = true, true
	default:
		e.Limit = q.Limit
		e.Remaining = max(0, q.Limit-used)
		e.Allowed = e.Remaining > 0
	}
	return e
}

// Resolve is every entitlement of tier for this month's usage (missing keys = 0 uses), in display order, under
// the code defaults. This is the one place entitlements are decided.
func Resolve(t Tier, used map[Key]int) []Entitlement { return ResolveAll(definitions, t, used) }

// ResolveAll is Resolve over defs (the definitions with the admin settings applied, Service.Definitions).
func ResolveAll(defs []Definition, t Tier, used map[Key]int) []Entitlement {
	out := make([]Entitlement, 0, len(defs))
	for _, d := range defs {
		out = append(out, ResolveOne(t, d, used[d.Key]))
	}
	return out
}

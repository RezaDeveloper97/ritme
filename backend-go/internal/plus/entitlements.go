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
	LabAI              Key = "plus.lab_ai"              // AI lab-result analysis, 10 per month (B-N6)
	AssistantUnlimited Key = "plus.assistant_unlimited" // health assistant without a message cap (B-N7)
	PDFShare           Key = "plus.pdf_share"           // doctor report PDF + 7-day share link (B-N6-04)
	VisitDiscount      Key = "plus.visit_discount"      // discount on online doctor visits (B-N7)
	DocAI              Key = "plus.doc_ai"              // AI reading of record documents, 20 per month (CB-REC-02)
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
// calendar month (Tehran).
type Quota struct {
	Enabled bool
	Limit   int
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

// definitions is the single source of the entitlement rules, in display order (the trial sheet's usage lines).
// Free quotas: only the assistant has a designed free allowance (Plus is «نامحدود»); every other Plus row is locked
// for free users on nbl_Prem_Paywall.
var definitions = []Definition{
	{Key: DeepAnalysis, Plus: unlimited, Free: locked},
	{Key: LabAI, Plus: Quota{Enabled: true, Limit: 10}, Free: locked},
	{Key: AssistantUnlimited, Plus: unlimited, Free: Quota{Enabled: true, Limit: 5}},
	{Key: PDFShare, Plus: unlimited, Free: locked},
	{Key: VisitDiscount, Plus: unlimited, Free: locked},
	{Key: VoiceLog, Plus: unlimited, Free: locked},
	{Key: DocAI, Plus: Quota{Enabled: true, Limit: 20}, Free: locked},
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
	Allowed   bool // may use it now (enabled and, when limited, under the limit)
	Unlimited bool
	Limit     int // 0 when unlimited or locked
	Used      int
	Remaining int // 0 when unlimited or locked
}

// ResolveOne applies tier's allowance to this month's usage of one feature.
func ResolveOne(t Tier, d Definition, used int) Entitlement {
	q := d.QuotaFor(t)
	e := Entitlement{Key: d.Key, Used: used}
	switch {
	case !q.Enabled:
	case q.Limit == 0:
		e.Allowed, e.Unlimited = true, true
	default:
		e.Limit = q.Limit
		e.Remaining = max(0, q.Limit-used)
		e.Allowed = e.Remaining > 0
	}
	return e
}

// Resolve is every entitlement of tier for this month's usage (missing keys = 0 uses), in display order.
// This is the one place entitlements are decided.
func Resolve(t Tier, used map[Key]int) []Entitlement {
	out := make([]Entitlement, 0, len(definitions))
	for _, d := range definitions {
		out = append(out, ResolveOne(t, d, used[d.Key]))
	}
	return out
}

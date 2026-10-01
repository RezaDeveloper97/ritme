package plus

import (
	"encoding/json"
	"time"

	"github.com/ritme/backend-go/internal/plus/store"
)

// Discount kinds.
const (
	DiscountPercent = "percent" // value 1–100
	DiscountAmount  = "amount"  // value in rials, capped at the plan price
)

// Discount rejection reasons (error_code "discount_<reason>").
const (
	DiscountInvalid    = "invalid"     // unknown or inactive code
	DiscountNotStarted = "not_started" // before starts_at
	DiscountExpired    = "expired"     // at or after expires_at
	DiscountPlan       = "plan"        // not valid for this plan
	DiscountExhausted  = "exhausted"   // max_redemptions reached
	DiscountUsed       = "used"        // the user's per_user_limit reached
)

// DiscountError rejects a discount code at checkout.
type DiscountError struct{ Reason string }

func (e *DiscountError) Error() string { return "plus: discount " + e.Reason }

// Amounts is a priced checkout in rials. VAT applies to the price after discount and is rounded down (in the
// buyer's favour); the rate is the config snapshot in basis points.
type Amounts struct {
	Subtotal   uint64
	Discount   uint64
	VATRateBps uint32
	VAT        uint64
	Total      uint64
}

// Price computes the amounts of subtotal with an optional discount (kind/value as stored) and the VAT rate.
func Price(subtotal uint64, kind string, value uint64, vatBps uint32) Amounts {
	a := Amounts{Subtotal: subtotal, VATRateBps: vatBps}
	switch kind {
	case DiscountPercent:
		a.Discount = subtotal * min(value, 100) / 100
	case DiscountAmount:
		a.Discount = min(value, subtotal)
	}
	net := subtotal - a.Discount
	a.VAT = net * uint64(vatBps) / 10000
	a.Total = net + a.VAT
	return a
}

// CheckDiscount decides whether code may be redeemed on planID at now. total and mine are the code's redemptions
// overall and by this user (paid invoices + pending ones still holding a reservation), counted under the code's row
// lock, so concurrent checkouts cannot overrun the limits.
func CheckDiscount(code store.PlusDiscountCode, planID uint64, now time.Time, total, mine int64) error {
	switch {
	case !code.IsActive || (code.Kind != DiscountPercent && code.Kind != DiscountAmount) || code.Value == 0:
		return &DiscountError{Reason: DiscountInvalid}
	case code.StartsAt.Valid && now.Before(code.StartsAt.Time):
		return &DiscountError{Reason: DiscountNotStarted}
	case code.ExpiresAt.Valid && !now.Before(code.ExpiresAt.Time):
		return &DiscountError{Reason: DiscountExpired}
	case !planAllowed(code, planID):
		return &DiscountError{Reason: DiscountPlan}
	case code.MaxRedemptions.Valid && total >= int64(code.MaxRedemptions.Int32):
		return &DiscountError{Reason: DiscountExhausted}
	case code.PerUserLimit.Valid && mine >= int64(code.PerUserLimit.Int32):
		return &DiscountError{Reason: DiscountUsed}
	}
	return nil
}

// planAllowed: plan_ids NULL (or not a list) = every plan.
func planAllowed(code store.PlusDiscountCode, planID uint64) bool {
	if !code.PlanIds.Valid {
		return true
	}
	var ids []uint64
	if err := json.Unmarshal(code.PlanIds.V, &ids); err != nil {
		return false // an unreadable restriction never widens a code
	}
	for _, id := range ids {
		if id == planID {
			return true
		}
	}
	return false
}

// MonthlyPrice is the «تومان / ماه» figure of a plan in rials: the admin override, else price / months (floor).
func MonthlyPrice(p store.PlusPlan) uint64 {
	if p.MonthlyDisplayRials.Valid && p.MonthlyDisplayRials.Int64 > 0 {
		return uint64(p.MonthlyDisplayRials.Int64)
	}
	if p.DurationMonths == 0 {
		return p.PriceRials
	}
	return p.PriceRials / uint64(p.DurationMonths)
}

// SavingsPercent is the «٪۲۰ صرفه‌جویی» figure: how much cheaper per month than the base plan (the active plan
// with the shortest duration), floored; 0 for the base plan itself or when it is not cheaper.
func SavingsPercent(p store.PlusPlan, base uint64) int {
	m := MonthlyPrice(p)
	if base == 0 || m >= base {
		return 0
	}
	return int((base - m) * 100 / base) //nolint:gosec // G115: a percentage, 0–100
}

// BaseMonthly is the per-month price of the shortest active plan (ties: the first in display order).
func BaseMonthly(plans []store.PlusPlan) uint64 {
	var best *store.PlusPlan
	for i := range plans {
		if best == nil || plans[i].DurationMonths < best.DurationMonths {
			best = &plans[i]
		}
	}
	if best == nil {
		return 0
	}
	return MonthlyPrice(*best)
}

// AddMonths adds n calendar months without overflowing into the next month (Jan 31 + 1 = Feb 28/29).
func AddMonths(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m+time.Month(n), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	last := first.AddDate(0, 1, -1).Day()
	return first.AddDate(0, 0, min(d, last)-1)
}

package credits_test

// T-M12-19: the ledger at its edges — the exact second a period ends, a refund that arrives after its period
// expired, and spends racing refunds on the same bucket. Every amount is read from the seeded / fixture rows
// (admin data), never assumed.
//
// Rules (docs/ai-credits.md § Rules): a period's credits are usable only while it runs (2), extra credits never
// expire (3) but need a running subscription (1), period credits go first (4), a failed call is refunded (6).
// A period is the half-open interval [starts_at, ends_at): at ends_at it no longer runs — the same edge as
// GET /plus/status (internal/plus TestPurchases_StackAndCancel: 09:59:59 plus, 10:00:00 free).

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai/credits"
)

// planCredits is a plan's ai_credits (admin data).
func (e *env) planCredits(t *testing.T, planID int) uint32 {
	t.Helper()
	n := e.count(t, `SELECT ai_credits FROM plus_plans WHERE id = ?`, planID)
	require.Positive(t, n, "the seeded plan carries credits")
	return uint32(n) //nolint:gosec // positive
}

// customPlan is a fixture plan carrying n credits.
func (e *env) customPlan(t *testing.T, code string, n int) int {
	t.Helper()
	id, err := e.exec(t, `INSERT INTO plus_plans (code, title, duration_months, price_rials, ai_credits, is_active, sort_order, created_at, updated_at)
		VALUES (?, '{"fa":"آزمون"}', 1, 1000, ?, 0, 99, NOW(), NOW())`, code, n).LastInsertId()
	require.NoError(t, err)
	return int(id)
}

func TestBoundary_PeriodEndSecond(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001201")
	start, end := t0, t0.Add(day(30))
	e.subscription(t, uid, plan1m, start, end)
	e.extra(t, uid, 10)
	total := e.planCredits(t, plan1m)

	cases := []struct {
		name   string
		at     time.Time
		active bool
	}{
		{"start - 1s: not started", start.Add(-time.Second), false},
		{"start: running", start, true},
		{"end - 1s: last running second", end.Add(-time.Second), true},
		{"end - 1ms: still running", end.Add(-time.Millisecond), true},
		{"end: over", end, false},
		{"end + 1s: over", end.Add(time.Second), false},
	}
	spent := uint32(0)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := e.balance(t, uid, c.at)
			assert.Equal(t, c.active, b.Active)
			assert.Equal(t, uint32(10), b.Extra, "extra credits are always reported (locked or not)")
			r, err := e.svc.Reserve(ctx, uid, "assistant", 1, c.at)
			if !c.active {
				require.ErrorIs(t, err, credits.ErrNoSubscription, "even with extra credits")
				assert.Zero(t, b.Spendable())
				assert.Zero(t, b.PlanTotal)
				return
			}
			require.NoError(t, err)
			spent++
			assert.Equal(t, end, r.Balance.PeriodEndsAt)
			assert.Equal(t, total-spent, r.Balance.PlanRemaining, "the period pays first")
			assert.Equal(t, uint32(10), r.Balance.Extra)
		})
	}
	assert.Equal(t, int(spent), e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ?`, uid), "the refusals recorded nothing")
	assert.Equal(t, 10, e.count(t, `SELECT remaining FROM ai_credit_grants WHERE user_id = ? AND kind = 'extra'`, uid))

	// The trial ends on the same edge.
	tu := e.user(t, "09120001202")
	e.trial(t, tu, start, start.Add(day(7)))
	_, err := e.svc.Reserve(ctx, tu, "assistant", 1, start.Add(day(7)).Add(-time.Second))
	require.NoError(t, err)
	_, err = e.svc.Reserve(ctx, tu, "assistant", 1, start.Add(day(7)))
	require.ErrorIs(t, err, credits.ErrNoSubscription)
}

func TestBoundary_BackToBackPeriodsHandOverAtTheSecond(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001203")
	end := t0.Add(day(30))
	e.subscription(t, uid, plan1m, t0, end)
	e.subscription(t, uid, plan3m, end, end.Add(day(90)))
	first, second := e.planCredits(t, plan1m), e.planCredits(t, plan3m)

	// The last second of the first period spends the first period's bucket only.
	r, err := e.svc.Reserve(ctx, uid, "assistant", 1, end.Add(-time.Second))
	require.NoError(t, err)
	assert.Equal(t, credits.Balance{Active: true, Period: credits.PeriodPlus, PeriodEndsAt: end, PlanTotal: first, PlanRemaining: first - 1}, r.Balance)

	// At end the next period runs alone, from its own plan's amount: one bucket at a time, never both.
	b := e.balance(t, uid, end)
	assert.Equal(t, credits.Balance{Active: true, Period: credits.PeriodPlus, PeriodEndsAt: end.Add(day(90)), PlanTotal: second, PlanRemaining: second}, b)
	r, err = e.svc.Reserve(ctx, uid, "assistant", 1, end)
	require.NoError(t, err)
	assert.Equal(t, second-1, r.Balance.PlanRemaining)
	assert.Equal(t, int(first-1), e.count(t, `SELECT g.remaining FROM ai_credit_grants g JOIN plus_subscriptions s ON s.id = g.subscription_id
		WHERE g.user_id = ? AND s.plan_id = ?`, uid, plan1m), "the old leftover is not touched and not carried")
}

func TestRefund_AfterThePeriodExpired(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001204")
	end := t0.Add(day(30))
	e.subscription(t, uid, plan1m, t0, end)
	e.extra(t, uid, 10)
	total := e.planCredits(t, plan1m)

	// Leave 2 period credits, then a 5-credit use in the last second: 2 from the period, 3 from the extra.
	_, err := e.svc.Reserve(ctx, uid, "assistant", total-2, t0)
	require.NoError(t, err)
	r, err := e.svc.Reserve(ctx, uid, "lab_analysis", 5, end.Add(-time.Second))
	require.NoError(t, err)
	assert.Equal(t, uint32(0), r.Balance.PlanRemaining)
	assert.Equal(t, uint32(7), r.Balance.Extra)

	// The next period starts at end; the failed job's refund arrives a minute later.
	e.subscription(t, uid, plan3m, end, end.Add(day(90)))
	next := e.planCredits(t, plan3m)
	at := end.Add(time.Minute)
	require.NoError(t, e.svc.RefundAt(ctx, uid, "lab_analysis", r.At, at))

	// Each part goes back to its own bucket: the extra part is spendable again (never expires), the period part
	// lands in the period that has ended and is gone with it — never in the new period's bucket.
	b := e.balance(t, uid, at)
	assert.Equal(t, uint32(10), b.Extra)
	assert.Equal(t, next, b.PlanTotal)
	assert.Equal(t, next, b.PlanRemaining, "the new period gains nothing from the old one's refund")
	assert.Equal(t, next+10, b.Spendable())
	assert.Equal(t, 2, e.count(t, `SELECT g.remaining FROM ai_credit_grants g JOIN plus_subscriptions s ON s.id = g.subscription_id
		WHERE g.user_id = ? AND s.plan_id = ?`, uid, plan1m), "returned to the expired bucket")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE id = ? AND status = 'refunded'`, r.ID))

	// Twice is once, even after the expiry.
	require.NoError(t, e.svc.Refund(ctx, r, at.Add(time.Minute)))
	assert.Equal(t, b, e.balance(t, uid, at.Add(time.Minute)))

	// Without a next period the extra part is back but locked (rule 1), the period part is gone.
	lone := e.user(t, "09120001205")
	e.subscription(t, lone, plan1m, t0, end)
	e.extra(t, lone, 4)
	_, err = e.svc.Reserve(ctx, lone, "assistant", total-1, t0)
	require.NoError(t, err)
	r, err = e.svc.Reserve(ctx, lone, "lab_analysis", 3, end.Add(-time.Second))
	require.NoError(t, err)
	require.NoError(t, e.svc.Refund(ctx, r, end.Add(time.Hour)))
	b = e.balance(t, lone, end.Add(time.Hour))
	assert.False(t, b.Active)
	assert.Equal(t, uint32(4), b.Extra)
	assert.Zero(t, b.Spendable())
	assert.Zero(t, b.PlanRemaining)
}

// Spends and refunds race on the same buckets: the balance never goes negative and every credit is accounted for
// (remaining + counted spends = granted), however the goroutines interleave.
func TestConcurrentSpendAndRefund_ConservesCredits(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001206")
	planID := e.customPlan(t, "race_refund", 6)
	planAmount := e.planCredits(t, planID)
	e.subscription(t, uid, planID, t0.Add(-time.Hour), t0.Add(day(30)))
	e.extra(t, uid, 4)
	granted := int(planAmount) + 4

	// Seed reservations that the racing refunds give back (two goroutines per reservation: refunding twice must
	// still be once).
	seeded := make([]*credits.Reservation, 3)
	for i := range seeded {
		r, err := e.svc.Reserve(ctx, uid, "assistant", 1, t0)
		require.NoError(t, err)
		seeded[i] = r
	}

	const spenders = 24
	var wg sync.WaitGroup
	spendErr := make([]error, spenders)
	for i := range spendErr {
		wg.Go(func() {
			_, spendErr[i] = e.svc.Reserve(context.Background(), uid, "lab_analysis", 1, t0.Add(time.Minute))
		})
	}
	refundErr := make([]error, 2*len(seeded))
	for i := range refundErr {
		wg.Go(func() {
			refundErr[i] = e.svc.Refund(context.Background(), seeded[i/2], t0.Add(time.Minute))
		})
	}
	wg.Wait()
	for _, err := range refundErr {
		require.NoError(t, err)
	}
	ok := 0
	for _, err := range spendErr {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, credits.ErrInsufficient):
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}

	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM ai_credit_grants WHERE user_id = ? AND remaining < 0`, uid))
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM ai_credit_grants WHERE user_id = ? AND remaining > amount`, uid))
	remaining := e.count(t, `SELECT COALESCE(SUM(remaining), 0) FROM ai_credit_grants WHERE user_id = ?`, uid)
	counted := e.count(t, `SELECT COALESCE(SUM(credits), 0) FROM ai_credit_spends WHERE user_id = ? AND status = 'spent'`, uid)
	assert.Equal(t, granted, remaining+counted, "every credit is either left or counted once")
	assert.Equal(t, len(seeded), e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ? AND status = 'refunded'`, uid))
	assert.Equal(t, ok, e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ? AND feature = 'lab_analysis' AND status = 'spent'`, uid))
	// The refunds landed before or after the spends, so the spenders got between granted-seeded and granted.
	assert.GreaterOrEqual(t, ok, granted-len(seeded))
	assert.LessOrEqual(t, ok, granted)
	assert.Equal(t, uint32(remaining), e.balance(t, uid, t0.Add(time.Minute)).Spendable()) //nolint:gosec // small

	// Every reservation's allocations add up to its credits and point at this user's buckets.
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ? AND plan_credits + extra_credits <> credits`, uid))
}

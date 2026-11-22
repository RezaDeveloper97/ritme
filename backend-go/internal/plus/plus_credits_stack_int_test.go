package plus_test

// T-M12-19: buying a plan while one is running. The new period is queued after the running one ("buying again
// extends, never overlaps") and carries its own plan's credits from its own start (docs/ai-credits.md rule 2):
// never both periods' credits at once, never a bucket twice — replayed or concurrent verifies included.

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/plus"
)

func (e *env) planCredits(t *testing.T, planID int) uint32 {
	t.Helper()
	n := e.count(t, `SELECT ai_credits FROM plus_plans WHERE id = ?`, planID)
	require.Positive(t, n, "the seeded plan carries credits")
	return uint32(n) //nolint:gosec // positive
}

func TestStackedPurchase_PeriodCreditsPerPeriodNeverTwice(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	ledger := credits.NewService(e.db)
	ctx := context.Background()
	uid, tok := e.user(t, "09120001301")
	first, second := e.planCredits(t, plan1m), e.planCredits(t, plan3m)

	ref, authority, _ := e.checkout(t, now, tok, `{"plan_id":1}`)
	r := e.do(t, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref, authority, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	res, err := ledger.Reserve(ctx, uid, "assistant", 1, tNow.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, first-1, res.Balance.PlanRemaining)

	// A second plan bought while the first runs: queued, its credits not granted yet.
	mid := "2026-10-01T09:00:00+03:30"
	ref2, auth2, _ := e.checkout(t, mid, tok, `{"plan_id":2}`)
	r = e.doAt(t, mid, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref2, auth2, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var end1 time.Time
	require.NoError(t, e.db.QueryRow(`SELECT ends_at FROM plus_subscriptions WHERE user_id = ? AND plan_id = ?`, uid, plan1m).Scan(&end1))
	end1 = end1.In(tehran)

	b, err := ledger.Balance(ctx, uid, mustTime(mid))
	require.NoError(t, err)
	assert.Equal(t, first, b.PlanTotal, "only the running period's credits, not both plans'")
	assert.Equal(t, first-1, b.PlanRemaining)
	assert.Equal(t, end1, b.PeriodEndsAt)

	// A replayed verify adds no period and no credits.
	r = e.doAt(t, mid, http.MethodPost, "/api/v1/plus/verify", tok, "en", verifyBody(ref2, auth2, "OK"))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.data()["already_verified"])
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))

	// Last second of the first period: still the first plan's bucket.
	b, err = ledger.Balance(ctx, uid, end1.Add(-time.Second))
	require.NoError(t, err)
	assert.Equal(t, first, b.PlanTotal)
	// From the queued period's first second: the second plan's amount, the first's leftover gone.
	b, err = ledger.Balance(ctx, uid, end1)
	require.NoError(t, err)
	assert.Equal(t, second, b.PlanTotal)
	assert.Equal(t, second, b.PlanRemaining)
	for range 3 {
		_, err = ledger.Balance(ctx, uid, end1.Add(time.Minute))
		require.NoError(t, err)
	}
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM ai_credit_grants WHERE user_id = ? AND kind = 'plan'`, uid), "one bucket per period")
	assert.Equal(t, int(first+second), e.count(t, `SELECT SUM(amount) FROM ai_credit_grants WHERE user_id = ? AND kind = 'plan'`, uid))
}

// Two different invoices of one user settled at the same moment (two tabs, or «بازیابی خرید» racing the return
// page) must still queue one period after the other: overlapping periods would lose paid time and run two plans'
// credits at once.
func TestStackedPurchase_ConcurrentSettlesNeverOverlap(t *testing.T) {
	e := setup(t, plus.FakeGateway{})
	ledger := credits.NewService(e.db)
	ctx := context.Background()
	uid, tok := e.user(t, "09120001302")
	const n = 4
	refs, auths := make([]string, n), make([]string, n)
	for i := range n {
		refs[i], auths[i], _ = e.checkout(t, now, tok, `{"plan_id":1}`)
	}
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Go(func() {
			_, errs[i] = e.svc.Verify(context.Background(), uid, refs[i], auths[i], map[string]string{"status": "OK"}, tNow)
		})
	}
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, n, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, uid))
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM plus_subscriptions a JOIN plus_subscriptions b
		ON a.user_id = b.user_id AND a.id < b.id AND a.starts_at < b.ends_at AND b.starts_at < a.ends_at
		WHERE a.user_id = ?`, uid), "the periods follow each other")
	var last time.Time
	require.NoError(t, e.db.QueryRow(`SELECT MAX(ends_at) FROM plus_subscriptions WHERE user_id = ?`, uid).Scan(&last))
	assert.Equal(t, plus.AddMonths(tNow, n).Format(time.DateTime), last.In(tehran).Format(time.DateTime), "every paid month is kept")

	first := e.planCredits(t, plan1m)
	b, err := ledger.Balance(ctx, uid, tNow)
	require.NoError(t, err)
	assert.Equal(t, first, b.PlanTotal, "one period's credits at a time")
	// The second month too: had the periods overlapped, every running one would bring its own bucket.
	b, err = ledger.Balance(ctx, uid, plus.AddMonths(tNow, 1))
	require.NoError(t, err)
	assert.Equal(t, first, b.PlanTotal, "one period's credits at a time")
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM ai_credit_grants WHERE user_id = ?`, uid))
}

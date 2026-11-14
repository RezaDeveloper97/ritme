package credits_test

// The AI credit ledger (goose 00125) against MariaDB: spending order, period credits that live and die with their
// period, never-expiring extra credits that need a running subscription, the trial's credits, reserve-first
// concurrency, idempotent refunds, user isolation and the admin adjustments.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	plan1m = 1 // seeded plus_1m: 100 credits
	plan3m = 2 // seeded plus_3m: 300 credits
)

var t0 = time.Date(2026, 11, 12, 10, 0, 0, 0, civildate.Tehran)

type env struct {
	db  *sql.DB
	svc *credits.Service
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	return &env{db: db, svc: credits.NewService(db)}
}

func (e *env) exec(t *testing.T, q string, args ...any) sql.Result {
	t.Helper()
	res, err := e.db.Exec(q, args...)
	require.NoError(t, err)
	return res
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

func (e *env) user(t *testing.T, mobile string) uint64 {
	t.Helper()
	id, err := e.exec(t, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, NOW(), NOW())`, mobile).LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // positive id
}

func stamp(tm time.Time) string { return tm.Format("2006-01-02 15:04:05") }

// subscription adds a running-or-not Plus period [from, to); planID 0 = a period without a plan (admin gift).
func (e *env) subscription(t *testing.T, userID uint64, planID int, from, to time.Time) {
	t.Helper()
	var plan any
	if planID > 0 {
		plan = planID
	}
	e.exec(t, `INSERT INTO plus_subscriptions (user_id, plan_id, status, source, starts_at, ends_at, created_at, updated_at)
		VALUES (?, ?, 'active', 'purchase', ?, ?, NOW(), NOW())`, userID, plan, stamp(from), stamp(to))
}

func (e *env) trial(t *testing.T, userID uint64, from, to time.Time) {
	t.Helper()
	e.exec(t, `INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`,
		userID, stamp(from), stamp(to))
}

// extra adds a bought pack's credits (kind extra, never expiring).
func (e *env) extra(t *testing.T, userID uint64, n int) {
	t.Helper()
	e.exec(t, `INSERT INTO ai_credit_grants (user_id, kind, amount, remaining, created_at, updated_at) VALUES (?, 'extra', ?, ?, NOW(), NOW())`,
		userID, n, n)
}

func (e *env) balance(t *testing.T, userID uint64, at time.Time) credits.Balance {
	t.Helper()
	b, err := e.svc.Balance(context.Background(), userID, at)
	require.NoError(t, err)
	return b
}

func day(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }

func TestSpendOrder_PeriodFirstThenExtra(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001001")
	e.subscription(t, uid, plan1m, t0.Add(-day(1)), t0.Add(day(29)))
	e.extra(t, uid, 10)

	b := e.balance(t, uid, t0)
	assert.Equal(t, credits.Balance{Active: true, Period: credits.PeriodPlus, PeriodEndsAt: t0.Add(day(29)), PlanTotal: 100, PlanRemaining: 100, Extra: 10}, b)
	assert.Equal(t, uint32(110), b.Spendable())

	r, err := e.svc.Reserve(ctx, uid, "assistant", 95, t0)
	require.NoError(t, err)
	assert.Equal(t, uint32(5), r.Balance.PlanRemaining)
	assert.Equal(t, uint32(10), r.Balance.Extra, "the period's credits go first")

	// One use split across both buckets: 5 from the period, 3 from the extra credits.
	r, err = e.svc.Reserve(ctx, uid, "lab_analysis", 8, t0.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, uint32(0), r.Balance.PlanRemaining)
	assert.Equal(t, uint32(7), r.Balance.Extra)
	var plan, extra int
	require.NoError(t, e.db.QueryRow(`SELECT plan_credits, extra_credits FROM ai_credit_spends WHERE id = ?`, r.ID).Scan(&plan, &extra))
	assert.Equal(t, [2]int{5, 3}, [2]int{plan, extra})
	b = e.balance(t, uid, t0.Add(time.Minute))
	assert.Equal(t, uint32(0), b.PlanRemaining)
	assert.Equal(t, uint32(7), b.Extra)

	// Refunding the split use puts each part back into its own bucket.
	require.NoError(t, e.svc.Refund(ctx, r, t0.Add(2*time.Minute)))
	b = e.balance(t, uid, t0.Add(2*time.Minute))
	assert.Equal(t, uint32(5), b.PlanRemaining)
	assert.Equal(t, uint32(10), b.Extra)
}

func TestPeriodCredits_EndWithThePeriodAndNeverCarryOver(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001002")
	end := t0.Add(day(30))
	e.subscription(t, uid, plan1m, t0, end)
	_, err := e.svc.Reserve(ctx, uid, "assistant", 40, t0)
	require.NoError(t, err)
	assert.Equal(t, uint32(60), e.balance(t, uid, end.Add(-time.Second)).PlanRemaining)

	// The period is over: its 60 credits are gone with it, nothing is spendable.
	b := e.balance(t, uid, end)
	assert.False(t, b.Active)
	assert.Zero(t, b.PlanTotal)
	assert.Zero(t, b.PlanRemaining)
	assert.Zero(t, b.Spendable())
	_, err = e.svc.Reserve(ctx, uid, "assistant", 1, end)
	require.ErrorIs(t, err, credits.ErrNoSubscription)

	// A new period starts from its plan's amount — the old leftover does not carry over.
	e.subscription(t, uid, plan3m, end.Add(day(2)), end.Add(day(92)))
	b = e.balance(t, uid, end.Add(day(3)))
	assert.True(t, b.Active)
	assert.Equal(t, uint32(300), b.PlanTotal)
	assert.Equal(t, uint32(300), b.PlanRemaining)
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM ai_credit_grants WHERE user_id = ? AND kind = 'plan'`, uid), "one bucket per period")

	// The bucket follows its period by JOIN: an admin who ends the period early takes its credits with it.
	e.exec(t, `UPDATE plus_subscriptions SET ends_at = ? WHERE user_id = ? AND plan_id = ?`, stamp(end.Add(day(4))), uid, plan3m)
	assert.False(t, e.balance(t, uid, end.Add(day(5))).Active)
}

func TestExtraCredits_LockedWithoutSubscriptionNeverLost(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001003")
	e.subscription(t, uid, plan1m, t0, t0.Add(day(30)))
	e.extra(t, uid, 50)
	_, err := e.svc.Reserve(ctx, uid, "assistant", 100, t0)
	require.NoError(t, err)
	assert.Equal(t, uint32(50), e.balance(t, uid, t0).Extra, "the period paid for the whole use")

	// The subscription ends: the extra credits are kept (reported) but cannot be spent.
	after := t0.Add(day(31))
	b := e.balance(t, uid, after)
	assert.False(t, b.Active)
	assert.Equal(t, uint32(50), b.Extra)
	assert.Zero(t, b.Spendable())
	r, err := e.svc.Reserve(ctx, uid, "assistant", 1, after)
	require.ErrorIs(t, err, credits.ErrNoSubscription)
	require.NotNil(t, r, "the refusal carries the balance")
	assert.Equal(t, uint32(50), r.Balance.Extra)
	assert.Equal(t, 50, e.count(t, `SELECT remaining FROM ai_credit_grants WHERE user_id = ? AND kind = 'extra'`, uid), "untouched")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ?`, uid), "the refusal recorded nothing")

	// Months later a new subscription (here one without a plan: no credits of its own) unlocks them.
	later := after.Add(day(200))
	e.subscription(t, uid, 0, later, later.Add(day(30)))
	b = e.balance(t, uid, later)
	assert.True(t, b.Active)
	assert.Zero(t, b.PlanTotal)
	assert.Equal(t, uint32(50), b.Spendable())
	r, err = e.svc.Reserve(ctx, uid, "lab_analysis", 5, later)
	require.NoError(t, err)
	assert.Equal(t, uint32(45), r.Balance.Extra)
}

func TestTrial_CarriesTrialCredits(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001004")
	e.trial(t, uid, t0, t0.Add(day(7)))
	b := e.balance(t, uid, t0)
	assert.Equal(t, credits.Balance{Active: true, Period: credits.PeriodTrial, PeriodEndsAt: t0.Add(day(7)), PlanTotal: 20, PlanRemaining: 20}, b)
	n, err := e.svc.TrialCredits(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(credits.DefaultTrialCredits), n)

	// The admin setting decides the amount of trials whose bucket does not exist yet.
	e.exec(t, "UPDATE plus_settings SET value = '7' WHERE `key` = 'trial_ai_credits'")
	other := e.user(t, "09120001005")
	e.trial(t, other, t0, t0.Add(day(7)))
	assert.Equal(t, uint32(7), e.balance(t, other, t0).PlanTotal)
	assert.Equal(t, uint32(20), e.balance(t, uid, t0).PlanTotal, "an existing bucket is never changed")

	// 0 = the trial has no AI credits of its own (still a running period for extra credits).
	e.exec(t, "UPDATE plus_settings SET value = '0' WHERE `key` = 'trial_ai_credits'")
	third := e.user(t, "09120001006")
	e.trial(t, third, t0, t0.Add(day(7)))
	b = e.balance(t, third, t0)
	assert.True(t, b.Active)
	assert.Zero(t, b.PlanTotal)
	_, err = e.svc.Reserve(ctx, third, "assistant", 1, t0)
	require.ErrorIs(t, err, credits.ErrInsufficient)

	// An unreadable setting falls back to the default.
	e.exec(t, "UPDATE plus_settings SET value = 'lots' WHERE `key` = 'trial_ai_credits'")
	n, err = e.svc.TrialCredits(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(credits.DefaultTrialCredits), n)

	// After the trial its credits are gone.
	assert.False(t, e.balance(t, uid, t0.Add(day(7))).Active)
}

func TestReserve_InsufficientTakesNothing(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001007")
	e.subscription(t, uid, 0, t0, t0.Add(day(30)))
	e.extra(t, uid, 3)
	r, err := e.svc.Reserve(ctx, uid, "lab_analysis", 5, t0)
	require.ErrorIs(t, err, credits.ErrInsufficient)
	require.NotNil(t, r)
	assert.Equal(t, uint32(3), r.Balance.Spendable())
	assert.Equal(t, 3, e.count(t, `SELECT remaining FROM ai_credit_grants WHERE user_id = ?`, uid))
	assert.Zero(t, e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ?`, uid))

	// A zero cost reserves nothing but still needs the running period.
	r, err = e.svc.Reserve(ctx, uid, "assistant", 0, t0)
	require.NoError(t, err)
	assert.Zero(t, r.ID)
	assert.NoError(t, e.svc.Refund(ctx, r, t0), "refunding nothing is a no-op")
	_, err = e.svc.Reserve(ctx, uid, "assistant", 0, t0.Add(day(40)))
	require.ErrorIs(t, err, credits.ErrNoSubscription)
}

func TestReserve_ParallelNeverOverspends(t *testing.T) {
	e := setup(t)
	uid := e.user(t, "09120001008")
	// A plan of 5 credits: the period's bucket is created by the racing reservations themselves.
	planID, err := e.exec(t, `INSERT INTO plus_plans (code, title, duration_months, price_rials, ai_credits, is_active, sort_order, created_at, updated_at)
		VALUES ('race', '{"fa":"آزمون"}', 1, 1000, 5, 0, 99, NOW(), NOW())`).LastInsertId()
	require.NoError(t, err)
	e.subscription(t, uid, int(planID), t0.Add(-time.Hour), t0.Add(day(30)))

	var wg sync.WaitGroup
	errs := make([]error, 20)
	for i := range errs {
		wg.Go(func() {
			_, errs[i] = e.svc.Reserve(context.Background(), uid, "assistant", 1, t0)
		})
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, credits.ErrInsufficient):
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	assert.Equal(t, 5, ok, "exactly the balance, however parallel")
	assert.Zero(t, e.balance(t, uid, t0).Spendable())
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM ai_credit_grants WHERE user_id = ?`, uid), "one bucket, however parallel")
	assert.Zero(t, e.count(t, `SELECT remaining FROM ai_credit_grants WHERE user_id = ?`, uid), "never negative")
	assert.Equal(t, 5, e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE user_id = ? AND status = 'spent'`, uid))
}

func TestRefund_IdempotentCappedAndByTime(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001009")
	e.subscription(t, uid, plan1m, t0, t0.Add(day(30)))

	r, err := e.svc.Reserve(ctx, uid, "lab_analysis", 5, t0)
	require.NoError(t, err)
	assert.Equal(t, t0, r.At)
	require.NoError(t, e.svc.Refund(ctx, r, t0))
	require.NoError(t, e.svc.Refund(ctx, r, t0), "a second refund is a no-op")
	assert.Equal(t, uint32(100), e.balance(t, uid, t0).PlanRemaining)
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE id = ? AND status = 'refunded'`, r.ID))
	require.NoError(t, e.svc.Refund(ctx, nil, t0), "nil is a no-op")

	// A refund never lifts a bucket above what it was granted.
	r, err = e.svc.Reserve(ctx, uid, "lab_analysis", 5, t0.Add(time.Minute))
	require.NoError(t, err)
	e.exec(t, `UPDATE ai_credit_grants SET remaining = amount WHERE user_id = ?`, uid)
	require.NoError(t, e.svc.Refund(ctx, r, t0.Add(time.Minute)))
	assert.Equal(t, 100, e.count(t, `SELECT remaining FROM ai_credit_grants WHERE user_id = ?`, uid))

	// RefundAt finds the reservation by its moment (a queued job's refund); twice is still once.
	at := t0.Add(2 * time.Minute)
	_, err = e.svc.Reserve(ctx, uid, "doc_extract", 3, at)
	require.NoError(t, err)
	assert.Equal(t, uint32(97), e.balance(t, uid, at).PlanRemaining)
	require.NoError(t, e.svc.RefundAt(ctx, uid, "assistant", at, at), "another feature's moment finds nothing")
	assert.Equal(t, uint32(97), e.balance(t, uid, at).PlanRemaining)
	require.NoError(t, e.svc.RefundAt(ctx, uid, "doc_extract", at.Add(time.Second), at), "another moment finds nothing")
	assert.Equal(t, uint32(97), e.balance(t, uid, at).PlanRemaining)
	require.NoError(t, e.svc.RefundAt(ctx, uid, "doc_extract", at, at))
	require.NoError(t, e.svc.RefundAt(ctx, uid, "doc_extract", at, at))
	assert.Equal(t, uint32(100), e.balance(t, uid, at).PlanRemaining)

	// Two uses of the same second: each RefundAt gives back one of them.
	for range 2 {
		_, err = e.svc.Reserve(ctx, uid, "doc_extract", 3, at.Add(time.Hour))
		require.NoError(t, err)
	}
	require.NoError(t, e.svc.RefundAt(ctx, uid, "doc_extract", at.Add(time.Hour), at.Add(time.Hour)))
	assert.Equal(t, uint32(97), e.balance(t, uid, at.Add(time.Hour)).PlanRemaining)
	require.NoError(t, e.svc.RefundAt(ctx, uid, "doc_extract", at.Add(time.Hour), at.Add(time.Hour)))
	assert.Equal(t, uint32(100), e.balance(t, uid, at.Add(time.Hour)).PlanRemaining)

	// Refunded uses never appear in the history or the summary.
	items, next, err := e.svc.History(ctx, uid, 0)
	require.NoError(t, err)
	assert.Empty(t, items)
	assert.Zero(t, next)
}

func TestRefund_OtherUserCannotRefund(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a := e.user(t, "09120001010")
	b := e.user(t, "09120001011")
	e.subscription(t, a, plan1m, t0, t0.Add(day(30)))
	e.subscription(t, b, plan1m, t0, t0.Add(day(30)))
	r, err := e.svc.Reserve(ctx, a, "lab_analysis", 5, t0)
	require.NoError(t, err)
	_, err = e.svc.Reserve(ctx, b, "lab_analysis", 5, t0)
	require.NoError(t, err)

	forged := *r
	forged.UserID = b
	require.NoError(t, e.svc.Refund(ctx, &forged, t0))
	require.NoError(t, e.svc.RefundAt(ctx, b, "lab_analysis", t0.Add(time.Hour), t0))
	assert.Equal(t, uint32(95), e.balance(t, a, t0).PlanRemaining, "A's reservation stays counted")
	assert.Equal(t, uint32(95), e.balance(t, b, t0).PlanRemaining, "B gains nothing from A's reservation")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM ai_credit_spends WHERE id = ? AND status = 'spent'`, r.ID))

	// B's history never shows A's uses.
	items, _, err := e.svc.History(ctx, b, 0)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.NotEqual(t, r.ID, items[0].ID)
}

func TestAdminGrantAndDeduct(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001012")
	e.subscription(t, uid, plan1m, t0, t0.Add(day(30)))

	id, err := e.svc.AdminGrant(ctx, uid, 10, "support gift", t0)
	require.NoError(t, err)
	assert.Positive(t, id)
	_, err = e.svc.AdminGrant(ctx, uid, 0, "", t0)
	require.Error(t, err)
	_, err = e.svc.AdminGrant(ctx, uid, credits.MaxAdminGrant+1, "", t0)
	require.Error(t, err)
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM ai_credit_grants WHERE user_id = ? AND kind = 'admin' AND note = 'support gift'`, uid))
	b := e.balance(t, uid, t0)
	assert.Equal(t, uint32(10), b.Extra)
	assert.Equal(t, uint32(100), b.PlanRemaining)

	e.extra(t, uid, 4)
	taken, err := e.svc.AdminDeduct(ctx, uid, 12, t0)
	require.NoError(t, err)
	assert.Equal(t, uint32(12), taken)
	taken, err = e.svc.AdminDeduct(ctx, uid, 50, t0)
	require.NoError(t, err)
	assert.Equal(t, uint32(2), taken, "only what the extra buckets hold")
	b = e.balance(t, uid, t0)
	assert.Zero(t, b.Extra)
	assert.Equal(t, uint32(100), b.PlanRemaining, "a deduction never touches the period's credits")

	// A gift without a subscription is kept (locked) until one runs.
	other := e.user(t, "09120001013")
	_, err = e.svc.AdminGrant(ctx, other, 25, "", t0)
	require.NoError(t, err)
	b = e.balance(t, other, t0)
	assert.False(t, b.Active)
	assert.Equal(t, uint32(25), b.Extra)
}

func TestHistoryAndSummary(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid := e.user(t, "09120001014")
	e.subscription(t, uid, plan3m, t0.Add(-day(10)), t0.Add(day(80)))
	for i := range credits.HistoryPageSize + 2 {
		feature := "assistant"
		if i%2 == 1 {
			feature = "lab_analysis"
		}
		_, err := e.svc.Reserve(ctx, uid, feature, 1, t0.Add(-day(i%3)).Add(time.Duration(i)*time.Second))
		require.NoError(t, err, fmt.Sprint(i))
	}
	items, next, err := e.svc.History(ctx, uid, 0)
	require.NoError(t, err)
	require.Len(t, items, credits.HistoryPageSize)
	assert.Greater(t, items[0].ID, items[1].ID, "newest first")
	require.NotZero(t, next)
	more, last, err := e.svc.History(ctx, uid, next)
	require.NoError(t, err)
	assert.Len(t, more, 2)
	assert.Zero(t, last)

	byFeature, byDay, err := e.svc.Summary(ctx, uid, t0.Add(-day(1)).Truncate(24*time.Hour).Add(-3*time.Hour-30*time.Minute))
	require.NoError(t, err)
	total := int64(0)
	for _, f := range byFeature {
		total += f.Credits
	}
	days := int64(0)
	for _, d := range byDay {
		days += d.Credits
		assert.Len(t, d.Day, 10)
	}
	assert.Equal(t, total, days)
	assert.Positive(t, total)
}

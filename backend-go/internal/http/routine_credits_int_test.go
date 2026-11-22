package http_test

// T-M12-16 (T-05/T-06): the AI-credit money path end to end through the real routes (docs/ai-credits.md rules 1–8):
// buy a plan on the fake gateway → spend on a real AI-gated route (POST /logs/voice, fake AI provider) → a failed
// call is refunded to its own bucket → the period runs dry (402 ai_credits_exhausted) → a credit pack → the period's
// credits are spent first, then the extra ones → the period ends ([starts_at, ends_at): at ends_at exactly) → the
// extra credits are kept but locked (402 ai_subscription_required, the provider never called) → renew → a fresh
// period bucket (nothing carried over) and the extras usable again → every bank callback replayed changes nothing.
//
// Every amount comes from the database (admin data: ai_features.credit_cost, plus_plans.ai_credits,
// ai_credit_packs.credits), never from a literal; the assertions read the ledger (ai_credit_grants, ai_credit_spends)
// as well as the HTTP answers. "The provider was called" is read from ai_usage_logs: the AI client writes one row
// per provider call (internal/ai/usage), so a gated request that reaches no provider leaves it unchanged.

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"mime/multipart"
	"net/url"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/http/testenv"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

const (
	pathVoice    = "/api/v1/logs/voice"
	pathCredits  = "/api/v1/ai/credits"
	pathVerify   = "/api/v1/plus/verify"
	featureVoice = "voice_log"
)

// creditsFlow is the routine's user and the env.
type creditsFlow struct {
	t   *testing.T
	e   *testenv.Env
	acc testenv.Account
}

// voice posts a recording to the AI-gated voice route; fixture "" = the default transcript, "error" = the fake
// provider fails.
func (f *creditsFlow) voice(fixture string) testenv.Reply {
	f.t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("audio", "voice.webm")
	require.NoError(f.t, err)
	data := []byte("\x1a\x45\xdf\xa3\x9f\x42\x86\x81\x01webm-opus ")
	if fixture != "" {
		data = append(data, []byte(ai.FakeMarker+fixture)...)
	}
	_, err = fw.Write(data)
	require.NoError(f.t, err)
	require.NoError(f.t, w.Close())
	return f.e.Send(testenv.Request{Method: fiber.MethodPost, Path: pathVoice, Raw: b.Bytes(),
		ContentType: w.FormDataContentType(), Token: f.acc.Token})
}

// clearVoiceThrottles drops the voice feature's burst / hourly counters (they run on the real clock; the flow makes
// more calls than one minute's burst allows, and the throttles are not what this test is about).
func (f *creditsFlow) clearVoiceThrottles() {
	f.t.Helper()
	ctx := context.Background()
	rdb := f.e.Cache.Redis()
	iter := rdb.Scan(ctx, 0, f.e.Cache.Prefix()+"throttle:voice-*", 100).Iterator()
	for iter.Next(ctx) {
		require.NoError(f.t, rdb.Del(ctx, iter.Val()).Err())
	}
	require.NoError(f.t, iter.Err())
}

// providerCalls is how many provider calls the AI client logged for the user.
func (f *creditsFlow) providerCalls() int {
	return f.e.Count(`SELECT COUNT(*) FROM ai_usage_logs WHERE user_id = ?`, f.acc.ID)
}

// uint reads one unsigned number.
func (f *creditsFlow) uint(q string, args ...any) uint32 {
	f.t.Helper()
	var n uint32
	require.NoError(f.t, f.e.DB.QueryRowContext(context.Background(), q, args...).Scan(&n))
	return n
}

// grant is one ai_credit_grants row.
type grant struct {
	ID                uint64
	Kind              string
	SubID, InvoiceID  sql.NullInt64
	Amount, Remaining uint32
}

func (f *creditsFlow) grants() []grant {
	f.t.Helper()
	rows, err := f.e.DB.QueryContext(context.Background(),
		`SELECT id, kind, subscription_id, invoice_id, amount, remaining FROM ai_credit_grants WHERE user_id = ? ORDER BY id`, f.acc.ID)
	require.NoError(f.t, err)
	defer rows.Close()
	var out []grant
	for rows.Next() {
		var g grant
		require.NoError(f.t, rows.Scan(&g.ID, &g.Kind, &g.SubID, &g.InvoiceID, &g.Amount, &g.Remaining))
		out = append(out, g)
	}
	require.NoError(f.t, rows.Err())
	return out
}

// spend is one ai_credit_spends row.
type spend struct {
	ID                           uint64
	Feature, Status              string
	Credits, PlanPart, ExtraPart uint32
	Allocations                  string
}

func (f *creditsFlow) spends() []spend {
	f.t.Helper()
	rows, err := f.e.DB.QueryContext(context.Background(),
		`SELECT id, feature, status, credits, plan_credits, extra_credits, allocations FROM ai_credit_spends WHERE user_id = ? ORDER BY id`, f.acc.ID)
	require.NoError(f.t, err)
	defer rows.Close()
	var out []spend
	for rows.Next() {
		var s spend
		require.NoError(f.t, rows.Scan(&s.ID, &s.Feature, &s.Status, &s.Credits, &s.PlanPart, &s.ExtraPart, &s.Allocations))
		out = append(out, s)
	}
	require.NoError(f.t, rows.Err())
	return out
}

// credits is GET /ai/credits.
func (f *creditsFlow) credits() map[string]any {
	f.t.Helper()
	return f.e.MustOK(fiber.StatusOK, fiber.MethodGet, pathCredits, nil, f.acc.Token).Data()
}

// pay checks out (plan or pack), decides "success" on the fake gateway's TEST page and verifies the invoice:
// returns the verify body (for the replay).
func (f *creditsFlow) pay(checkoutPath string, body map[string]any) map[string]any {
	f.t.Helper()
	r := f.e.MustOK(fiber.StatusOK, fiber.MethodPost, checkoutPath, body, f.acc.Token)
	ref, _ := testenv.Path(r.Data(), "invoice", "reference").(string)
	authority, _ := testenv.Path(r.Data(), "payment", "authority").(string)
	require.NotEmpty(f.t, ref, r.Raw)
	require.NotEmpty(f.t, authority, r.Raw)

	// The tester presses "success" on the fake provider's page (a form post, 303 back to the return URL).
	page := f.e.Send(testenv.Request{Method: fiber.MethodPost, Path: "/api/v1/payments/fake/pay/" + url.PathEscape(authority),
		Raw: []byte("decision=success"), ContentType: fiber.MIMEApplicationForm})
	require.Equal(f.t, fiber.StatusSeeOther, page.Status, page.Raw)

	v := map[string]any{"reference": ref, "authority": authority, "status": "OK"}
	r = f.e.MustOK(fiber.StatusOK, fiber.MethodPost, pathVerify, v, f.acc.Token)
	require.Equal(f.t, "paid", testenv.Path(r.Data(), "invoice", "status"), r.Raw)
	require.NotEqual(f.t, true, r.Data()["already_verified"], r.Raw)
	return v
}

// refused asserts a gate refusal: the HTTP status, the managed error code, and that no provider call, no grant
// change and no spend row came of it.
func (f *creditsFlow) refused(label string, status int, code string, before []grant) testenv.Reply {
	f.t.Helper()
	calls, spends := f.providerCalls(), len(f.spends())
	r := f.voice("")
	require.Equal(f.t, status, r.Status, "%s: %s", label, r.Raw)
	assert.Equal(f.t, code, r.Body["error_code"], label)
	assert.Equal(f.t, featureVoice, r.Body["feature"], label)
	assert.Equal(f.t, calls, f.providerCalls(), "%s: the provider must not be called", label)
	assert.Len(f.t, f.spends(), spends, "%s: no spend row", label)
	assert.Equal(f.t, before, f.grants(), "%s: no bucket changes", label)
	return r
}

// spent asserts one successful use: 200, the provider called, one new `spent` row of cost credits split
// plan/extra as given, and each bucket down by its share.
func (f *creditsFlow) spent(label string, cost, planPart, extraPart uint32) spend {
	f.t.Helper()
	calls, before := f.providerCalls(), f.spends()
	r := f.voice("")
	require.Equal(f.t, fiber.StatusOK, r.Status, "%s: %s", label, r.Raw)
	assert.NotEmpty(f.t, r.Data()["transcript"], label)
	assert.Greater(f.t, f.providerCalls(), calls, "%s: the provider is called", label)
	after := f.spends()
	require.Len(f.t, after, len(before)+1, label)
	s := after[len(after)-1]
	assert.Equal(f.t, spend{ID: s.ID, Feature: featureVoice, Status: "spent", Credits: cost, PlanPart: planPart,
		ExtraPart: extraPart, Allocations: s.Allocations}, s, label)
	return s
}

func asUint(v any) uint32 {
	n, _ := v.(float64)
	return uint32(n)
}

func TestRoutineCredits_BuySpendPackExpireRenew(t *testing.T) {
	e := testenv.New(t)
	ctx := context.Background()

	// Admin fixtures (what the panel writes: هوش مصنوعی → feature cost, plan credits). The cost is > 1 so a single
	// use can straddle the period and the extra buckets; the plan carries two uses and a remainder below one use.
	_, err := e.DB.ExecContext(ctx, "UPDATE ai_features SET credit_cost = 3, is_active = 1 WHERE `key` = ?", featureVoice)
	require.NoError(t, err)
	f := &creditsFlow{t: t, e: e}
	cost := f.uint("SELECT credit_cost FROM ai_features WHERE `key` = ?", featureVoice)
	require.Greater(t, cost, uint32(1))
	_, err = e.DB.ExecContext(ctx, `UPDATE plus_plans SET ai_credits = ? WHERE code = 'plus_1m'`, 2*cost+1)
	require.NoError(t, err)
	var planID uint64
	require.NoError(t, e.DB.QueryRowContext(ctx, `SELECT id FROM plus_plans WHERE code = 'plus_1m'`).Scan(&planID))
	planCredits := f.uint(`SELECT ai_credits FROM plus_plans WHERE id = ?`, planID)
	var packID uint64
	require.NoError(t, e.DB.QueryRowContext(ctx,
		`SELECT id FROM ai_credit_packs WHERE is_active = 1 AND credits >= ? ORDER BY sort_order, id LIMIT 1`, cost).Scan(&packID))
	packCredits := f.uint(`SELECT credits FROM ai_credit_packs WHERE id = ?`, packID)

	f.acc = e.SignUp()
	// The test clock in Tehran wall time (what the ledger stores).
	t0, err := time.Parse(time.RFC3339, testenv.DefaultNow)
	require.NoError(t, err)

	// 1. No subscription: 402 ai_subscription_required, nothing reserved, the provider not called (rule 1).
	r := f.refused("before purchase", fiber.StatusPaymentRequired, "ai_subscription_required", nil)
	assert.EqualValues(t, 0, r.Body["extra_credits"])
	c := f.credits()
	assert.Equal(t, false, c["can_use"])
	assert.EqualValues(t, 0, c["spendable"])

	// 2. Buy the plan on the fake gateway: a running period with the plan's credits (rule 2).
	planVerify := f.pay("/api/v1/plus/checkout", map[string]any{"plan_id": planID})
	c = f.credits()
	assert.Equal(t, true, c["can_use"])
	assert.Equal(t, "plus", c["period"])
	assert.Equal(t, planCredits, asUint(testenv.Path(c, "plan", "total")))
	assert.Equal(t, planCredits, asUint(testenv.Path(c, "plan", "remaining")))
	assert.Equal(t, uint32(0), asUint(testenv.Path(c, "extra", "remaining")))
	assert.Equal(t, planCredits, asUint(c["spendable"]))
	var subID int64
	var startsAt, endsAt time.Time
	require.NoError(t, e.DB.QueryRowContext(ctx, `SELECT id, starts_at, ends_at FROM plus_subscriptions WHERE user_id = ?`, f.acc.ID).
		Scan(&subID, &startsAt, &endsAt))
	g := f.grants()
	require.Len(t, g, 1, "one bucket for the period")
	planGrant := g[0]
	assert.Equal(t, grant{ID: planGrant.ID, Kind: "plan", SubID: sql.NullInt64{Int64: subID, Valid: true},
		Amount: planCredits, Remaining: planCredits}, planGrant)

	// 3. A use costs the admin-set cost, taken from the period (rule 4, 5).
	f.spent("first use", cost, cost, 0)
	assert.Equal(t, planCredits-cost, f.grants()[0].Remaining)
	c = f.credits()
	assert.Equal(t, cost, asUint(testenv.Path(c, "plan", "used")))
	assert.Equal(t, planCredits-cost, asUint(c["spendable"]))

	// 4. The provider fails: the reserved credits go back to the bucket they came from (rule 6, 7).
	calls := f.providerCalls()
	r = f.voice("error")
	require.Equal(t, fiber.StatusServiceUnavailable, r.Status, r.Raw)
	assert.Equal(t, "ai_failed", r.Body["error_code"])
	assert.Greater(t, f.providerCalls(), calls, "the failing provider was called")
	sp := f.spends()
	require.Len(t, sp, 2)
	assert.Equal(t, "refunded", sp[1].Status)
	assert.Equal(t, cost, sp[1].PlanPart)
	assert.Equal(t, planCredits-cost, f.grants()[0].Remaining, "refunded to the period bucket")

	// 5. The second use leaves less than one use in the period.
	f.spent("second use", cost, cost, 0)
	left := planCredits - 2*cost
	require.Positive(t, left)
	require.Less(t, left, cost)

	// 6. The period cannot cover a use: 402 ai_credits_exhausted with the balance, nothing taken (boundary:
	// spendable = cost - 1 or less).
	r = f.refused("period exhausted", fiber.StatusPaymentRequired, "ai_credits_exhausted", f.grants())
	assert.Equal(t, cost, asUint(r.Body["required"]))
	assert.Equal(t, left, asUint(r.Body["balance"]))
	assert.Equal(t, left, asUint(r.Body["plan_remaining"]))
	assert.EqualValues(t, 0, r.Body["extra_credits"])
	assert.NotNil(t, r.Body["period_ends_at"])

	// 7. Buy a credit pack: an extra bucket tied to its invoice (rule 3).
	packVerify := f.pay("/api/v1/plus/credit-packs/checkout", map[string]any{"pack_id": packID})
	g = f.grants()
	require.Len(t, g, 2)
	extraGrant := g[1]
	assert.Equal(t, "extra", extraGrant.Kind)
	assert.True(t, extraGrant.InvoiceID.Valid)
	assert.Equal(t, packCredits, extraGrant.Amount)
	assert.Equal(t, packCredits, extraGrant.Remaining)
	assert.Equal(t, 1, e.Count(`SELECT COUNT(*) FROM plus_invoices WHERE id = ? AND kind = 'credit_pack' AND credits = ?`,
		extraGrant.InvoiceID.Int64, packCredits))

	// 8. The next use takes what is left of the period first, the rest from the extra credits (rule 4).
	straddle := f.spent("straddling use", cost, left, cost-left)
	assert.JSONEq(t, fmt.Sprintf(`[{"grant_id":%d,"credits":%d},{"grant_id":%d,"credits":%d}]`,
		planGrant.ID, left, extraGrant.ID, cost-left), straddle.Allocations)
	g = f.grants()
	assert.Equal(t, uint32(0), g[0].Remaining)
	extraLeft := packCredits - (cost - left)
	assert.Equal(t, extraLeft, g[1].Remaining)

	// 9. One second before ends_at the period still runs; at ends_at exactly it is over (rule 7, half-open).
	endsLocal := time.Date(endsAt.Year(), endsAt.Month(), endsAt.Day(), endsAt.Hour(), endsAt.Minute(), endsAt.Second(), 0, civildate.Tehran)
	require.True(t, endsLocal.After(t0))
	e.SetNow(endsLocal.Add(-time.Second).Format(time.RFC3339))
	assert.Equal(t, true, f.credits()["can_use"], "one second before the end")
	e.SetNow(endsLocal.Format(time.RFC3339))
	c = f.credits()
	assert.Equal(t, false, c["can_use"], "at ends_at the subscription is over")
	assert.Nil(t, c["period"])
	assert.EqualValues(t, 0, c["spendable"])
	assert.Equal(t, extraLeft, asUint(testenv.Path(c, "extra", "remaining")), "extra credits are kept")
	assert.Equal(t, true, testenv.Path(c, "extra", "locked"), "… but locked")

	// 10. The gated route refuses with the managed sheet; the provider is never called, the extras untouched.
	f.clearVoiceThrottles()
	r = f.refused("after the period", fiber.StatusPaymentRequired, "ai_subscription_required", f.grants())
	assert.Equal(t, extraLeft, asUint(r.Body["extra_credits"]))

	// 11. Renew a day later: a new period with a fresh bucket of the plan's amount (nothing carried over), the
	// extras unlocked again.
	e.SetNow(endsLocal.Add(24 * time.Hour).Format(time.RFC3339))
	renewVerify := f.pay("/api/v1/plus/checkout", map[string]any{"plan_id": planID})
	c = f.credits()
	assert.Equal(t, true, c["can_use"])
	assert.Equal(t, planCredits, asUint(testenv.Path(c, "plan", "total")))
	assert.Equal(t, planCredits, asUint(testenv.Path(c, "plan", "remaining")))
	assert.Equal(t, extraLeft, asUint(testenv.Path(c, "extra", "remaining")))
	assert.Equal(t, false, testenv.Path(c, "extra", "locked"))
	assert.Equal(t, planCredits+extraLeft, asUint(c["spendable"]))
	g = f.grants()
	require.Len(t, g, 3)
	renewGrant := g[2]
	assert.Equal(t, "plan", renewGrant.Kind)
	assert.NotEqual(t, sql.NullInt64{Int64: subID, Valid: true}, renewGrant.SubID, "the new period's own bucket")
	assert.Equal(t, uint32(0), g[0].Remaining, "the old period's bucket stays spent")

	// 12. Extra credits are usable again — the new period's credits go first.
	f.spent("after renewal", cost, cost, 0)
	assert.Equal(t, planCredits-cost, f.grants()[2].Remaining)
	assert.Equal(t, extraLeft, f.grants()[1].Remaining)

	// 13. Every bank callback replayed: 200 already_verified, no new period, bucket, receipt or credit.
	subs := e.Count(`SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, f.acc.ID)
	receipts := e.Count(`SELECT COUNT(*) FROM plus_receipts WHERE user_id = ?`, f.acc.ID)
	assert.Equal(t, 2, subs)
	before, beforeCredits := f.grants(), f.credits()
	for _, v := range []map[string]any{planVerify, packVerify, renewVerify} {
		r = e.MustOK(fiber.StatusOK, fiber.MethodPost, pathVerify, v, f.acc.Token)
		assert.Equal(t, true, r.Data()["already_verified"], r.Raw)
	}
	assert.Equal(t, subs, e.Count(`SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?`, f.acc.ID))
	assert.Equal(t, receipts, e.Count(`SELECT COUNT(*) FROM plus_receipts WHERE user_id = ?`, f.acc.ID))
	assert.Equal(t, before, f.grants())
	assert.Equal(t, beforeCredits, f.credits())

	// The ledger in the end: five uses of the admin cost, the failed one refunded.
	sp = f.spends()
	require.Len(t, sp, 5)
	for i, want := range []string{"spent", "refunded", "spent", "spent", "spent"} {
		assert.Equal(t, want, sp[i].Status, "spend %d", i)
		assert.Equal(t, cost, sp[i].Credits, "spend %d", i)
	}
}

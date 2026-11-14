package access

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Every AI feature has a policy whose consent code is in the consent catalog and a declared feature slot.
func TestPoliciesComplete(t *testing.T) {
	for _, f := range []ai.Feature{ai.FeatureVoiceLog, ai.FeatureLabAnalysis, ai.FeatureAssistant, ai.FeatureDocExtract} {
		p, ok := PolicyFor(f)
		require.True(t, ok, f)
		_, ok = consent.Lookup(p.Consent)
		assert.True(t, ok, "%s: consent %s", f, p.Consent)
		_, ok = ai.SpecOf(f)
		assert.True(t, ok, "%s: declared with ai.RegisterFeature", f)
	}
	assert.Panics(t, func() { NewGuard(Options{}).Require("nope") })
	assert.Len(t, Policies(), 4)
	for _, p := range Policies() {
		assert.NotEmpty(t, p.Throttles, "%s: every AI feature is throttled (B-N6-05b)", p.Feature)
	}
}

func TestInflight(t *testing.T) {
	f := &inflight{max: 2, n: map[uint64]int{}}
	assert.True(t, f.acquire(1))
	assert.True(t, f.acquire(1))
	assert.False(t, f.acquire(1), "a third parallel request of the same user")
	assert.True(t, f.acquire(2), "other users are independent")
	f.release(1)
	assert.True(t, f.acquire(1))
	f.release(1)
	f.release(1)
	f.release(2)
	assert.Empty(t, f.n, "no entry left behind")
	var r *Reservation
	assert.NotPanics(t, func() { r.Refund(t.Context()) }, "a nil reservation refunds nothing")
}

func TestErrorMapping(t *testing.T) {
	status := func(err error) (int, string) {
		var fe *httpx.FailError
		require.ErrorAs(t, err, &fe)
		code, _ := fe.Body().Get("error_code")
		return fe.Status, fmt.Sprint(code)
	}
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{&consent.RequiredError{Code: consent.AIAssistant, Version: 1, Reason: consent.ReasonMissing}, 403, consent.CodeConsentRequired},
		{ai.ErrUnavailable, 503, CodeUnavailable},
		{fmt.Errorf("x: %w", ai.ErrBudgetExceeded), 503, CodeBudgetExhausted},
		{fmt.Errorf("%w: status 500", ai.ErrUpstream), 503, CodeFailed},
		{ai.ErrInvalidRequest, 422, CodeInvalidRequest},
		{fmt.Errorf("x: %w", ai.ErrUserBudgetExceeded), 429, CodeUserBudgetExhausted},
	}
	for _, tc := range cases {
		s, c := status(Error(tc.err, "fa"))
		assert.Equal(t, tc.status, s, tc.code)
		assert.Equal(t, tc.code, c)
	}
	other := errors.New("db")
	assert.Equal(t, other, Denied(other, ai.FeatureAssistant, 1, credits.Balance{}, "fa"))
	assert.Equal(t, other, Error(other, "fa"))
	for _, k := range []string{CodeUnavailable, CodeFailed, CodeBudgetExhausted, CodeInvalidRequest, CodeUserBudgetExhausted, CodeBusy, CodeTooMany,
		CodeSubscriptionRequired, CodeCreditsExhausted, CodeFeatureDisabled} {
		assert.NotEqual(t, k, T(k, "fa"))
		assert.NotEqual(t, T(k, "en"), T(k, "fa"))
		assert.Equal(t, T(k, "en"), T(k, "xx"))
	}
}

// The three credit refusals carry exactly the fields the clients' AI gate reads (docs/ai-credits.md).
func TestDeniedShapes(t *testing.T) {
	ends := time.Date(2026, 12, 12, 10, 0, 0, 0, civildate.Tehran)
	b := credits.Balance{Active: true, Period: credits.PeriodPlus, PeriodEndsAt: ends, PlanTotal: 100, PlanRemaining: 2, Extra: 1}
	body := func(err error) (int, string) {
		var fe *httpx.FailError
		require.ErrorAs(t, err, &fe)
		raw, mErr := json.Marshal(fe.Body())
		require.NoError(t, mErr)
		return fe.Status, string(raw)
	}
	st, raw := body(Denied(credits.ErrFeatureDisabled, ai.FeatureLabAnalysis, 5, b, "en"))
	assert.Equal(t, 403, st)
	assert.JSONEq(t, `{"success":false,"message":"`+T(CodeFeatureDisabled, "en")+`","error_code":"ai_feature_disabled","feature":"lab_analysis"}`, raw)

	st, raw = body(Denied(fmt.Errorf("x: %w", credits.ErrNoSubscription), ai.FeatureAssistant, 1, credits.Balance{Extra: 50}, "en"))
	assert.Equal(t, 402, st)
	assert.JSONEq(t, `{"success":false,"message":"`+T(CodeSubscriptionRequired, "en")+`","error_code":"ai_subscription_required",
		"feature":"assistant","extra_credits":50}`, raw)

	st, raw = body(Denied(credits.ErrInsufficient, ai.FeatureLabAnalysis, 5, b, "en"))
	assert.Equal(t, 402, st)
	assert.JSONEq(t, `{"success":false,"message":"`+T(CodeCreditsExhausted, "en")+`","error_code":"ai_credits_exhausted",
		"feature":"lab_analysis","required":5,"balance":3,"plan_remaining":2,"extra_credits":1,"period_ends_at":"2026-12-12T10:00:00+03:30"}`, raw)
	_, raw = body(Denied(credits.ErrInsufficient, ai.FeatureLabAnalysis, 5, credits.Balance{}, "en"))
	assert.Contains(t, raw, `"period_ends_at":null`)
}

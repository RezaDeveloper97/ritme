package access

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
)

// Every AI feature has a policy whose consent code is in the consent catalog and whose Plus key exists.
func TestPoliciesComplete(t *testing.T) {
	for _, f := range []ai.Feature{ai.FeatureVoiceLog, ai.FeatureLabAnalysis, ai.FeatureAssistant, ai.FeatureDocExtract} {
		p, ok := PolicyFor(f)
		require.True(t, ok, f)
		_, ok = consent.Lookup(p.Consent)
		assert.True(t, ok, "%s: consent %s", f, p.Consent)
		if p.Plus != "" {
			_, ok = plus.Lookup(p.Plus)
			assert.True(t, ok, "%s: plus %s", f, p.Plus)
		}
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
	assert.Equal(t, other, Error(other, "fa"))
	for _, k := range []string{CodeUnavailable, CodeFailed, CodeBudgetExhausted, CodeInvalidRequest, CodeUserBudgetExhausted, CodeBusy, CodeTooMany} {
		assert.NotEqual(t, k, T(k, "fa"))
		assert.NotEqual(t, T(k, "en"), T(k, "fa"))
		assert.Equal(t, T(k, "en"), T(k, "xx"))
	}
}

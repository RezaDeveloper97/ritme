package users_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/users"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// A-08 boundaries: recent/prior ≤ 50 % high, ≤ 75 % medium, else low; 7 days inactive high; < 14 days old new.
func TestChurn_SpecBoundaries(t *testing.T) {
	old := 100
	for _, c := range []struct {
		name          string
		recent, prior int
		want          string
	}{
		{"5/10", 5, 10, "high"},
		{"6/10", 6, 10, "medium"},
		{"7.5/10 (15/20 ≡ 9/12)", 9, 12, "medium"},
		{"8/10", 8, 10, "low"},
		{"no baseline", 0, 0, "low"},
		{"growth", 12, 4, "low"},
	} {
		got := users.Churn(users.ChurnInput{Recent: c.recent, Prior: c.prior, DaysInactive: 1, AccountAgeDays: old})
		assert.Equal(t, c.want, got, c.name)
	}
	assert.Equal(t, "high", users.Churn(users.ChurnInput{Recent: 9, Prior: 10, DaysInactive: 7, AccountAgeDays: old}))
	assert.Equal(t, "low", users.Churn(users.ChurnInput{Recent: 9, Prior: 10, DaysInactive: 6, AccountAgeDays: old}))
	assert.Equal(t, "high", users.Churn(users.ChurnInput{NeverActive: true, AccountAgeDays: old}))
	assert.Equal(t, "new", users.Churn(users.ChurnInput{Recent: 0, Prior: 10, DaysInactive: 9, AccountAgeDays: 10}))
	assert.Equal(t, "new", users.Churn(users.ChurnInput{NeverActive: true, AccountAgeDays: 13}))
	assert.Equal(t, "high", users.Churn(users.ChurnInput{NeverActive: true, AccountAgeDays: 14}))
}

func TestMainWarning_RequiresInputFirst(t *testing.T) {
	var s engine.WarningSet
	assert.Equal(t, "", users.MainWarning(uint32(s)))
	s.Add(engine.WarnLowHistory)
	s.Add(engine.WarnPredictionBasedOutput)
	assert.Equal(t, "low_history", users.MainWarning(uint32(s)))
	s.Add(engine.WarnStaleOnboardingAnchor)
	assert.Equal(t, "stale_onboarding_anchor", users.MainWarning(uint32(s)))
}

func TestRefCode_DeterministicAndKeyed(t *testing.T) {
	a, err := users.NewRefCoder(nil, "production", "secret-a", nil)
	require.NoError(t, err)
	b, err := users.NewRefCoder(nil, "production", "secret-b", nil)
	require.NoError(t, err)
	assert.Equal(t, a.Code(42, 6), a.Code(42, 6))
	assert.NotEqual(t, a.Code(42, 6), b.Code(42, 6), "the code depends on the secret, not only the id")
	assert.NotEqual(t, a.Code(42, 6), a.Code(43, 6))
	assert.Regexp(t, `^RT-[A-Z2-7]{6}$`, a.Code(42, 6))
	assert.True(t, strings.HasPrefix(a.Code(42, 8), a.Code(42, 6)), "longer codes extend the short one")
	assert.NotContains(t, a.Code(10, 6), "000A", "not the hex id")

	_, err = users.NewRefCoder(nil, "production", "", nil)
	require.ErrorIs(t, err, users.ErrRefSecretMissing)
	_, err = users.NewRefCoder(nil, "staging", "", nil)
	require.ErrorIs(t, err, users.ErrRefSecretMissing)
	_, err = users.NewRefCoder(nil, "testing", "", nil)
	require.NoError(t, err)
}

func TestCycleLengths_DropsDates(t *testing.T) {
	d := civildate.MustParse
	assert.Equal(t, []int{28, 31}, users.CycleLengths([]civildate.Date{d("2026-10-01"), d("2026-09-03"), d("2026-08-03")}))
	assert.Equal(t, []int{}, users.CycleLengths([]civildate.Date{d("2026-10-01")}))
	assert.Equal(t, []int{}, users.CycleLengths(nil))
}

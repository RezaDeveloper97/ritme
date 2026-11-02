package plus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func find(t *testing.T, es []Entitlement, k Key) Entitlement {
	t.Helper()
	for _, e := range es {
		if e.Key == k {
			return e
		}
	}
	t.Fatalf("entitlement %s missing", k)
	return Entitlement{}
}

func TestDefinitions_KeysAreUniqueAndNamespaced(t *testing.T) {
	seen := map[Key]bool{}
	for _, d := range Definitions() {
		assert.False(t, seen[d.Key], "duplicate %s", d.Key)
		seen[d.Key] = true
		assert.Regexp(t, `^plus\.[a-z_]+$`, string(d.Key))
		assert.True(t, d.Plus.Enabled, "%s: Plus must include every Plus feature", d.Key)
	}
	for _, k := range []Key{DeepAnalysis, VoiceLog, LabAI, AssistantUnlimited, PDFShare, VisitDiscount} {
		assert.True(t, seen[k], "%s has no definition", k)
	}
}

func TestResolve_Plus(t *testing.T) {
	es := Resolve(TierPlus, map[Key]int{LabAI: 3, AssistantUnlimited: 40})
	lab := find(t, es, LabAI)
	assert.Equal(t, Entitlement{Key: LabAI, Allowed: true, Limit: 10, Used: 3, Remaining: 7}, lab)
	assert.Equal(t, Entitlement{Key: AssistantUnlimited, Allowed: true, Unlimited: true, FairUse: true, Limit: 300, Used: 40, Remaining: 260},
		find(t, es, AssistantUnlimited))
	for _, k := range []Key{DeepAnalysis, VoiceLog, PDFShare, VisitDiscount, AnalysisLongRange} {
		e := find(t, es, k)
		assert.True(t, e.Allowed && e.Unlimited, k)
	}
}

func TestResolve_TrialGetsPlusQuotas(t *testing.T) {
	assert.Equal(t, Resolve(TierPlus, map[Key]int{LabAI: 2}), Resolve(TierTrial, map[Key]int{LabAI: 2}))
}

func TestResolve_LabAIQuotaSpent(t *testing.T) {
	lab := find(t, Resolve(TierPlus, map[Key]int{LabAI: 10}), LabAI)
	assert.False(t, lab.Allowed)
	assert.Equal(t, 0, lab.Remaining)
	over := find(t, Resolve(TierPlus, map[Key]int{LabAI: 12}), LabAI) // e.g. counted before a downgrade
	assert.Equal(t, 0, over.Remaining)
	assert.False(t, over.Allowed)
}

func TestResolve_Free(t *testing.T) {
	es := Resolve(TierFree, map[Key]int{AssistantUnlimited: 4, LabAI: 1, DeepAnalysis: 2})
	for _, k := range []Key{DeepAnalysis, VoiceLog, PDFShare, VisitDiscount, DocAI, AnalysisLongRange} {
		e := find(t, es, k)
		assert.False(t, e.Allowed, k)
		assert.False(t, e.Unlimited, k)
		assert.Zero(t, e.Limit, k)
	}
	assert.Equal(t, 2, find(t, es, DeepAnalysis).Used, "usage is still reported")
	a := find(t, es, AssistantUnlimited)
	assert.Equal(t, Entitlement{Key: AssistantUnlimited, Allowed: true, Limit: 5, Used: 4, Remaining: 1}, a)
	a = find(t, Resolve(TierFree, map[Key]int{AssistantUnlimited: 5}), AssistantUnlimited)
	assert.False(t, a.Allowed)
}

// TestResolve_Table is the plan of T-M10-29 (D-178): every key × tier at a few usage points.
func TestResolve_Table(t *testing.T) {
	type want struct {
		allowed, unlimited, fairUse bool
		limit, remaining            int
	}
	cases := []struct {
		key  Key
		tier Tier
		used int
		want want
	}{
		// A-03: lab AI 1 per month free, 10 Plus.
		{LabAI, TierFree, 0, want{allowed: true, limit: 1, remaining: 1}},
		{LabAI, TierFree, 1, want{limit: 1}},
		{LabAI, TierPlus, 9, want{allowed: true, limit: 10, remaining: 1}},
		{LabAI, TierPlus, 10, want{limit: 10}},
		{LabAI, TierTrial, 0, want{allowed: true, limit: 10, remaining: 10}},
		// A-04: assistant 5 free, Plus «unlimited» with a fair-use cap of 300.
		{AssistantUnlimited, TierFree, 4, want{allowed: true, limit: 5, remaining: 1}},
		{AssistantUnlimited, TierFree, 5, want{limit: 5}},
		{AssistantUnlimited, TierPlus, 0, want{allowed: true, unlimited: true, fairUse: true, limit: 300, remaining: 300}},
		{AssistantUnlimited, TierPlus, 299, want{allowed: true, unlimited: true, fairUse: true, limit: 300, remaining: 1}},
		{AssistantUnlimited, TierPlus, 300, want{unlimited: true, fairUse: true, limit: 300}},
		{AssistantUnlimited, TierTrial, 301, want{unlimited: true, fairUse: true, limit: 300}},
		// D-178: the 1-year (and all-time) analysis range.
		{AnalysisLongRange, TierFree, 0, want{}},
		{AnalysisLongRange, TierPlus, 0, want{allowed: true, unlimited: true}},
		{AnalysisLongRange, TierTrial, 0, want{allowed: true, unlimited: true}},
		{DeepAnalysis, TierFree, 0, want{}},
		{DeepAnalysis, TierPlus, 50, want{allowed: true, unlimited: true}},
		{DocAI, TierFree, 0, want{}},
		{DocAI, TierPlus, 20, want{limit: 20}},
	}
	for _, tc := range cases {
		e := find(t, Resolve(tc.tier, map[Key]int{tc.key: tc.used}), tc.key)
		got := want{allowed: e.Allowed, unlimited: e.Unlimited, fairUse: e.FairUse, limit: e.Limit, remaining: e.Remaining}
		assert.Equal(t, tc.want, got, "%s %s used=%d", tc.key, tc.tier, tc.used)
		assert.Equal(t, tc.used, e.Used)
		assert.Equal(t, tc.want.fairUse && !tc.want.allowed, e.AtFairUseCap(), "%s %s used=%d", tc.key, tc.tier, tc.used)
	}
}

func TestQuota_Cap(t *testing.T) {
	assert.Equal(t, 10, Quota{Enabled: true, Limit: 10}.Cap())
	assert.Equal(t, 300, Quota{Enabled: true, FairUse: 300}.Cap())
	assert.Zero(t, unlimited.Cap())
	assert.Zero(t, locked.Cap())
}

func TestResolveAll_UsesTheGivenDefinitions(t *testing.T) {
	defs := Definitions()
	for i := range defs {
		if defs[i].Key == LabAI {
			defs[i].Free = Quota{Enabled: true, Limit: 3}
		}
	}
	lab := find(t, ResolveAll(defs, TierFree, map[Key]int{LabAI: 2}), LabAI)
	assert.Equal(t, Entitlement{Key: LabAI, Allowed: true, Limit: 3, Used: 2, Remaining: 1}, lab)
	assert.Equal(t, 1, find(t, Resolve(TierFree, nil), LabAI).Limit, "Resolve keeps the code default")
}

func TestLookup(t *testing.T) {
	d, ok := Lookup(LabAI)
	require.True(t, ok)
	assert.Equal(t, 10, d.QuotaFor(TierTrial).Limit)
	_, ok = Lookup("plus.nope")
	assert.False(t, ok)
}

func TestDefinitions_IsACopy(t *testing.T) {
	ds := Definitions()
	ds[0].Plus = Quota{}
	d, _ := Lookup(ds[0].Key)
	assert.True(t, d.Plus.Enabled)
}

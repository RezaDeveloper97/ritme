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

// The production entitlements are all «unlimited on Plus, locked for free»; the quota arithmetic of ResolveOne
// (limits, fair-use caps) is kept for future keys and tested on locally built definitions.
const (
	testLimited Key = "plus.test_limited"
	testFairUse Key = "plus.test_fair_use"
)

var (
	limitedDef = Definition{Key: testLimited, Plus: Quota{Enabled: true, Limit: 10}, Free: Quota{Enabled: true, Limit: 1}}
	fairUseDef = Definition{Key: testFairUse, Plus: Quota{Enabled: true, FairUse: 300}, Free: Quota{Enabled: true, Limit: 5}}
)

func TestDefinitions_KeysAreUniqueAndNamespaced(t *testing.T) {
	seen := map[Key]bool{}
	for _, d := range Definitions() {
		assert.False(t, seen[d.Key], "duplicate %s", d.Key)
		seen[d.Key] = true
		assert.Regexp(t, `^plus\.[a-z_]+$`, string(d.Key))
		assert.True(t, d.Plus.Enabled, "%s: Plus must include every Plus feature", d.Key)
	}
	for _, k := range []Key{DeepAnalysis, PDFShare, VisitDiscount, AnalysisLongRange, SMSReminders, Companion} {
		assert.True(t, seen[k], "%s has no definition", k)
	}
	// AI features spend credits (goose 00125): their retired monthly keys must never come back.
	for _, k := range []Key{"plus.voice_log", "plus.lab_ai", "plus.assistant_unlimited", "plus.doc_ai"} {
		assert.False(t, seen[k], "%s is retired (AI credits)", k)
	}
}

func TestResolve_Plus(t *testing.T) {
	for _, e := range Resolve(TierPlus, map[Key]int{DeepAnalysis: 40}) {
		assert.True(t, e.Allowed && e.Unlimited, e.Key)
		assert.Zero(t, e.Limit, e.Key)
	}
	assert.Equal(t, 40, find(t, Resolve(TierPlus, map[Key]int{DeepAnalysis: 40}), DeepAnalysis).Used)
}

func TestResolve_TrialGetsPlusQuotas(t *testing.T) {
	assert.Equal(t, Resolve(TierPlus, map[Key]int{DeepAnalysis: 2}), Resolve(TierTrial, map[Key]int{DeepAnalysis: 2}))
	defs := []Definition{limitedDef, fairUseDef}
	assert.Equal(t, ResolveAll(defs, TierPlus, map[Key]int{testLimited: 2}), ResolveAll(defs, TierTrial, map[Key]int{testLimited: 2}))
}

func TestResolve_LimitedQuotaSpent(t *testing.T) {
	e := ResolveOne(TierPlus, limitedDef, 10)
	assert.False(t, e.Allowed)
	assert.Equal(t, 0, e.Remaining)
	over := ResolveOne(TierPlus, limitedDef, 12) // e.g. counted before a downgrade
	assert.Equal(t, 0, over.Remaining)
	assert.False(t, over.Allowed)
}

func TestResolve_Free(t *testing.T) {
	es := Resolve(TierFree, map[Key]int{DeepAnalysis: 2})
	for _, d := range Definitions() {
		e := find(t, es, d.Key)
		assert.False(t, e.Allowed, d.Key)
		assert.False(t, e.Unlimited, d.Key)
		assert.Zero(t, e.Limit, d.Key)
	}
	assert.Equal(t, 2, find(t, es, DeepAnalysis).Used, "usage is still reported")
	assert.Equal(t, Entitlement{Key: testFairUse, Allowed: true, Limit: 5, Used: 4, Remaining: 1}, ResolveOne(TierFree, fairUseDef, 4))
	assert.False(t, ResolveOne(TierFree, fairUseDef, 5).Allowed)
}

// TestResolve_Table: every quota shape × tier at a few usage points.
func TestResolve_Table(t *testing.T) {
	type want struct {
		allowed, unlimited, fairUse bool
		limit, remaining            int
	}
	analysis, _ := Lookup(AnalysisLongRange)
	deep, _ := Lookup(DeepAnalysis)
	cases := []struct {
		def  Definition
		tier Tier
		used int
		want want
	}{
		{limitedDef, TierFree, 0, want{allowed: true, limit: 1, remaining: 1}},
		{limitedDef, TierFree, 1, want{limit: 1}},
		{limitedDef, TierPlus, 9, want{allowed: true, limit: 10, remaining: 1}},
		{limitedDef, TierPlus, 10, want{limit: 10}},
		{limitedDef, TierTrial, 0, want{allowed: true, limit: 10, remaining: 10}},
		{fairUseDef, TierFree, 4, want{allowed: true, limit: 5, remaining: 1}},
		{fairUseDef, TierFree, 5, want{limit: 5}},
		{fairUseDef, TierPlus, 0, want{allowed: true, unlimited: true, fairUse: true, limit: 300, remaining: 300}},
		{fairUseDef, TierPlus, 299, want{allowed: true, unlimited: true, fairUse: true, limit: 300, remaining: 1}},
		{fairUseDef, TierPlus, 300, want{unlimited: true, fairUse: true, limit: 300}},
		{fairUseDef, TierTrial, 301, want{unlimited: true, fairUse: true, limit: 300}},
		// D-178: the 1-year (and all-time) analysis range.
		{analysis, TierFree, 0, want{}},
		{analysis, TierPlus, 0, want{allowed: true, unlimited: true}},
		{analysis, TierTrial, 0, want{allowed: true, unlimited: true}},
		{deep, TierFree, 0, want{}},
		{deep, TierPlus, 50, want{allowed: true, unlimited: true}},
	}
	for _, tc := range cases {
		e := ResolveOne(tc.tier, tc.def, tc.used)
		got := want{allowed: e.Allowed, unlimited: e.Unlimited, fairUse: e.FairUse, limit: e.Limit, remaining: e.Remaining}
		assert.Equal(t, tc.want, got, "%s %s used=%d", tc.def.Key, tc.tier, tc.used)
		assert.Equal(t, tc.used, e.Used)
		assert.Equal(t, tc.want.fairUse && !tc.want.allowed, e.AtFairUseCap(), "%s %s used=%d", tc.def.Key, tc.tier, tc.used)
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
		if defs[i].Key == DeepAnalysis {
			defs[i].Free = Quota{Enabled: true, Limit: 3}
		}
	}
	e := find(t, ResolveAll(defs, TierFree, map[Key]int{DeepAnalysis: 2}), DeepAnalysis)
	assert.Equal(t, Entitlement{Key: DeepAnalysis, Allowed: true, Limit: 3, Used: 2, Remaining: 1}, e)
	assert.False(t, find(t, Resolve(TierFree, nil), DeepAnalysis).Allowed, "Resolve keeps the code default")
}

func TestLookup(t *testing.T) {
	d, ok := Lookup(Companion)
	require.True(t, ok)
	assert.True(t, d.QuotaFor(TierTrial).Enabled)
	assert.False(t, d.QuotaFor(TierFree).Enabled)
	_, ok = Lookup("plus.nope")
	assert.False(t, ok)
	_, ok = Lookup("plus.lab_ai")
	assert.False(t, ok, "retired AI key")
}

func TestDefinitions_IsACopy(t *testing.T) {
	ds := Definitions()
	ds[0].Plus = Quota{}
	d, _ := Lookup(ds[0].Key)
	assert.True(t, d.Plus.Enabled)
}

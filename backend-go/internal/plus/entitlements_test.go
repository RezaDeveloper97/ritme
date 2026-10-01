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
	assert.Equal(t, Entitlement{Key: AssistantUnlimited, Allowed: true, Unlimited: true, Used: 40}, find(t, es, AssistantUnlimited))
	for _, k := range []Key{DeepAnalysis, VoiceLog, PDFShare, VisitDiscount} {
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
	es := Resolve(TierFree, map[Key]int{AssistantUnlimited: 4, LabAI: 1})
	for _, k := range []Key{DeepAnalysis, VoiceLog, LabAI, PDFShare, VisitDiscount} {
		e := find(t, es, k)
		assert.False(t, e.Allowed, k)
		assert.False(t, e.Unlimited, k)
		assert.Zero(t, e.Limit, k)
	}
	assert.Equal(t, 1, find(t, es, LabAI).Used, "usage is still reported")
	a := find(t, es, AssistantUnlimited)
	assert.Equal(t, Entitlement{Key: AssistantUnlimited, Allowed: true, Limit: 5, Used: 4, Remaining: 1}, a)
	a = find(t, Resolve(TierFree, map[Key]int{AssistantUnlimited: 5}), AssistantUnlimited)
	assert.False(t, a.Allowed)
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

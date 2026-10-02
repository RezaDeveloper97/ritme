package companionhome

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/messages/registry"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// The admin registry group and this package agree on the group, the phases and the item keys.
func TestTipKeysMatchRegistry(t *testing.T) {
	assert.Equal(t, TipGroup, registry.CompanionTipGroup)
	assert.Equal(t, TipPhases, registry.CompanionTipPhases)
	assert.Equal(t, TipsPerPhase, registry.CompanionTipsPerPhase)
	assert.Equal(t, TipKeys(), registry.Keys(registry.CompanionTipGroup))
	for _, k := range TipKeys() {
		it, ok := registry.Lookup(TipGroup, k)
		require.True(t, ok, k)
		assert.True(t, it.Typed, k)
		assert.Equal(t, []string{"name"}, it.Placeholders, k)
	}
}

// Every embedded language has the empty state and every note / tip.
func TestEmbeddedCopyComplete(t *testing.T) {
	tc := &tipCopy{rows: nil}
	for _, locale := range []string{"fa", "en"} {
		tc.locale = locale
		for _, k := range []string{"someone", "empty.title", "empty.body", "empty.action"} {
			line, ok := translator().Get("companion_home."+k, locale)
			require.True(t, ok, locale+" "+k)
			assert.NotEmpty(t, line, locale+" "+k)
		}
		for _, phase := range TipPhases {
			assert.Contains(t, tc.note(phase, "X"), "X", locale+" "+phase)
			tips := tc.tips(phase, "X")
			require.Len(t, tips, TipsPerPhase, locale+" "+phase)
			for _, tip := range tips {
				assert.NotEmpty(t, tip.Title, tip.Key)
				assert.NotEmpty(t, tip.Body, tip.Key)
				assert.NotContains(t, tip.Title+tip.Body, "{name}", tip.Key)
			}
		}
	}
}

// Admin rows win per item, the request locale over the default language; an empty title hides the tip.
func TestTipCopyRows(t *testing.T) {
	tc := &tipCopy{locale: "en", deflt: "fa", rows: map[string]map[string]map[string]any{
		"luteal_tip_1": {"fa": {"title": "fa title", "body": "fa body"}},
		"luteal_tip_2": {"en": {"title": "Hi {name}", "body": ""}, "fa": {"title": "x"}},
		"luteal_tip_3": {"en": {"title": " "}},
		"luteal_note":  {"en": {"body": ""}},
	}}
	tips := tc.tips(PhaseLuteal, "Sara")
	require.Len(t, tips, 2)
	assert.Equal(t, Tip{Key: "luteal_tip_1", Title: "fa title", Body: "fa body"}, tips[0])
	assert.Equal(t, Tip{Key: "luteal_tip_2", Title: "Hi Sara", Body: ""}, tips[1])
	assert.Empty(t, tc.note(PhaseLuteal, "Sara"), "an empty admin body hides the note")
	assert.Contains(t, tc.note(PhaseMenstrual, "Sara"), "Sara", "no row → embedded copy")
}

func TestPhases(t *testing.T) {
	assert.Equal(t, PhaseMenstrual, PhaseOf(enums.MainPhaseMenstrual))
	assert.Equal(t, PhaseFollicular, PhaseOf(enums.MainPhaseFollicular))
	assert.Equal(t, PhaseFertile, PhaseOf(enums.MainPhaseFertile))
	assert.Equal(t, PhaseLuteal, PhaseOf(enums.MainPhaseLuteal))
	assert.Equal(t, PhaseLuteal, PhaseOf(enums.MainPhasePeriodExpected))
	assert.Equal(t, PhaseGeneral, PhaseOf(enums.MainPhaseUnknown))

	cycle := jsonx.Obj("has_data", true, "main_phase", enums.MainPhaseFertile)
	assert.Equal(t, PhaseGeneral, tipPhase(map[companion.Section]any{}))
	assert.Equal(t, PhaseFertile, tipPhase(map[companion.Section]any{companion.SectionCycle: cycle}))
	assert.Equal(t, PhaseGeneral, tipPhase(map[companion.Section]any{
		companion.SectionCycle: jsonx.Obj("has_data", false, "main_phase", enums.MainPhaseUnknown),
	}))
	assert.Equal(t, PhaseFertile, tipPhase(map[companion.Section]any{
		companion.SectionCycle: cycle, companion.SectionPregnancy: jsonx.Obj("is_active", false),
	}))
	assert.Equal(t, PhasePregnancy, tipPhase(map[companion.Section]any{
		companion.SectionCycle: cycle, companion.SectionPregnancy: jsonx.Obj("is_active", true),
	}))

	for phase, tag := range map[string]string{PhaseMenstrual: "menstruation", PhaseFollicular: "follicular", PhaseFertile: "ovulation", PhaseLuteal: "luteal"} {
		got, ok := legacyPhase(phase)
		assert.True(t, ok, phase)
		assert.Equal(t, tag, got, phase)
	}
	for _, phase := range []string{PhasePregnancy, PhaseGeneral, ""} {
		_, ok := legacyPhase(phase)
		assert.False(t, ok, phase)
	}
}

package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveLifeMode(t *testing.T) {
	cases := []struct {
		name     string
		stored   string
		pregnant bool
		goal     string
		want     LifeMode
	}{
		// Users created before B-N2-01 (no stored mode) resolve exactly as the legacy detection.
		{"legacy cycle", "", false, "non_ttc", LifeModeCycle},
		{"legacy ttc", "", false, "ttc", LifeModeTTC},
		{"legacy pregnancy", "", true, "non_ttc", LifeModePregnancy},
		{"no profile", "", false, "", LifeModeCycle},
		// An active pregnancy profile always wins.
		{"pregnant over menopause", "menopause", true, "non_ttc", LifeModePregnancy},
		// Modes the legacy columns cannot express.
		{"postpartum", "postpartum", false, "non_ttc", LifeModePostpartum},
		{"menopause", "menopause", false, "ttc", LifeModeMenopause},
		{"teen", "teen", false, "non_ttc", LifeModeTeen},
		// cycle / ttc follow user_goal; a stored pregnancy without an active pregnancy falls back.
		{"stored cycle, goal ttc", "cycle", false, "ttc", LifeModeTTC},
		{"stored ttc", "ttc", false, "ttc", LifeModeTTC},
		{"stored pregnancy inactive", "pregnancy", false, "non_ttc", LifeModeCycle},
		{"unknown stored value", "bogus", false, "non_ttc", LifeModeCycle},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, ResolveLifeMode(c.stored, c.pregnant, c.goal))
		})
	}
}

func TestLifeModeEngineMapping(t *testing.T) {
	assert.Equal(t, []string{"cycle", "ttc", "pregnancy", "postpartum", "menopause", "teen"}, LifeModeValues())
	want := map[LifeMode]MessageMode{
		LifeModeCycle: MessageModeCycle, LifeModeTTC: MessageModeCycle, LifeModePregnancy: MessageModePregnancy,
		LifeModePostpartum: MessageModePostpartum, LifeModeMenopause: MessageModeCycle, LifeModeTeen: MessageModeCycle,
	}
	for m, mm := range want {
		assert.True(t, m.IsValid())
		assert.Equal(t, mm, m.MessageMode(), m)
		assert.True(t, m.MessageMode().IsValid(), m)
	}
	assert.False(t, LifeMode("x").IsValid())

	for _, m := range []LifeMode{LifeModeMenopause, LifeModeTeen} {
		assert.False(t, m.AllowsFertilityContent(), m)
	}
	for _, m := range []LifeMode{LifeModeCycle, LifeModeTTC, LifeModePregnancy, LifeModePostpartum} {
		assert.True(t, m.AllowsFertilityContent(), m)
	}
	assert.False(t, LifeModeMenopause.TracksCycle())
	for _, m := range []LifeMode{LifeModeCycle, LifeModeTTC, LifeModePregnancy, LifeModePostpartum, LifeModeTeen} {
		assert.True(t, m.TracksCycle(), m)
	}
	assert.Equal(t, UserGoalTtc, LifeModeTTC.LegacyUserGoal())
	for _, m := range []LifeMode{LifeModeCycle, LifeModePregnancy, LifeModePostpartum, LifeModeMenopause, LifeModeTeen} {
		assert.Equal(t, UserGoalNonTtc, m.LegacyUserGoal(), m)
	}
}

func TestOnboardingEnumValues(t *testing.T) {
	// Every goal is a life mode (the Goal step stores it as the mode).
	for _, g := range OnboardingGoalValues() {
		assert.True(t, LifeMode(g).IsValid(), g)
	}
	assert.Equal(t, []string{"female", "male"}, GenderValues())
	assert.Equal(t, []string{"peri", "meno", "post", "unsure"}, MenopauseStageValues())
	assert.Len(t, ChronicIllnessValues(), 7)
	assert.Len(t, GynConditionValues(), 5)
	assert.Len(t, MedicationValues(), 3)
}

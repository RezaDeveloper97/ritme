package shared

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
)

func entry(cat, param, item, code string) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Item: item, Code: sql.NullString{String: code, Valid: code != ""}}
}

func TestSharedEntry_CycleModeOnly(t *testing.T) {
	cycle := []string{taxonomy.ModeCycle}
	for _, e := range []taxonomy.Entry{
		entry("mood", "moods", "calm", "1"),
		entry("pain", "location", "abdomen", "mild"),
		entry("pain", "none", "", "yes"),
		entry("sleep", "quality", "", "good"),
		entry("appetite_energy", "energy", "", "very_low"), // legacy value of a cycle param
		entry("mood", "moods", taxonomy.CustomItemPrefix+"12", "1"),
	} {
		assert.True(t, sharedEntry(e, cycle), e.Slot())
	}
	for _, e := range []taxonomy.Entry{
		entry("symptoms", "general", "leg_cramps", "mild"),
		entry("pain", "location", "leg", "mild"),
		entry("pain", "location", "stitches", "mild"),
		entry("mood", "weekly_checkin", "", ""),
		entry("symptoms", "general", "night_sweats", "yes"),
		entry("sleep", "hours", "", ""),                      // postpartum param
		entry("pain", "location", "elbow", "mild"),           // unknown item
		entry("mood", "diary", "", ""),                       // unknown param
		entry("bank", "pin", "", ""),                         // unknown category
		entry("mood", "moods", "user_custom_free_text", "1"), // not a custom item code
	} {
		assert.False(t, sharedEntry(e, cycle), e.Slot())
	}
	preg := []string{taxonomy.ModeCycle, taxonomy.ModePregnancy}
	assert.True(t, sharedEntry(entry("symptoms", "general", "leg_cramps", "mild"), preg))
	assert.False(t, sharedEntry(entry("pain", "location", "stitches", "mild"), preg))
}

func TestScopeAndPhase(t *testing.T) {
	for _, tc := range []struct {
		mode    enums.LifeMode
		preg    bool
		neutral bool
		modes   []string
	}{
		{enums.LifeModeCycle, false, false, []string{"cycle"}},
		{enums.LifeModeTeen, false, false, []string{"cycle"}},
		{enums.LifeModePregnancy, false, true, []string{"cycle"}},
		{enums.LifeModePregnancy, true, false, []string{"cycle", "pregnancy"}},
		{enums.LifeModePostpartum, false, true, []string{"cycle"}},
		{enums.LifeModePostpartum, true, false, []string{"cycle", "postpartum"}},
		{enums.LifeModeMenopause, false, true, []string{"cycle"}},
		{enums.LifeModeMenopause, true, true, []string{"cycle"}},
	} {
		v := viewScope{mode: tc.mode, pregnancy: tc.preg}
		assert.Equal(t, tc.neutral, v.neutralCycle(), "%s/%v", tc.mode, tc.preg)
		assert.Equal(t, tc.modes, v.symptomModes(), "%s/%v", tc.mode, tc.preg)
	}
	assert.Equal(t, enums.MainPhaseFollicular, sharedPhase(enums.MainPhaseFertile, true), "teen: no fertile window")
	assert.Equal(t, enums.MainPhaseFertile, sharedPhase(enums.MainPhaseFertile, false))
	assert.Equal(t, enums.MainPhaseLuteal, sharedPhase(enums.MainPhaseLuteal, true))
	n := func(i int) *int { return &i }
	assert.False(t, tooLate(nil))
	assert.False(t, tooLate(n(MaxLateDays)))
	assert.True(t, tooLate(n(MaxLateDays+1)))
}

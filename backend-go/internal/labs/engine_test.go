package labs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M9-37: the v6 phase is the engine's content phase of the lab date; the prompt names it as the lab-date phase.
func TestPhaseOfDay_LabDateNotToday(t *testing.T) {
	d := func(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }
	var in engine.Input
	for _, s := range []string{"2026-06-20", "2026-07-18", "2026-08-15", "2026-09-12", "2026-10-03"} {
		in.Periods = append(in.Periods, engine.Period{Start: d(s), End: d(s) + 4})
	}
	today := civildate.MustParse("2026-10-06")
	st := engine.Prepare(in, engineparams.For(today))
	assert.Equal(t, "luteal", phaseOfDay(st, civildate.MustParse("2026-07-10"), today))
	assert.Equal(t, "menstruation", phaseOfDay(st, today, today))
	assert.Empty(t, phaseOfDay(st, civildate.MustParse("2026-01-01"), today), "before any data: unknown")

	in.Hormonal = true // the pill: no natural phase outside the bleed
	st = engine.Prepare(in, engineparams.For(today))
	assert.Empty(t, phaseOfDay(st, civildate.MustParse("2026-07-10"), today))
}

func TestPrompt_LabDatePhaseKey(t *testing.T) {
	uc := UserContext{Mode: enums.LifeModeCycle, Phase: "luteal", PhaseOn: civildate.MustParse("2026-07-10")}
	text := buildPrompt(nil, uc, "en", nil).Messages[0].Text
	assert.Contains(t, text, `"approximate_cycle_phase_on_lab_date": "luteal"`)
	assert.NotContains(t, text, "approximate_cycle_phase_today")

	uc.PhaseOn = civildate.Date{} // v1: today's estimate keeps its key
	text = buildPrompt(nil, uc, "en", nil).Messages[0].Text
	assert.Contains(t, text, `"approximate_cycle_phase_today": "luteal"`)
}

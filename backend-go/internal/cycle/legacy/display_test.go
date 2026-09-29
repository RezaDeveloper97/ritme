package legacy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestDisplayWindow(t *testing.T) {
	for _, tc := range []struct {
		name        string
		phase       enums.CyclePhase
		fertile     bool
		day         int
		wantPhase   enums.CyclePhase
		wantFertile bool
	}{
		{"O+1 is luteal, outside the window", enums.CyclePhaseOvulation, true, 16, enums.CyclePhaseLuteal, false},
		{"ovulation day stays in the window", enums.CyclePhaseOvulation, true, 15, enums.CyclePhaseOvulation, true},
		{"fertile ramp unchanged", enums.CyclePhaseFollicular, true, 11, enums.CyclePhaseFollicular, true},
		{"menstruation override kept", enums.CyclePhaseMenstruation, false, 10, enums.CyclePhaseMenstruation, false},
		{"luteal unchanged", enums.CyclePhaseLuteal, false, 20, enums.CyclePhaseLuteal, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			phase, fertile := DisplayWindow(tc.phase, tc.fertile, tc.day, 15)
			assert.Equal(t, tc.wantPhase, phase)
			assert.Equal(t, tc.wantFertile, fertile)
		})
	}
}

// phaseTipSource returns one tip named after the phase it is asked for (the admin content path).
type phaseTipSource struct {
	asked    []enums.CyclePhase
	askedSub []enums.CycleSubphase
}

func (*phaseTipSource) HasContent(context.Context) (bool, error) { return true, nil }
func (s *phaseTipSource) ForDay(_ context.Context, phase enums.CyclePhase, sub enums.CycleSubphase, _ enums.TriggerLog) ([]recommendation.Tip, error) {
	s.asked = append(s.asked, phase)
	s.askedSub = append(s.askedSub, sub)
	if phase == enums.CyclePhaseOvulation {
		return []recommendation.Tip{{Type: "fertility", EN: "Peak fertility days.", FA: "روزهای اوج باروری."}}, nil
	}
	return []recommendation.Tip{{Type: "nutrition", EN: string(phase), FA: string(phase)}}, nil
}

// F-1 / D-30 (T-M2-36): /cycle/today's calculation on the legacy O + 1 day is luteal and not
// fertile, and its text flags and daily tips carry no fertile copy — for an avoiding and a trying
// user alike (the engine does not read the goal), with the built-in and the admin tips. The tips are the
// early-luteal ones (luteal + early_luteal) while the emitted sub-phase stays post_ovulation.
func TestCalculateForDisplay_OPlusOne(t *testing.T) {
	ctx := context.Background()
	lmp := civildate.New(2026, 9, 1) // 28-day cycle, O = day 15 = 09-15
	oPlus1, oDay := lmp.AddDays(15), lmp.AddDays(14)

	for _, goal := range []enums.UserGoal{enums.UserGoalNonTtc, enums.UserGoalTtc} {
		profile := &model.Profile{LastPeriodStart: lmp, CycleDuration: model.Int(28), PeriodDuration: model.Int(5), Goal: string(goal)}

		t.Run(string(goal)+"/built-in tips", func(t *testing.T) {
			e := New(Input{Profile: profile, Tips: noTips{}, Today: oPlus1})

			legacyCalc, err := e.CalculateForDate(ctx, oPlus1, true)
			require.NoError(t, err)
			require.Equal(t, 16, legacyCalc.Day)
			assert.Equal(t, enums.CyclePhaseOvulation, legacyCalc.Phase, "the legacy (month, home) path is unchanged")
			assert.True(t, legacyCalc.IsFertileWindow)

			c, err := e.CalculateForDisplay(ctx, oPlus1)
			require.NoError(t, err)
			assert.Equal(t, 16, c.Day)
			assert.Equal(t, 15, c.OvulationDay)
			assert.Equal(t, enums.CyclePhaseLuteal, c.Phase)
			assert.Equal(t, enums.CycleSubphasePostOvulation, c.CurrentSubphase, "sub-phase untouched")
			assert.False(t, c.IsFertileWindow)
			assert.Equal(t, legacyCalc.FinalProbability, c.FinalProbability, "scores untouched")

			assert.Equal(t, phaseTips(enums.CyclePhaseLuteal, enums.CycleSubphaseEarlyLuteal), c.DailyTips)
			assert.NotEmpty(t, c.DailyTips)
			for _, tip := range c.DailyTips {
				assert.NotEqual(t, "fertility", tip.Type)
				assert.NotContains(t, tip.FA, "باروری")
				assert.NotContains(t, tip.EN, "fertil")
			}
			for _, f := range c.TextFlags {
				assert.NotEqual(t, "fertility_status", f.Key)
			}
			flags := map[string]any{}
			for _, f := range c.TextFlags {
				flags[f.Key] = f.EN
			}
			assert.Equal(t, "Current phase: "+enums.CyclePhaseLuteal.Label("en")+" ("+enums.CycleSubphasePostOvulation.Label("en")+")", flags["phase_info"])

			// The ovulation day itself is still in the display window.
			o, err := e.CalculateForDisplay(ctx, oDay)
			require.NoError(t, err)
			want, err := e.CalculateForDate(ctx, oDay, true)
			require.NoError(t, err)
			assert.Equal(t, want, o)
			assert.Equal(t, enums.CyclePhaseOvulation, o.Phase)
			assert.True(t, o.IsFertileWindow)
		})

		t.Run(string(goal)+"/admin tips", func(t *testing.T) {
			src := &phaseTipSource{}
			e := New(Input{Profile: profile, Tips: src, Today: oPlus1})
			c, err := e.CalculateForDisplay(ctx, oPlus1)
			require.NoError(t, err)
			assert.Equal(t, []enums.CyclePhase{enums.CyclePhaseLuteal}, src.asked, "tips looked up for the display phase")
			assert.Equal(t, []enums.CycleSubphase{enums.CycleSubphaseEarlyLuteal}, src.askedSub, "O + 1 tips are the early-luteal ones")
			assert.Equal(t, enums.CycleSubphasePostOvulation, c.CurrentSubphase, "emitted sub-phase unchanged")
			require.Len(t, c.DailyTips, 1)
			assert.Equal(t, "nutrition", c.DailyTips[0].Type)
		})
	}
}

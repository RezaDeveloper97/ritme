package view

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/enums"
)

func day(t *testing.T, s string) engine.Day {
	t.Helper()
	tm, err := time.Parse(time.DateOnly, s)
	require.NoError(t, err)
	d, ok := engine.MakeDay(tm.Year(), tm.Month(), tm.Day())
	require.True(t, ok)
	return d
}

// regular: four logged 5-day periods every 29 days (the §2.35 example: last start 07-01, next 07-30, O = 07-16).
func regular(t *testing.T) engine.Input {
	t.Helper()
	var in engine.Input
	for _, s := range []string{"2026-04-04", "2026-05-03", "2026-06-01"} {
		st := day(t, s)
		in.Periods = append(in.Periods, engine.Period{Start: st, End: st + 4})
	}
	in.Periods = append(in.Periods, engine.Period{Start: day(t, "2026-07-01"), End: day(t, "2026-07-05")})
	in.Declared = engine.Declared{CycleLength: engine.Some(30), PeriodLength: engine.Some(6)}
	return in
}

func legacyOn(t *testing.T, in engine.Input, target, today string) (Legacy, engine.Status) {
	t.Helper()
	st := engine.Prepare(in, engine.DefaultParams())
	s := st.Resolve(day(t, target), day(t, today))
	ui := st.TodayUI(s)
	return LegacyFromV6(V6Input{State: st, Status: &s, UI: &ui, Today: day(t, today), Locale: "en"}), s
}

// I-11 (DECISIONS.md): v6 subphase → legacy content phase.
func TestContentPhase_I11(t *testing.T) {
	cases := map[engine.Subphase]enums.CyclePhase{
		engine.SubphaseMenstrual:                enums.CyclePhaseMenstruation,
		engine.SubphaseMenstrualPossible:        enums.CyclePhaseMenstruation,
		engine.SubphaseEarlyFollicular:          enums.CyclePhaseFollicular,
		engine.SubphaseLateFollicularTransition: enums.CyclePhaseFollicular,
		engine.SubphaseFertileRising:            enums.CyclePhaseFollicular,
		engine.SubphaseHighFertility:            enums.CyclePhaseOvulation,
		engine.SubphaseOvulationLikely:          enums.CyclePhaseOvulation,
		engine.SubphasePostOvulation:            enums.CyclePhaseLuteal,
		engine.SubphasePMSPossible:              enums.CyclePhaseLuteal,
		engine.SubphasePeriodExpected:           enums.CyclePhaseLuteal,
	}
	for sub, want := range cases {
		got, ok := ContentPhase(sub)
		assert.True(t, ok, sub.String())
		assert.Equal(t, want, got, sub.String())
	}
	_, ok := ContentPhase(engine.SubphaseUnknown)
	assert.False(t, ok, "unknown → no phase content")
}

// §2.35 example day (07-18, early luteal): the predictions block is the prediction card (period next..next+EPL−1,
// display window O−5..O — not O+1), the one confidence, the three value layers.
func TestLegacyFromV6_Spec235Day(t *testing.T) {
	l, s := legacyOn(t, regular(t), "2026-07-18", "2026-07-18")
	require.Equal(t, engine.SubphaseEarlyLuteal, s.Subphase)
	require.NotNil(t, l.Phase)
	assert.Equal(t, enums.CyclePhaseLuteal, *l.Phase)
	require.NotNil(t, l.Predictions)
	p := l.Predictions
	assert.Equal(t, "2026-07-30", p.NextPeriodStart.String())
	assert.Equal(t, "2026-08-03", p.NextPeriodEnd.String())
	assert.Equal(t, "2026-07-16", p.EstimatedOvulationDate.String())
	assert.Equal(t, "2026-07-11", p.FertileWindowStart.String())
	assert.Equal(t, "2026-07-16", p.FertileWindowEnd.String(), "display window ends on O (X-06), not O+1")
	assert.Equal(t, enums.ConfidenceLevel(s.Confidence.String()), p.Confidence)
	assert.Equal(t, p.Confidence, l.DataQualityDetails.Confidence, "one confidence everywhere (X-05)")
	assert.Equal(t, enums.EffectiveSourceRecentValidCycles, l.EffectiveValues.Source)
	assert.Equal(t, 29, l.EffectiveValues.CycleLength)
	require.NotNil(t, l.ProfileValues.CycleLength)
	assert.Equal(t, 30, *l.ProfileValues.CycleLength, "declared value")
	require.NotNil(t, l.CalculatedValues.BasedOnCycles)
	assert.Equal(t, 3, *l.CalculatedValues.BasedOnCycles)
	assert.Equal(t, enums.RegularityStatusRelativelyRegular, l.DataQualityDetails.RegularityStatus)
	require.NotNil(t, l.DailyCard)
	assert.Equal(t, enums.DataStatusPredicted, *l.DataStatus)
	assert.Equal(t, enums.FertilityLevelLow, l.DailyCard.FertilityLevel)
}

func TestDailyCardV6_Cases(t *testing.T) {
	t.Run("closed logged period day is actual", func(t *testing.T) {
		l, _ := legacyOn(t, regular(t), "2026-07-03", "2026-07-18")
		require.NotNil(t, l.DailyCard)
		assert.Equal(t, enums.DataStatusActual, l.DailyCard.DataStatus)
		assert.Equal(t, "Day 3 of your period", l.DailyCard.Title)
	})
	t.Run("open period asks for the end once the ring says end", func(t *testing.T) {
		in := regular(t)
		in.Periods[len(in.Periods)-1].End = engine.NoDay
		l, _ := legacyOn(t, in, "2026-07-05", "2026-07-05")
		require.NotNil(t, l.DailyCard)
		assert.Equal(t, enums.DataStatusIncomplete, l.DailyCard.DataStatus)
		assert.Equal(t, "log_period_end", l.DailyCard.PrimaryAction.Type)
	})
	t.Run("period expected: due, then overdue, fertility unknown", func(t *testing.T) {
		l, s := legacyOn(t, regular(t), "2026-07-30", "2026-07-30")
		require.Equal(t, engine.MainPhasePeriodExpected, s.MainPhase)
		require.NotNil(t, l.DailyCard)
		assert.Equal(t, "confirm_period_start", l.DailyCard.PrimaryAction.Type)
		assert.Equal(t, enums.FertilityLevelUnknown, l.DailyCard.FertilityLevel)
		l, _ = legacyOn(t, regular(t), "2026-08-01", "2026-08-01")
		assert.Equal(t, "2 day(s) past the predicted date", l.DailyCard.Title)
		assert.Equal(t, enums.FertilityLevelUnknown, l.DailyCard.FertilityLevel)
	})
	t.Run("hormonal contraception: a plain cycle-day card, no fertile window", func(t *testing.T) {
		in := regular(t)
		in.Hormonal = true
		l, s := legacyOn(t, in, "2026-07-14", "2026-07-14")
		require.True(t, s.Limits.FertilitySuppressed)
		require.NotNil(t, l.DailyCard)
		assert.Equal(t, enums.DataStatusPredicted, l.DailyCard.DataStatus)
		assert.NotContains(t, l.DailyCard.Title, "fertile")
		assert.Equal(t, enums.FertilityLevelUnknown, l.DailyCard.FertilityLevel)
		require.NotNil(t, l.Predictions)
		assert.True(t, l.Predictions.FertileWindowStart.IsZero(), "no fertile window under suppression")
	})
	// T-M9-53 QA persona: birth 2026-10-20, breastfeeding, eight days later. No cycle runs before cycles return
	// (E-14): no cycle day, no «یک روز معمول چرخه» card, no predictions; a pre-pregnancy logged day keeps its card.
	t.Run("postpartum before cycles return: no cycle-day card", func(t *testing.T) {
		in := regular(t)
		in.Mode, in.Breastfeeding, in.DeliveryDate = engine.ModePostpartum, true, engine.Some(day(t, "2026-10-20"))
		for _, loc := range []string{"en", "fa"} {
			st := engine.Prepare(in, engine.DefaultParams())
			today := day(t, "2026-10-28")
			s, ui := st.Today(today)
			l := LegacyFromV6(V6Input{State: st, Status: &s, UI: &ui, Today: today, Locale: loc})
			require.Equal(t, engine.SuppressionPostpartumNotReturned, s.Limits.SuppressionReason)
			assert.False(t, s.CycleDay.OK())
			assert.Empty(t, s.FutureCycles)
			assert.Nil(t, l.DailyCard, loc)
			assert.Nil(t, l.DataStatus, loc)
			assert.Nil(t, l.Phase, loc)
			assert.Nil(t, l.Predictions, loc)
			// Every day from the birth on: no card (no cycle day, no usual-day copy).
			for d := day(t, "2026-10-20"); d <= today+40; d++ {
				ds := st.Resolve(d, today)
				dui := st.TodayUI(ds)
				assert.Nil(t, DailyCardV6(CardV6Input{Status: &ds, UI: &dui, Today: today, Locale: loc}), "%s %s", loc, d)
			}
		}
		l, _ := legacyOn(t, in, "2026-07-03", "2026-10-28")
		require.NotNil(t, l.DailyCard, "logged history keeps its card")
		assert.Equal(t, "Day 3 of your period", l.DailyCard.Title)
	})
	t.Run("teen keeps the numbers but drops the fertility copy", func(t *testing.T) {
		in := regular(t)
		in.Mode = engine.ModeTeen
		l, s := legacyOn(t, in, "2026-07-15", "2026-07-15")
		require.Equal(t, engine.MainPhaseFertile, s.MainPhase)
		require.NotNil(t, l.DailyCard)
		assert.NotContains(t, l.DailyCard.Title, "fertile")
		assert.Equal(t, enums.FertilityLevelUnknown, l.DailyCard.FertilityLevel)
	})
	t.Run("no data: no card, no predictions", func(t *testing.T) {
		l, _ := legacyOn(t, engine.Input{}, "2026-07-15", "2026-07-15")
		assert.Nil(t, l.DailyCard)
		assert.Nil(t, l.DataStatus)
		assert.Nil(t, l.Phase)
		assert.Nil(t, l.Predictions)
	})
}

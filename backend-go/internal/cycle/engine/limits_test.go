package engine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pill is a regular 28-day user (current cycle from 2026-07-01) on a combined pill (I-03 → Hormonal).
func pill() Input {
	in := regular(day(julyStart), 28, 5)
	in.Hormonal = true
	return in
}

// assertSuppressedFertility is §2.40 rule 9 / §2.26 / §2.34 for one suppressed status.
func assertSuppressedFertility(t *testing.T, st Status, msg string) {
	t.Helper()
	assert.True(t, st.Limits.FertilitySuppressed, msg)
	assert.NotEqual(t, MainPhaseFertile, st.MainPhase, msg)
	assert.Equal(t, FertilityUnknown, st.FertilityLevel, msg)
	assert.False(t, st.DaysToOvulation.OK(), msg)
	assert.Equal(t, NoDay, st.Anchors.EstimatedOvulationDate, msg)
	assert.Equal(t, OvulationUnknown, st.Anchors.OvulationSource, msg)
	assert.False(t, st.SuggestPregnancyTest, msg)
	assert.False(t, st.Warnings.Has(WarnLHBBTConflict), msg)
}

// §6.1 pill (E-13).
func TestLimits_HormonalContraception(t *testing.T) {
	cur := day(julyStart)
	s := prep(pill())

	t.Run("cycle day 14 → unknown/unknown/unknown", func(t *testing.T) {
		target := cur + 13
		st := s.Resolve(target, target)
		assert.Equal(t, Some(14), st.CycleDay)
		assert.Equal(t, MainPhaseUnknown, st.MainPhase)
		assert.Equal(t, SubphaseUnknown, st.Subphase)
		assertSuppressedFertility(t, st, "day 14")
		assert.Equal(t, ResolutionPrediction, st.ResolutionSource)
		assert.True(t, st.Warnings.Has(WarnHormonalContraceptionActive))
		assert.False(t, st.Warnings.Has(WarnInsufficientAnchorData), "suppression is not insufficient data")
		assert.True(t, st.ConfidenceReasons.Has(ReasonHormonalContraceptionActive))
		assert.Equal(t, ConfidenceLow, st.Confidence)
		assert.False(t, st.RequiresUserInput)
		assert.Equal(t, Limits{FertilitySuppressed: true, SuppressionReason: SuppressionHormonalContraception},
			st.Limits)
		assert.Equal(t, cur+28, st.Anchors.PredictedNextPeriodStart, "the withdrawal bleed is still predicted")
		assert.Equal(t, Some(15), st.DaysToPeriod)
	})

	t.Run("every non-bleeding day of the cycle is unknown", func(t *testing.T) {
		for d := cur + 5; d < cur+28; d++ {
			st := s.Resolve(d, d)
			assert.Equal(t, MainPhaseUnknown, st.MainPhase, "%s", d)
			assertSuppressedFertility(t, st, d.String())
		}
	})

	t.Run("logged withdrawal bleed → menstrual", func(t *testing.T) {
		in := pill()
		in.Periods = append(in.Periods, Period{Start: cur + 28, End: cur + 31})
		st := prep(in).Resolve(cur+29, cur+29)
		assert.Equal(t, SubphaseMenstrual, st.Subphase)
		assert.Equal(t, ResolutionUserLogged, st.ResolutionSource)
		assertSuppressedFertility(t, st, "bleed")
		assert.Equal(t, ConfidenceLow, st.Confidence)
		assert.Equal(t, SubphaseMenstrual, s.Resolve(cur+2, cur+2).Subphase, "the current period")
	})

	t.Run("3 days after predicted_next without a bleed → period_expected, no test", func(t *testing.T) {
		for _, mode := range []Mode{ModeCycle, ModeTTC} {
			in := pill()
			in.Mode = mode
			target := cur + 28 + 3
			st := prep(in).Resolve(target, target)
			assert.Equal(t, SubphasePeriodExpected, st.Subphase, mode.String())
			assert.Equal(t, Some(3), st.DaysLate)
			assert.False(t, st.SuggestPregnancyTest, mode.String())
			assertSuppressedFertility(t, st, mode.String())
		}
		st := s.Resolve(cur+28+9, cur+28+9)
		assert.Equal(t, SubphasePeriodExpected, st.Subphase)
		assert.True(t, st.Warnings.Has(WarnPredictedPeriodOverdue), "the §2.22 tiers stay")
	})

	t.Run("BBT/LH markers are ignored: next stays calendar", func(t *testing.T) {
		in := pill()
		in.BBT = shiftAt(cur + 10)
		in.LH = []LHTest{lhOn(cur+15, true)} // conflicts with the BBT shift
		st := prep(in).Resolve(cur+12, cur+12)
		assert.Equal(t, cur+28, st.Anchors.PredictedNextPeriodStart)
		assertSuppressedFertility(t, st, "markers")
	})

	t.Run("completed past cycles: no fertile window either", func(t *testing.T) {
		for d := cur - 56; d < cur; d++ {
			st := s.Resolve(d, cur+10)
			assertSuppressedFertility(t, st, d.String())
			assert.NotEqual(t, MainPhasePeriodExpected, st.MainPhase, "E-22")
		}
	})

	t.Run("declared onboarding start keeps its assumed bleed", func(t *testing.T) {
		in := declared(Declared{LastPeriodStart: Some(cur), CycleLength: Some(28), PeriodLength: Some(5)})
		in.Hormonal = true
		ps := prep(in)
		assert.Equal(t, MainPhaseMenstrual, ps.Resolve(cur+1, cur+1).MainPhase)
		st := ps.Resolve(cur+12, cur+12)
		assert.Equal(t, MainPhaseUnknown, st.MainPhase)
		assert.Equal(t, ResolutionOnboardingBased, st.ResolutionSource, "E-10")
		assert.Equal(t, ConfidenceLow, st.Confidence)
	})
}

// A copper IUD is not hormonal (I-03): the output is the normal one.
func TestLimits_CopperIUDIsNormal(t *testing.T) {
	cur := day(julyStart)
	in := regular(cur, 28, 5)
	in.Hormonal = false
	normal := prep(regular(cur, 28, 5))
	s := prep(in)
	for d := cur - 30; d <= cur+45; d++ {
		st := s.Resolve(d, cur+10)
		assert.Equal(t, normal.Resolve(d, cur+10), st, "%s", d)
		assert.Zero(t, st.Limits, "%s", d)
	}
	assert.Equal(t, MainPhaseFertile, s.Resolve(cur+13, cur+13).MainPhase)
}

// E-25: pregnancy mode suspends the engine; logged history stays.
func TestLimits_PregnancyMode(t *testing.T) {
	last := day("2026-03-01") // the last period before the pregnancy
	in := regular(last, 28, 5)
	in.Mode = ModePregnancy
	in.Hormonal, in.Breastfeeding = true, true // pregnancy wins over every other condition
	s := prep(in)
	today := day("2026-07-15")

	st := s.Resolve(today, today)
	assert.Equal(t, MainPhaseUnknown, st.MainPhase)
	assert.Equal(t, SubphaseUnknown, st.Subphase)
	assert.Equal(t, ResolutionUnknown, st.ResolutionSource)
	assert.Equal(t, ConfidenceUnknown, st.Confidence)
	assert.Zero(t, st.ConfidenceReasons)
	assert.Zero(t, st.Warnings)
	assert.False(t, st.RequiresUserInput)
	assert.Equal(t, DataQualityInsufficient, st.DataQuality)
	assert.Equal(t, Limits{FertilitySuppressed: true, SuppressionReason: SuppressionPregnancy, EngineSuspended: true},
		st.Limits)
	assert.Equal(t, NoDay, st.Anchors.PredictedNextPeriodStart)
	assert.False(t, st.CycleDay.OK())
	assert.False(t, st.DaysToPeriod.OK())
	assert.False(t, st.DaysLate.OK())
	assertSuppressedFertility(t, st, "today")
	assert.Equal(t, st.Limits, s.Suppression(today))

	// A period logged before the pregnancy still resolves menstrual on its date (history views).
	st = s.Resolve(last+2, today)
	assert.Equal(t, MainPhaseMenstrual, st.MainPhase)
	assert.Equal(t, ResolutionUserLogged, st.ResolutionSource)
	assert.Equal(t, Some(3), st.CycleDay)
	assert.Equal(t, ConfidenceUnknown, st.Confidence)
	assert.True(t, st.Limits.EngineSuspended)
	assertSuppressedFertility(t, st, "history")
	// Other history days: nothing predicted.
	assert.Equal(t, MainPhaseUnknown, s.Resolve(last-14, today).MainPhase)

	// No data at all: still a complete suspended status, nothing to ask.
	none := prep(Input{Mode: ModePregnancy}).Resolve(today, today)
	assert.Zero(t, none.Warnings)
	assert.False(t, none.RequiresUserInput)
	assert.True(t, none.Limits.EngineSuspended)

	// An open period past the caps asks nothing while suspended.
	open := regular(last, 28, 5)
	open.Periods = append(open.Periods, Period{Start: last + 28, End: NoDay})
	open.Mode = ModePregnancy
	assert.False(t, prep(open).Resolve(today, today).RequiresUserInput)
}

// §6.1 postpartum without a period (E-14).
func TestLimits_Postpartum(t *testing.T) {
	delivery := day("2026-08-01")
	base := func() Input {
		in := none()
		in.Mode, in.Breastfeeding, in.DeliveryDate = ModePostpartum, true, Some(delivery)
		return in
	}
	first := Period{Start: day("2026-10-10"), End: day("2026-10-14")}
	second := Period{Start: day("2026-11-08"), End: day("2026-11-12")}

	t.Run("no period → unknown, warning, suppressed, confidence unknown", func(t *testing.T) {
		today := day("2026-09-15")
		st := prep(base()).Resolve(today, today)
		assert.Equal(t, MainPhaseUnknown, st.MainPhase)
		assert.True(t, st.Warnings.Has(WarnPostpartumCyclesNotReturned))
		assert.False(t, st.Warnings.Has(WarnInsufficientAnchorData))
		assert.False(t, st.RequiresUserInput)
		assert.Equal(t, ConfidenceUnknown, st.Confidence)
		assert.Equal(t, Limits{FertilitySuppressed: true, SuppressionReason: SuppressionPostpartumNotReturned},
			st.Limits)
		assertSuppressedFertility(t, st, "no period")

		// With pre-pregnancy history the last old period is the anchor, but nothing is predicted from it.
		in := base()
		in.Periods = periodsEnding(day("2025-10-20"), 5, 28, 28)
		st = prep(in).Resolve(today, today)
		assert.Equal(t, MainPhaseUnknown, st.MainPhase)
		assert.Equal(t, ResolutionUnknown, st.ResolutionSource)
		assert.Equal(t, NoDay, st.Anchors.PredictedNextPeriodStart)
		assert.False(t, st.DaysToPeriod.OK())
		assert.False(t, st.DaysLate.OK())
		assert.Equal(t, ConfidenceUnknown, st.Confidence, "periods before the delivery do not count")
		assert.False(t, st.RequiresUserInput)
		assert.False(t, st.CycleDay.OK(), "T-M9-53: no cycle day counted from a pre-delivery start")
		assertSuppressedFertility(t, st, "old history")
	})

	// T-M9-53 QA persona: birth 2026-10-20, breastfeeding, the last period before the pregnancy on record. Eight days
	// after the birth there is no cycle to count: no cycle day, nothing predicted, no future cycles, no today ring
	// prompt; the period logged before the pregnancy still resolves menstrual (history, with its own day).
	t.Run("QA persona: eight days after birth → no cycle day", func(t *testing.T) {
		in := none()
		in.Mode, in.Breastfeeding, in.DeliveryDate = ModePostpartum, true, Some(day("2026-10-20"))
		in.Periods = periodsEnding(day("2026-01-13"), 5, 28, 28)
		s := prep(in)
		today := day("2026-10-28")
		st, ui := s.Today(today)
		assert.Equal(t, SuppressionPostpartumNotReturned, st.Limits.SuppressionReason)
		assert.True(t, st.Warnings.Has(WarnPostpartumCyclesNotReturned))
		assert.Equal(t, MainPhaseUnknown, st.MainPhase)
		assert.False(t, st.CycleDay.OK())
		assert.False(t, st.DaysToPeriod.OK())
		assert.False(t, st.DaysLate.OK())
		assert.False(t, st.DaysToOvulation.OK())
		assert.Equal(t, NoDay, st.Anchors.PredictedNextPeriodStart)
		assert.Empty(t, st.FutureCycles)
		assert.Equal(t, RingEdit, ui.RingAction)
		assert.False(t, ui.PredictionCard.OK())
		for d := day("2026-10-20"); d <= today+60; d++ {
			got := s.Resolve(d, today)
			assert.False(t, got.CycleDay.OK(), "%s: no cycle day", d)
			assert.NotEqual(t, MainPhaseMenstrual, got.MainPhase, "%s: no predicted period", d)
		}
		old := s.Resolve(day("2026-01-15"), today)
		assert.Equal(t, MainPhaseMenstrual, old.MainPhase)
		assert.Equal(t, Some(3), old.CycleDay)
	})

	t.Run("one period → that period menstrual, other days unknown, confidence low", func(t *testing.T) {
		in := base()
		in.Periods = []Period{first}
		s := prep(in)
		today := day("2026-10-25")
		st := s.Resolve(day("2026-10-12"), today)
		assert.Equal(t, MainPhaseMenstrual, st.MainPhase)
		assert.Equal(t, ResolutionUserLogged, st.ResolutionSource)
		assert.Equal(t, ConfidenceLow, st.Confidence)
		assertSuppressedFertility(t, st, "first period")

		st = s.Resolve(today, today)
		assert.Equal(t, MainPhaseUnknown, st.MainPhase)
		assert.Equal(t, ConfidenceLow, st.Confidence)
		assert.False(t, st.CycleDay.OK(), "T-M9-53: cycles have not returned yet")
		assert.Equal(t, Some(3), s.Resolve(day("2026-10-12"), today).CycleDay, "logged bleeding keeps its day")
		assert.Equal(t, NoDay, st.Anchors.PredictedNextPeriodStart)
		assert.False(t, st.DaysToPeriod.OK())
		assert.True(t, st.Warnings.Has(WarnPostpartumCyclesNotReturned))
		assertSuppressedFertility(t, st, "after the first period")
		for d := first.Start; d < first.Start+60; d++ {
			st := s.Resolve(d, d)
			assert.NotEqual(t, MainPhasePeriodExpected, st.MainPhase, "%s: nothing is predicted", d)
			assertSuppressedFertility(t, st, d.String())
		}
		// Before the first period the history day is still unknown-confidence.
		assert.Equal(t, ConfidenceUnknown, s.Resolve(day("2026-09-01"), today).Confidence)
	})

	t.Run("second period → normal prediction from the second start", func(t *testing.T) {
		in := base()
		in.Periods = []Period{first, second}
		s := prep(in)
		today := day("2026-11-20")
		st := s.Resolve(today, today)
		assert.Zero(t, st.Limits)
		assert.False(t, st.Warnings.Has(WarnPostpartumCyclesNotReturned))
		assert.Equal(t, second.Start, st.Anchors.CurrentPeriodStart)
		assert.Equal(t, second.Start+29, st.Anchors.PredictedNextPeriodStart, "one 29-day cycle")
		assert.Equal(t, OvulationCalendar, st.Anchors.OvulationSource)
		assert.True(t, st.DaysToOvulation.OK())
		assert.Equal(t, MainPhaseFertile, s.Resolve(second.Start+15, today).MainPhase)
		assert.Zero(t, s.Suppression(today))
		// The days between the two periods stay as they were (no prediction then).
		assert.True(t, s.Resolve(day("2026-10-25"), today).Limits.FertilitySuppressed)
	})

	t.Run("breastfeeding without a delivery date counts from the first period on record", func(t *testing.T) {
		in := none()
		in.Breastfeeding = true // mode cycle
		in.Periods = []Period{first}
		st := prep(in).Resolve(day("2026-10-25"), day("2026-10-25"))
		assert.Equal(t, SuppressionPostpartumNotReturned, st.Limits.SuppressionReason)
		assert.Equal(t, ConfidenceLow, st.Confidence)
		in.Periods = []Period{first, second}
		assert.Zero(t, prep(in).Resolve(day("2026-11-20"), day("2026-11-20")).Limits)
	})

	t.Run("postpartum beats hormonal; both warnings", func(t *testing.T) {
		in := base()
		in.Hormonal = true
		in.Periods = []Period{first}
		st := prep(in).Resolve(day("2026-10-25"), day("2026-10-25"))
		assert.Equal(t, SuppressionPostpartumNotReturned, st.Limits.SuppressionReason)
		assert.True(t, st.Warnings.Has(WarnPostpartumCyclesNotReturned))
		assert.True(t, st.Warnings.Has(WarnHormonalContraceptionActive))
		assert.Equal(t, NoDay, st.Anchors.PredictedNextPeriodStart)
		// Once cycles return, hormonal contraception takes over.
		in.Periods = []Period{first, second}
		st = prep(in).Resolve(day("2026-11-20"), day("2026-11-20"))
		assert.Equal(t, SuppressionHormonalContraception, st.Limits.SuppressionReason)
	})
}

// personaFixture is the Input and today of one persona fixture of testdata/personas by name.
func personaFixture(t *testing.T, name string) (Input, Day) {
	t.Helper()
	for _, p := range loadPersonas(t) {
		if p.Name == name {
			return p.input(t), day(p.Today)
		}
	}
	t.Fatalf("no persona %s", name)
	return Input{}, NoDay
}

// personaState prepares one persona fixture of testdata/personas by name.
func personaState(t *testing.T, name string) (*State, Day) {
	t.Helper()
	in, today := personaFixture(t, name)
	return Prepare(in, DefaultParams()), today
}

// T-M9-48b: E-14 applies only on and after the delivery date; the history before it resolves normally.
func TestLimits_PostpartumOnlyFromDelivery(t *testing.T) {
	s, today := personaState(t, "postpartum_breastfeeding")
	delivery := day("2026-06-23")

	t.Run("closed pre-pregnancy cycle keeps its phases and cycle days", func(t *testing.T) {
		for _, c := range []struct {
			date string
			cd   int
		}{{"2025-06-10", 10}, {"2025-06-20", 20}} {
			st := s.Resolve(day(c.date), today)
			assert.Equal(t, Some(c.cd), st.CycleDay, c.date)
			assert.NotEqual(t, MainPhaseUnknown, st.MainPhase, c.date)
			assert.Equal(t, ResolutionPrediction, st.ResolutionSource, c.date)
			assert.False(t, st.Warnings.Has(WarnPostpartumCyclesNotReturned), c.date)
			assert.Zero(t, st.Limits, c.date)
			assert.Equal(t, day("2025-06-30"), st.Anchors.PredictedNextPeriodStart, c.date)
			assert.True(t, st.Anchors.EstimatedOvulationDate.IsSet(), c.date)
		}
		assert.Equal(t, MainPhaseFertile, s.Resolve(day("2025-06-14"), today).MainPhase) // O = 06-16 (E-22)
	})

	t.Run("pregnancy gap: unknown, nothing predicted, no postpartum warning or suppression", func(t *testing.T) {
		for _, d := range []Day{day("2025-08-15"), day("2026-04-22"), delivery - 1} {
			st := s.Resolve(d, today)
			assert.Equal(t, MainPhaseUnknown, st.MainPhase, d)
			assert.False(t, st.CycleDay.OK(), d)
			assert.Equal(t, NoDay, st.Anchors.PredictedNextPeriodStart, d)
			assert.Equal(t, NoDay, st.Anchors.EstimatedOvulationDate, d)
			assert.False(t, st.DaysToOvulation.OK(), d)
			assert.False(t, st.Warnings.Has(WarnPostpartumCyclesNotReturned), d)
			assert.False(t, st.Warnings.Has(WarnInsufficientAnchorData), d)
			assert.Zero(t, st.Limits, d)
			assert.False(t, st.RequiresUserInput, d)
		}
		// The logged bleeding that opens the gap cycle stays menstrual.
		st := s.Resolve(day("2025-07-29"), today)
		assert.Equal(t, MainPhaseMenstrual, st.MainPhase)
		assert.Equal(t, Some(2), st.CycleDay)
	})

	t.Run("on and after the delivery date E-14 is unchanged", func(t *testing.T) {
		for _, d := range []Day{delivery, delivery + 30, today} {
			st := s.Resolve(d, today)
			assert.True(t, st.Warnings.Has(WarnPostpartumCyclesNotReturned), d)
			assert.Equal(t, SuppressionPostpartumNotReturned, st.Limits.SuppressionReason, d)
		}
	})
}

// E-26: a positive pregnancy test.
func TestLimits_PositivePregnancyTest(t *testing.T) {
	start := day("2026-07-01")
	test := day("2026-07-20")
	next := day("2026-08-02")
	in := regular(start, 28, 5)
	in.PregnancyTests = []PregnancyTest{{Date: test, Positive: true}, {Date: start - 10, Positive: true},
		{Date: test - 3}}
	s := prep(in)

	assert.True(t, s.Resolve(test, test).Limits.PromptPregnancyMode)
	assert.True(t, s.Resolve(test+5, test+5).Limits.PromptPregnancyMode)
	assert.True(t, s.Suppression(test+1).PromptPregnancyMode)
	assert.False(t, s.Resolve(test-1, test-1).Limits.PromptPregnancyMode, "negative test, or before the test")
	assert.False(t, s.Suppression(test).FertilitySuppressed, "the prompt is not a suppression")

	// It does not change phases by itself.
	plain := prep(regular(start, 28, 5))
	for d := start; d < start+40; d++ {
		got, want := s.Resolve(d, d), plain.Resolve(d, d)
		got.Limits.PromptPregnancyMode = false
		assert.Equal(t, want, got, "%s", d)
	}

	ttc := in
	ttc.Mode = ModeTTC
	assert.True(t, prep(ttc).Resolve(test+1, test+1).Limits.PromptPregnancyMode)

	// The next real start clears it.
	after := in
	after.Periods = append(append([]Period(nil), in.Periods...), Period{Start: next, End: next + 4})
	sa := prep(after)
	assert.False(t, sa.Resolve(next, next).Limits.PromptPregnancyMode)
	assert.False(t, sa.Suppression(next+3).PromptPregnancyMode)
	assert.True(t, sa.Resolve(test+1, next).Limits.PromptPregnancyMode, "history keeps the prompt of its cycle")

	// A mode change clears it.
	for _, m := range []Mode{ModePregnancy, ModePostpartum, ModeMenopause, ModeTeen} {
		other := in
		other.Mode = m
		assert.False(t, prep(other).Resolve(test+1, test+1).Limits.PromptPregnancyMode, m.String())
	}

	// No anchor: no current_period_start, no prompt.
	noAnchor := Input{PregnancyTests: []PregnancyTest{{Date: test, Positive: true}}}
	assert.False(t, prep(noAnchor).Suppression(test).PromptPregnancyMode)
}

// E-29: menopause and teen modes are computed normally.
func TestLimits_MenopauseAndTeen(t *testing.T) {
	cur := day(julyStart)
	normal := prep(regular(cur, 28, 5))
	for _, m := range []Mode{ModeMenopause, ModeTeen} {
		in := regular(cur, 28, 5)
		in.Mode = m
		s := prep(in)
		for d := cur - 30; d < cur+28; d++ { // not late: suggest_pregnancy_test is mode-dependent
			st := s.Resolve(d, d)
			assert.Equal(t, normal.Resolve(d, d), st, "%s %s", m, d)
		}
		assert.Zero(t, s.Suppression(cur+10), m.String())
		assert.Equal(t, MainPhaseFertile, s.Resolve(cur+13, cur+13).MainPhase, m.String())
	}
}

// Invariant over every fixture: fertility suppressed ⇒ no day is fertile, fertility_level is unknown, no ovulation
// and no countdown; hormonal contraception ⇒ suppressed (§2.40 rule 9); suppression never adds
// insufficient_anchor_data to an anchored status.
func TestLimits_Invariant(t *testing.T) {
	cur := day(julyStart)
	with := func(in Input, f func(*Input)) Input { f(&in); return in }
	markers := func(in *Input) { in.BBT, in.LH = shiftAt(cur+12), []LHTest{lhOn(cur+11, true)} }
	fixtures := map[string]Input{
		"regular":           regular(cur, 28, 5),
		"regular markers":   with(regular(cur, 28, 5), markers),
		"pill":              pill(),
		"pill markers":      with(pill(), markers),
		"pill open":         with(pill(), func(in *Input) { in.Periods = append(in.Periods, Period{Start: cur + 28, End: NoDay}) }),
		"pill short cycles": with(withPeriods(periodsEnding(cur, 5, 21, 21)), func(in *Input) { in.Hormonal = true }),
		"pill declared": with(declared(Declared{LastPeriodStart: Some(cur)}),
			func(in *Input) { in.Hormonal = true }),
		"pregnancy":       with(regular(cur, 28, 5), func(in *Input) { in.Mode = ModePregnancy }),
		"pregnancy empty": {Mode: ModePregnancy},
		"postpartum": with(regular(cur, 28, 5), func(in *Input) {
			in.Mode, in.DeliveryDate = ModePostpartum, Some(cur-20)
			markers(in)
		}),
		"postpartum returned": with(regular(cur, 28, 5), func(in *Input) {
			in.Mode, in.DeliveryDate = ModePostpartum, Some(cur-40)
		}),
		"breastfeeding": with(regular(cur, 28, 5), func(in *Input) { in.Breastfeeding = true }),
		"breastfeeding one period": with(withPeriods([]Period{{Start: cur, End: cur + 4}}),
			func(in *Input) { in.Breastfeeding = true }),
		"ttc positive test": with(regular(cur, 28, 5), func(in *Input) {
			in.Mode = ModeTTC
			in.PregnancyTests = []PregnancyTest{{Date: cur + 20, Positive: true}}
		}),
		"menopause": with(regular(cur, 28, 5), func(in *Input) { in.Mode = ModeMenopause }),
		"teen":      with(regular(cur, 28, 5), func(in *Input) { in.Mode = ModeTeen }),
	}
	for name, in := range fixtures {
		s := prep(in)
		for _, today := range []Day{cur + 10, cur + 40} {
			for d := cur - 80; d <= cur+60; d++ {
				st := s.Resolve(d, today)
				msg := fmt.Sprintf("%s %s (today %s)", name, d, today)
				if in.Hormonal {
					require.True(t, st.Limits.FertilitySuppressed, msg)
				}
				if st.Limits.FertilitySuppressed {
					assertSuppressedFertility(t, st, msg)
					assert.False(t, st.Anchors.CurrentPeriodStart != NoDay &&
						st.Warnings.Has(WarnInsufficientAnchorData), msg)
				}
				assert.Equal(t, st.Limits.SuppressionReason != SuppressionNone, st.Limits.FertilitySuppressed, msg)
				assert.Equal(t, st.Limits.SuppressionReason == SuppressionPregnancy, st.Limits.EngineSuspended, msg)
				if st.Limits.EngineSuspended || st.Limits.SuppressionReason == SuppressionPostpartumNotReturned {
					assert.Contains(t, []MainPhase{MainPhaseUnknown, MainPhaseMenstrual}, st.MainPhase, msg)
				}
			}
		}
	}
}

func TestLimits_NoAllocs(t *testing.T) {
	cur := day(julyStart)
	pp := regular(cur, 28, 5)
	pp.Mode, pp.Breastfeeding, pp.DeliveryDate = ModePostpartum, true, Some(cur-20)
	preg := regular(cur, 28, 5)
	preg.Mode = ModePregnancy
	ttc := regular(cur, 28, 5)
	ttc.Mode, ttc.PregnancyTests = ModeTTC, []PregnancyTest{{Date: cur + 20, Positive: true}}
	states := []*State{prep(pill()), prep(pp), prep(preg), prep(ttc)}
	targets := []Day{cur + 3, cur + 12, cur + 21, cur + 31, cur + 60, cur - 30}
	var st Status
	var l Limits
	allocs := testing.AllocsPerRun(200, func() {
		for _, s := range states {
			for _, t := range targets {
				st = s.Resolve(t, cur+25)
			}
			l = s.Suppression(cur + 25)
		}
	})
	assert.Zero(t, allocs)
	_, _ = st, l
}

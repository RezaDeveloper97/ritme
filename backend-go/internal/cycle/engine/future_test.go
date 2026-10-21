package engine

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFuture_Chain is §2.16 / E-05: start 07-01, L = 29.
func TestFuture_Chain(t *testing.T) {
	cur := day("2026-07-01")
	s := prep(regular(cur, 29, 5))
	today := day("2026-07-10")
	fc := s.FutureCycles(today)
	require.Len(t, fc, 6)
	want := []string{"2026-07-30", "2026-08-28", "2026-09-26", "2026-10-25", "2026-11-23", "2026-12-22"}
	todayConf := s.Resolve(today, today).Confidence
	for i, c := range fc {
		assert.Equal(t, i+1, c.K)
		assert.Equal(t, want[i], c.PeriodStart.String(), "k=%d", c.K)
		assert.Equal(t, c.PeriodStart+4, c.PeriodEnd, "period_end = start + EPL − 1")
		assert.Equal(t, c.Ovulation-5, c.FertileStart, "fertile = O−5..O")
		assert.Equal(t, c.Ovulation, c.FertileEnd)
		if c.K == 1 {
			assert.Equal(t, todayConf, c.Confidence)
			assert.Equal(t, s.Resolve(today, today).Anchors.EstimatedOvulationDate, c.Ovulation)
			assert.Equal(t, "2026-07-16", c.Ovulation.String())
		} else {
			assert.LessOrEqual(t, c.Confidence, ConfidenceMedium)
			assert.Equal(t, c.PeriodStart-14, c.Ovulation)
			assert.Equal(t, OvulationCalendar, c.OvulationSource)
		}
	}
	assert.Equal(t, "2026-08-14", fc[1].Ovulation.String())

	// Targets: before the next start → the current cycle; from it on → future_cycles[k].
	st := s.Resolve(day("2026-07-29"), today)
	assert.Zero(t, st.FutureK)
	assert.Equal(t, StartUserLogged, st.Anchors.CurrentPeriodStartSource)
	assert.False(t, st.Warnings.Has(WarnPredictionBasedOutput))

	st = s.Resolve(day("2026-07-30"), today)
	assert.Equal(t, 1, st.FutureK)
	assert.Equal(t, SubphaseMenstrual, st.Subphase)
	assert.Equal(t, ResolutionPrediction, st.ResolutionSource)
	assert.Equal(t, StartPredictedReference, st.Anchors.CurrentPeriodStartSource)
	assert.Equal(t, Some(1), st.CycleDay)
	assert.Equal(t, Some(0), st.DaysLate)
	assert.True(t, st.Warnings.Has(WarnPredictionBasedOutput))
	assert.True(t, st.ConfidenceReasons.Has(ReasonPredictionBasedOutput))
	assert.Equal(t, todayConf, st.Confidence, "k = 1: today's confidence")

	st = s.Resolve(fc[1].Ovulation, today) // the ovulation before period 2 lies in cycle 1
	assert.Equal(t, 1, st.FutureK)
	assert.Equal(t, SubphaseOvulationLikely, st.Subphase)
	assert.Equal(t, fc[1].PeriodStart, st.Anchors.PredictedNextPeriodStart)

	st = s.Resolve(fc[1].PeriodStart+10, today)
	assert.Equal(t, 2, st.FutureK)
	assert.LessOrEqual(t, st.Confidence, ConfidenceMedium)
	assert.Equal(t, fc[1].PeriodStart, st.Anchors.CurrentPeriodStart)

	// A logged regular history with high confidence today: k = 1 high, k ≥ 2 medium.
	assert.Equal(t, ConfidenceHigh, todayConf)
	assert.Equal(t, ConfidenceMedium, fc[1].Confidence)
}

// TestFuture_LateRebase is E-04.
func TestFuture_LateRebase(t *testing.T) {
	s := prep(regular(day("2026-01-01"), 28, 5))
	today := day("2026-02-01") // next 01-29, late 3
	require.Equal(t, Some(3), s.Resolve(today, today).DaysLate)
	fc := s.FutureCycles(today)
	require.Len(t, fc, 6)
	assert.Equal(t, "2026-02-02", fc[0].PeriodStart.String())
	assert.Equal(t, "2026-03-02", fc[1].PeriodStart.String())

	st := s.Resolve(day("2026-02-02"), today)
	assert.Equal(t, MainPhaseMenstrual, st.MainPhase, "never «day 5»")
	assert.Equal(t, 1, st.FutureK)
	assert.Equal(t, Some(1), st.CycleDay)
	assert.True(t, st.IsPredicted())
	for d := day("2026-01-29"); d <= today; d++ {
		assert.Equal(t, SubphasePeriodExpected, s.Resolve(d, today).Subphase, "%s", d)
	}

	today = day("2026-02-13") // late 15
	assert.Empty(t, s.FutureCycles(today))
	for _, d := range []Day{today + 1, today + 2, today + 40} {
		st = s.Resolve(d, today)
		assert.Equal(t, MainPhaseUnknown, st.MainPhase, "%s", d)
		assert.Equal(t, ResolutionUnknown, st.ResolutionSource)
		assert.False(t, st.Warnings.Has(WarnInsufficientAnchorData))
		assert.Zero(t, st.FutureK)
		assert.False(t, st.DaysToOvulation.OK())
	}
}

// TestFuture_BeyondK6 is E-28.
func TestFuture_BeyondK6(t *testing.T) {
	cur := day("2026-07-01")
	s := prep(regular(cur, 28, 5))
	fc := s.FutureCycles(cur)
	last := fc[len(fc)-1]
	assert.Equal(t, MainPhaseMenstrual, s.Resolve(last.PeriodEnd, cur).MainPhase)
	assert.Equal(t, 6, s.Resolve(last.PeriodEnd, cur).FutureK)
	for _, d := range []Day{last.PeriodEnd + 1, last.Ovulation + 28, last.PeriodEnd + 400} {
		st := s.Resolve(d, cur)
		assert.Equal(t, MainPhaseUnknown, st.MainPhase)
		assert.Equal(t, ResolutionUnknown, st.ResolutionSource)
		assert.Equal(t, FertilityUnknown, st.FertilityLevel)
		assert.False(t, st.DaysToOvulation.OK())
		assert.Equal(t, NoDay, st.Anchors.EstimatedOvulationDate)
		assert.False(t, st.RequiresUserInput)
	}
}

// TestFuture_StaleAnchor is §2.41.
func TestFuture_StaleAnchor(t *testing.T) {
	lmp := day("2026-07-01")
	s := prep(declared(Declared{LastPeriodStart: Some(lmp)})) // default lengths 28 / 5

	st := s.Resolve(lmp+42, lmp+42) // diff 42 = 28 + 14 → not stale
	assert.Equal(t, SubphasePeriodExpected, st.Subphase)
	assert.Equal(t, Some(14), st.DaysLate)
	assert.Equal(t, StartOnboardingDeclared, st.Anchors.CurrentPeriodStartSource)
	assert.False(t, st.Warnings.Has(WarnStaleOnboardingAnchor))

	st = s.Resolve(lmp+43, lmp+43) // 43 → stale, n = 1
	assert.Equal(t, lmp+28, st.Anchors.CurrentPeriodStart)
	assert.Equal(t, Some(16), st.CycleDay)
	assert.True(t, st.Warnings.Has(WarnStaleOnboardingAnchor))

	target := day("2026-09-29") // diff 90 → n = 3
	st = s.Resolve(target, target)
	assert.Equal(t, "2026-09-23", st.Anchors.CurrentPeriodStart.String())
	assert.Equal(t, StartPredictedReference, st.Anchors.CurrentPeriodStartSource)
	assert.Equal(t, Some(7), st.CycleDay)
	assert.Equal(t, SubphaseMidFollicular, st.Subphase)
	assert.Equal(t, ConfidenceLow, st.Confidence)
	assert.True(t, st.Warnings.Has(WarnStaleOnboardingAnchor))
	assert.True(t, st.Warnings.Has(WarnPredictionBasedOutput))
	assert.True(t, st.ConfidenceReasons.Has(ReasonStaleOnboardingAnchor))
	assert.True(t, st.RequiresUserInput)
	assert.Equal(t, ResolutionDefaultBased, st.ResolutionSource, "E-10: derived from the onboarding date")
	assert.Equal(t, Some(0), st.DaysLate, "a stale anchor is never late")

	// The projection from a stale today keeps the warning and stays low.
	fc := s.FutureCycles(target)
	require.Len(t, fc, 6)
	assert.Equal(t, "2026-10-21", fc[0].PeriodStart.String())
	assert.Equal(t, ConfidenceLow, fc[0].Confidence)
	fst := s.Resolve(fc[0].PeriodStart+2, target)
	assert.Equal(t, 1, fst.FutureK)
	assert.True(t, fst.Warnings.Has(WarnStaleOnboardingAnchor))
	assert.Equal(t, ConfidenceLow, fst.Confidence)

	// Any real period → no stale anchor.
	in := declared(Declared{LastPeriodStart: Some(lmp)})
	in.Periods = []Period{{Start: day("2026-03-01"), End: day("2026-03-05")}}
	st = prep(in).Resolve(target, target)
	assert.False(t, st.Warnings.Has(WarnStaleOnboardingAnchor))
	assert.Equal(t, StartUserLogged, st.Anchors.CurrentPeriodStartSource)
}

// TestFuture_Skip is E-15 / §2.39.8.
func TestFuture_Skip(t *testing.T) {
	cur := day("2026-07-01")
	base := regular(cur, 28, 5)
	skipped := func(skips ...string) Input {
		in := base
		for _, d := range skips {
			in.Skips = append(in.Skips, day(d))
		}
		return in
	}
	target := day("2026-08-10")

	// Without the skip 08-10 is 12 days late.
	plain := prep(base).Resolve(target, target)
	require.Equal(t, Some(12), plain.DaysLate)
	require.True(t, plain.Warnings.Has(WarnPredictedPeriodOverdue))

	s := prep(skipped("2026-07-29"))
	assert.Empty(t, s.Diagnostics().IgnoredSkips)
	st := s.Resolve(target, target)
	assert.Equal(t, Some(13), st.CycleDay)
	assert.Equal(t, day("2026-07-29"), st.Anchors.CurrentPeriodStart)
	assert.Equal(t, StartPredictedReference, st.Anchors.CurrentPeriodStartSource)
	assert.Equal(t, "2026-08-26", st.Anchors.PredictedNextPeriodStart.String())
	assert.Equal(t, Some(0), st.DaysLate)
	assert.Equal(t, ConfidenceLow, st.Confidence)
	assert.True(t, st.Warnings.Has(WarnSkippedCycleRecorded))
	assert.True(t, st.Warnings.Has(WarnPredictionBasedOutput))
	assert.False(t, st.Warnings.Has(WarnPredictedPeriodOverdue))
	assert.Equal(t, NoDay, st.Anchors.CurrentPeriodEnd, "a skipped cycle has no period")
	assert.Equal(t, EndUnknown, st.Anchors.CurrentPeriodEndSource)
	assert.Equal(t, SubphaseEarlyFollicular, s.Resolve(day("2026-07-30"), day("2026-07-30")).Subphase,
		"no menstrual days after a skip")
	assert.Equal(t, "2026-08-26", s.FutureCycles(target)[0].PeriodStart.String())

	// Two consecutive skips chain.
	s2 := prep(skipped("2026-07-29", "2026-08-26"))
	st = s2.Resolve(day("2026-09-05"), day("2026-09-05"))
	assert.Equal(t, day("2026-08-26"), st.Anchors.CurrentPeriodStart)
	assert.Equal(t, Some(11), st.CycleDay)
	assert.Equal(t, "2026-09-23", st.Anchors.PredictedNextPeriodStart.String())
	assert.True(t, st.Warnings.Has(WarnSkippedCycleRecorded))

	// A real start 08-24 clears the warning; the 07-01 → 08-24 gap is excluded (skipped, not an outlier).
	in := skipped("2026-07-29")
	in.Periods = append(in.Periods, Period{Start: day("2026-08-24"), End: NoDay})
	s3 := prep(in)
	st = s3.Resolve(day("2026-08-25"), day("2026-08-25"))
	assert.False(t, st.Warnings.Has(WarnSkippedCycleRecorded))
	assert.False(t, st.Warnings.Has(WarnLongCycleOutlierDetected))
	gap := s3.Cycles()[len(s3.Cycles())-1]
	assert.True(t, gap.Skipped)
	assert.False(t, gap.Valid)
	assert.False(t, gap.Long)
	n, src := s3.CycleLength()
	assert.Equal(t, 28, n)
	assert.Equal(t, LengthHistory, src)
	// History inside the skipped cycle keeps the skip anchor and the actual next start (E-22).
	st = s3.Resolve(day("2026-08-10"), day("2026-08-25"))
	assert.Equal(t, day("2026-07-29"), st.Anchors.CurrentPeriodStart)
	assert.Equal(t, day("2026-08-24"), st.Anchors.PredictedNextPeriodStart)
	assert.True(t, st.Warnings.Has(WarnSkippedCycleRecorded))

	// Deleting the skip restores the late output.
	assert.Equal(t, plain, prep(base).Resolve(target, target))

	// A skip that matches no predicted start is ignored and reported.
	s4 := prep(skipped("2026-07-30"))
	assert.Equal(t, []Day{day("2026-07-30")}, s4.Diagnostics().IgnoredSkips)
	st = s4.Resolve(target, target)
	assert.Equal(t, plain, st)
	assert.False(t, st.Warnings.Has(WarnSkippedCycleRecorded))
	// So is a skip before any anchor, or on a real start.
	assert.Len(t, prep(skipped("2025-01-01", "2026-07-01")).Diagnostics().IgnoredSkips, 2)

	// A skip after a declared onboarding start: lmp + ECL.
	decl := declared(Declared{LastPeriodStart: Some(cur)})
	decl.Skips = []Day{cur + 28}
	st = prep(decl).Resolve(cur+35, cur+35)
	assert.Equal(t, cur+28, st.Anchors.CurrentPeriodStart)
	assert.True(t, st.Warnings.Has(WarnSkippedCycleRecorded))
	assert.Equal(t, ResolutionDefaultBased, st.ResolutionSource)
}

// TestFuture_Suppression is T-M9-19 → E-13 / E-14 / E-25.
func TestFuture_Suppression(t *testing.T) {
	cur := day(julyStart)
	s := prep(pill())
	fc := s.FutureCycles(cur + 3)
	require.Len(t, fc, 6)
	for _, c := range fc {
		assert.Equal(t, NoDay, c.Ovulation)
		assert.Equal(t, NoDay, c.FertileStart)
		assert.Equal(t, NoDay, c.FertileEnd)
		assert.Equal(t, OvulationUnknown, c.OvulationSource)
		assert.NotEqual(t, NoDay, c.PeriodStart)
	}
	for d := fc[0].PeriodStart; d <= fc[5].PeriodEnd; d++ {
		assertSuppressedFertility(t, s.Resolve(d, cur+3), d.String())
	}

	pp := regular(cur, 28, 5)
	pp.Mode, pp.DeliveryDate = ModePostpartum, Some(cur-200)
	pp.Periods = pp.Periods[len(pp.Periods)-1:] // one period after delivery: not returned
	ps := prep(pp)
	require.Equal(t, SuppressionPostpartumNotReturned, ps.Suppression(cur+3).SuppressionReason)
	assert.Equal(t, []FutureCycle{}, ps.FutureCycles(cur+3))
	assert.Equal(t, MainPhaseUnknown, ps.Resolve(cur+40, cur+3).MainPhase)

	preg := regular(cur, 28, 5)
	preg.Mode = ModePregnancy
	assert.Empty(t, prep(preg).FutureCycles(cur+3))
	assert.Empty(t, prep(none()).FutureCycles(cur))
}

// TestFuture_RealStartAfterToday: a logged start after today resolves as logged, not from the projection.
func TestFuture_RealStartAfterToday(t *testing.T) {
	cur := day("2026-07-01")
	in := regular(cur, 28, 5)
	in.Periods = append(in.Periods, Period{Start: cur + 30, End: cur + 34})
	s := prep(in)
	st := s.Resolve(cur+31, cur+10)
	assert.Zero(t, st.FutureK)
	assert.Equal(t, ResolutionUserLogged, st.ResolutionSource)
}

// TestFuture_EditOldPeriod is §6.1 "edit 3 months back": a pure recompute changes today's medians and regularity.
func TestFuture_EditOldPeriod(t *testing.T) {
	cur := day("2026-07-01")
	in := withPeriods(periodsEnding(cur, 5, 26, 30, 28))
	today := cur + 10
	before := prep(in).Resolve(today, today)
	require.Equal(t, RegularityRegular, before.Metrics.CycleRegularity)
	require.Equal(t, 28, before.Metrics.EffectiveCycleLength)

	edited := withPeriods(periodsEnding(cur, 5, 26, 30, 28))
	edited.Periods[0].Start -= 10 // the start 3 months back moves 10 days earlier: cycles 36, 30, 28
	edited.Periods[0].End -= 10
	after := prep(edited).Resolve(today, today)
	assert.Equal(t, 30, after.Metrics.EffectiveCycleLength)
	assert.Equal(t, RegularityIrregular, after.Metrics.CycleRegularity) // 36 − 28 = 8 > 7
	assert.Equal(t, Some(4), before.Metrics.CycleVariability)
	assert.Equal(t, Some(8), after.Metrics.CycleVariability)
	assert.Equal(t, cur+30, after.Anchors.PredictedNextPeriodStart)
	assert.NotEqual(t, before.Metrics.CycleVariability, after.Metrics.CycleVariability)
}

func TestRange_Bounds(t *testing.T) {
	s := prep(regular(day("2026-07-01"), 28, 5))
	d := day("2026-07-01")
	assert.Nil(t, s.Range(d, d-1, d))
	assert.Nil(t, s.Range(d, d+MaxRangeDays, d))
	assert.Len(t, s.Range(d, d+MaxRangeDays-1, d), MaxRangeDays)
	assert.Len(t, s.Range(d, d, d), 1)
}

// randomInput is a seeded random history: real periods (some open, some conflicting), skips (some matching), a
// declared start, lengths, markers, modes and contraception.
func randomInput(r *rand.Rand, today Day) Input {
	in := none()
	start := today - Day(r.IntN(500))
	n := r.IntN(14)
	if r.IntN(6) == 0 {
		n = 0
	}
	d := start - Day(n*30)
	for i := 0; i < n; i++ {
		p := Period{Start: d, End: d + Day(2+r.IntN(9))}
		if r.IntN(8) == 0 {
			p.End = NoDay
		}
		in.Periods = append(in.Periods, p)
		if r.IntN(5) == 0 { // a skip at (or near) the predicted start
			in.Skips = append(in.Skips, d+Day(26+r.IntN(5)))
		}
		d += Day(18 + r.IntN(40))
	}
	if r.IntN(3) == 0 {
		in.Declared.LastPeriodStart = Some(today - Day(r.IntN(200)))
	}
	if r.IntN(3) == 0 {
		in.Declared.CycleLength = Some(20 + r.IntN(30))
		in.Declared.PeriodLength = Some(1 + r.IntN(10))
		in.Declared.Manual = r.IntN(4) == 0
	}
	if r.IntN(4) == 0 {
		in.Skips = append(in.Skips, today-Day(r.IntN(120)))
	}
	if r.IntN(4) == 0 && n > 0 {
		last := in.Periods[n-1].Start
		for i := 0; i < 20; i++ {
			in.BBT = append(in.BBT, BBTReading{Date: last + Day(i), Centi: int32(3640 + r.IntN(10) + 30*(i/14))})
		}
		in.LH = append(in.LH, LHTest{Date: last + Day(10+r.IntN(6)), Positive: r.IntN(2) == 0})
	}
	in.Mode = Mode(r.IntN(6))
	in.Hormonal = r.IntN(6) == 0
	if r.IntN(6) == 0 {
		in.Breastfeeding = true
		in.DeliveryDate = Some(today - Day(r.IntN(400)))
	}
	if r.IntN(5) == 0 {
		in.BirthDate = Some(today - Day(365*(12+r.IntN(40))))
	}
	if r.IntN(8) == 0 {
		in.PregnancyTests = []PregnancyTest{{Date: today - Day(r.IntN(30)), Positive: true}}
	}
	return in
}

// TestRange_EqualsResolve is the property test: Range == map(Resolve) over 2 000 seeded random histories × a
// 400-day window crossing today.
func TestRange_EqualsResolve(t *testing.T) {
	r := rand.New(rand.NewPCG(20, 2026))
	today := day("2026-10-21")
	future := 0
	for i := 0; i < 2000; i++ {
		in := randomInput(r, today)
		s := prep(in)
		from := today - Day(r.IntN(400))
		to := from + MaxRangeDays - 1
		got := s.Range(from, to, today)
		require.Len(t, got, MaxRangeDays)
		for j, st := range got {
			d := from + Day(j)
			want := s.Resolve(d, today)
			if !assert.Equal(t, want, st, "history %d, day %s", i, d) {
				t.FailNow()
			}
			if st.FutureK > 0 {
				future++
			}
		}
	}
	assert.Positive(t, future, "the window reaches future cycles")
}

func TestFuture_NoAllocs(t *testing.T) {
	cur := day("2026-07-01")
	in := withPeriods(periodsEnding(cur, 5, 28, 29, 27, 30, 28, 28))
	in.Skips = []Day{cur + 28}
	s := prep(in)
	var st Status
	targets := []Day{cur + 3, cur + 30, cur + 60, cur + 100, cur + 400}
	allocs := testing.AllocsPerRun(200, func() {
		for _, t := range targets {
			st = s.Resolve(t, cur+29)
		}
	})
	assert.Zero(t, allocs)
	_ = st
	assert.LessOrEqual(t, testing.AllocsPerRun(50, func() { _ = s.FutureCycles(cur + 29) }), 1.0)
}

func bench24() (*State, Day) { return prep(withPeriods(bench24Periods())), day("2026-07-01") }

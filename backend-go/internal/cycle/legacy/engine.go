// Package legacy ports HealthDataEngine::calculateForDate (backend/app/Services/HealthEngine/
// HealthDataEngine.php) — the `calculation` of /cycle/today|date, the month view, home and the
// message system. It disagrees with the v1.1 resolver by design; both are ported exactly.
//
// The engine is pure: histories, profile, logs and "today" are inputs, and the only I/O is the
// recommendation repository behind TipSource.
package legacy

import (
	"context"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

const (
	// lutealLength is the fixed luteal-phase length of the MVP ovulation model (§2).
	lutealLength = 14
	// minOvulationDay keeps ovulation from collapsing the follicular phase on very short cycles.
	minOvulationDay = 7
)

// TipSource is the admin-managed recommendations (recommendation.Repository).
type TipSource interface {
	HasContent(ctx context.Context) (bool, error)
	ForDay(ctx context.Context, phase enums.CyclePhase, subphase enums.CycleSubphase, log enums.TriggerLog) ([]recommendation.Tip, error)
}

// Input is everything calculateForDate reads.
type Input struct {
	// Profile is the user's profile (nil = no profile row).
	Profile *model.Profile
	// Histories is the user's full cycle_histories (any order; every row takes part in the anchor
	// search, not only confirmed ones).
	Histories []model.History
	// Logs holds the user's daily logs by log_date for (at least) the dates calculated.
	Logs map[civildate.Date]*DailyLog
	// Today is the Tehran calendar day; only the age factor reads it (Carbon `->age` is relative
	// to now).
	Today civildate.Date
	// Tips supplies the database recommendations (required; a recommendation.Repository).
	Tips TipSource
}

// Engine is one HealthDataEngine instance: the metrics and the sorted history are computed once
// and shared by every date it calculates.
type Engine struct {
	in      Input
	metrics metrics.Metrics
	// starts is the history newest start first (HealthDataEngine::periodStarts).
	starts []model.History
	mapper resolver.PhaseMapper
}

// New builds an engine over in.
func New(in Input) *Engine {
	starts := slices.Clone(in.Histories)
	slices.SortStableFunc(starts, func(a, b model.History) int { return b.PeriodStart.Compare(a.PeriodStart) })
	return &Engine{in: in, metrics: metrics.Calculate(in.Histories, in.Profile), starts: starts}
}

// Metrics returns the three-layer metrics the engine calculates with.
func (e *Engine) Metrics() metrics.Metrics { return e.metrics }

// CalculateForDate is HealthDataEngine::calculateForDate. With withContent false (calendar mode)
// only CalendarFields are emitted and no text flags, tips or snapshots are built.
func (e *Engine) CalculateForDate(ctx context.Context, date civildate.Date, withContent bool) (Calculation, error) {
	profile := e.in.Profile
	if profile == nil || profile.LastPeriodStart.IsZero() {
		return emptyCalculation(date, withContent), nil
	}

	log := e.in.Logs[date]
	cycleLength := e.metrics.EffectiveCycleLength
	variability := e.metrics.Variability

	anchor := e.relevantCycleStart(date, cycleLength)
	cycleDay := cycleDayFor(anchor, date, cycleLength)
	ovulationDay := OvulationDay(cycleLength)

	bleeding := e.bleedingLength(cycleLength, e.loggedBleedingLength(anchor))
	phase := e.mapper.PhaseFor(cycleDay, ovulationDay, bleeding)
	subphase := e.mapper.SubphaseFor(cycleDay, ovulationDay, cycleLength, bleeding)

	// Menstruation wins: a bleeding day is never also fertile or PMS.
	isPeriodDay := phase == enums.CyclePhaseMenstruation
	isFertile := InFertileWindow(cycleDay, ovulationDay) && !isPeriodDay
	isPms := InPmsWindow(cycleDay, cycleLength) && !isPeriodDay
	isPeriodTomorrow := PeriodTomorrow(cycleDay, cycleLength, variability)
	isLutealSpotting := LutealSpotting(cycleDay, log)

	cycleScore := CycleScore(variability, log)
	ageFactor := e.ageFactor()
	baseProbability := BaseProbability(cycleDay - ovulationDay)
	symptomScore := SymptomScore(log, isPms, isLutealSpotting)
	finalProbability := FinalProbability(baseProbability, ageFactor, cycleScore, symptomScore)

	c := Calculation{
		Date:             date,
		Complete:         true,
		Day:              cycleDay,
		Phase:            phase,
		CurrentSubphase:  subphase,
		OvulationDay:     ovulationDay,
		CycleLength:      cycleLength,
		IsFertileWindow:  isFertile,
		IsPmsWindow:      isPms,
		IsPeriodTomorrow: isPeriodTomorrow,
		IsLutealSpotting: isLutealSpotting,
		CycleScore:       phpround.Round(cycleScore, 4),
		AgeFactor:        phpround.Round(ageFactor, 4),
		BaseProbability:  phpround.Round(baseProbability, 4),
		SymptomScore:     phpround.Round(symptomScore, 4),
		FinalProbability: phpround.Round(finalProbability*100, 2),
		Variability:      variability,
		UncertaintyRange: variability.UncertaintyRange(),
		calendar:         !withContent,
	}
	if !withContent {
		return c, nil
	}

	c.TextFlags = TextFlags(phase, subphase, isFertile, isPms, isPeriodTomorrow, finalProbability, variability)
	tips, err := e.dailyTips(ctx, phase, subphase, log)
	if err != nil {
		return Calculation{}, err
	}
	c.DailyTips = tips
	c.Profile = &ProfileSnapshot{
		Birthday:        profile.Birthday,
		PeriodDuration:  profile.PeriodDuration,
		CycleDuration:   profile.CycleDuration,
		LastPeriodStart: profile.LastPeriodStart,
	}
	c.Log = log
	return c, nil
}

// emptyCalculation is getEmptyCalculation (HealthDataEngine.php:1101), or its calendar subset.
func emptyCalculation(date civildate.Date, withContent bool) Calculation {
	return Calculation{
		Date: date,
		TextFlags: []TextFlag{{
			Key: "incomplete_profile",
			EN:  "Please complete your profile to get cycle predictions.",
			FA:  "لطفاً پروفایل خود را کامل کنید تا پیش\u200cبینی سیکل را دریافت کنید.",
		}},
		DailyTips: []recommendation.Tip{},
		calendar:  !withContent,
	}
}

// relevantCycleStart is findRelevantCycleStart (HealthDataEngine.php:311): the latest history start
// on or before date (any row), rolled forward by whole cycles; else the profile LMP, rolled
// forward, or the LMP itself for earlier dates.
func (e *Engine) relevantCycleStart(date civildate.Date, cycleLength int) civildate.Date {
	for _, h := range e.starts {
		if h.PeriodStart.Compare(date) <= 0 {
			return rollForward(h.PeriodStart, date, cycleLength)
		}
	}
	lmp := e.in.Profile.LastPeriodStart
	if lmp.DiffDays(date) < 0 {
		return lmp
	}
	return rollForward(lmp, date, cycleLength)
}

// rollForward returns the start of the cycle containing date when cycles of cycleLength repeat
// from start (start <= date).
func rollForward(start, date civildate.Date, cycleLength int) civildate.Date {
	days := start.DiffDays(date)
	if days < cycleLength {
		return start
	}
	return start.AddDays(days / cycleLength * cycleLength)
}

// cycleDayFor is calculateCycleDay (HealthDataEngine.php:278). Before the anchor it extrapolates
// backwards with PHP's quirk: exactly k cycles before the anchor is day cycleLength, not day 1.
func cycleDayFor(anchor, date civildate.Date, cycleLength int) int {
	days := anchor.DiffDays(date)
	if days < 0 {
		return cycleLength - (-days)%cycleLength
	}
	return days%cycleLength + 1
}

// OvulationDay is calculateOvulationDay: max(L − 14 + 1, 7).
func OvulationDay(cycleLength int) int {
	return max(cycleLength-lutealLength+1, minOvulationDay)
}

// loggedBleedingLength is the bleeding_length of the history row whose start is the anchor (nil
// when the anchor is not a logged start or the column is NULL).
func (e *Engine) loggedBleedingLength(anchor civildate.Date) *int {
	for _, h := range e.starts {
		if h.PeriodStart == anchor {
			return h.BleedingLength
		}
	}
	return nil
}

// bleedingLength is effectiveBleedingLength (HealthDataEngine.php:413): the logged or effective
// length, falling back to min(5, L−1) when it would span the whole cycle.
func (e *Engine) bleedingLength(cycleLength int, logged *int) int {
	bleeding := e.metrics.EffectivePeriodDuration
	if logged != nil {
		bleeding = *logged
	}
	if bleeding >= cycleLength {
		return max(1, min(5, cycleLength-1))
	}
	return max(1, bleeding)
}

// InFertileWindow is O−5 ≤ day ≤ O+1.
func InFertileWindow(cycleDay, ovulationDay int) bool {
	return cycleDay >= ovulationDay-5 && cycleDay <= ovulationDay+1
}

// InPmsWindow is day ≥ L − 6.
func InPmsWindow(cycleDay, cycleLength int) bool { return cycleDay >= cycleLength-6 }

// PeriodTomorrow is L − u − 1 ≤ day ≤ L + u − 1 with u the variability's uncertainty range.
func PeriodTomorrow(cycleDay, cycleLength int, variability enums.CycleVariability) bool {
	u := variability.UncertaintyRange()
	return cycleDay >= cycleLength-u-1 && cycleDay <= cycleLength+u-1
}

// LutealSpotting is spotting logged on cycle day 15–28.
func LutealSpotting(cycleDay int, log *DailyLog) bool {
	if log == nil {
		return false
	}
	return cycleDay >= 15 && cycleDay <= 28 && isTrue(log.Spotting)
}

// ageFactor is calculateAgeFactor against the Tehran "today".
func (e *Engine) ageFactor() float64 {
	if e.in.Profile.Birthday.IsZero() {
		return noBirthdayAgeFactor
	}
	return AgeFactor(e.in.Profile.Birthday.AgeOn(e.in.Today))
}

// dailyTips is generateDailyTips (HealthDataEngine.php:762): the database recommendations once any
// row exists, else the built-in copy (phase tips, then symptom tips when there is a log).
func (e *Engine) dailyTips(ctx context.Context, phase enums.CyclePhase, subphase enums.CycleSubphase, log *DailyLog) ([]recommendation.Tip, error) {
	has, err := e.in.Tips.HasContent(ctx)
	if err != nil {
		return nil, fmt.Errorf("legacy: tips: %w", err)
	}
	if has {
		var trigger enums.TriggerLog
		if log != nil {
			trigger = triggerLog{log}
		}
		tips, err := e.in.Tips.ForDay(ctx, phase, subphase, trigger)
		if err != nil {
			return nil, fmt.Errorf("legacy: tips: %w", err)
		}
		return tips, nil
	}

	tips := phaseTips(phase, subphase)
	if log != nil {
		tips = append(tips, symptomTips(log)...)
	}
	return tips, nil
}

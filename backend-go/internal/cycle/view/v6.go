package view

import (
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The legacy cycle_view keys rebuilt from a v6 engine status (T-M9-32, DECISIONS.md X-05, D-120). The v6 payload is
// the §2.35 object from engine.AppendJSON; the frontend still parses the v1.1 render keys, so they are appended after
// it until T-M9-51 removes them. Everything here is a pure function of the status, its today_ui block and the state's
// effective lengths: no clock, no DB. A value v6 does not have is null (the frontend tolerance lands in T-M9-40).
//
//	phase                 I-11 content phase of the subphase (null for unknown)
//	data_status           the daily card's data status (null without a card)
//	predictions           next period start..start + EPL − 1, ovulation, display fertile window O−5..O (§2.19, null
//	                      under suppression), the one §2.29 confidence; null without a predicted start or while the
//	                      engine is suspended (the prediction card is null then)
//	profile_values        the declared lengths (onboarding / settings)
//	calculated_values     the history medians the engine uses (null when the effective value is not from history)
//	effective_values      the engine's effective lengths and their source
//	data_quality_details  confidence + reasons (v6), regularity, missing_period_end = warning or hard cap exceeded
//	daily_card            built from the status and today_ui (DailyCardV6); fertility `unknown` on period_expected

// Legacy is the block of legacy render keys appended to a v6 cycle_view, in the v1.1 key order.
type Legacy struct {
	Phase              *enums.CyclePhase        `json:"phase"`
	DataStatus         *enums.DataStatus        `json:"data_status"`
	Predictions        *PredictionsAPI          `json:"predictions"`
	ProfileValues      metrics.ProfileValues    `json:"profile_values"`
	CalculatedValues   metrics.CalculatedValues `json:"calculated_values"`
	EffectiveValues    metrics.EffectiveValues  `json:"effective_values"`
	DataQualityDetails DataQualityDetails       `json:"data_quality_details"`
	DailyCard          *DailyCard               `json:"daily_card"`
}

// V6Input is what LegacyFromV6 reads.
type V6Input struct {
	State  *engine.State
	Status *engine.Status
	UI     *engine.TodayUI // the status's today_ui block (State.TodayUI)
	Today  engine.Day      // the user-local day
	Locale string
}

// LegacyFromV6 rebuilds the legacy keys of one v6 status.
func LegacyFromV6(in V6Input) Legacy {
	st := in.Status
	var out Legacy
	if p, ok := ContentPhase(st.Subphase); ok {
		out.Phase = &p
	}

	cycleLen, cycleSrc := in.State.CycleLength()
	periodLen, periodSrc := in.State.PeriodLength()
	window := min(in.State.ValidCycleCount(), max(in.State.Params().HistoryWindow, 1))
	out.ProfileValues, out.CalculatedValues = lengthLayers(in.State.Declared(), cycleLen, cycleSrc, periodLen, periodSrc, window)
	out.EffectiveValues = metrics.EffectiveValues{
		CycleLength:          cycleLen,
		PeriodDuration:       periodLen,
		Source:               effectiveSource(cycleSrc),
		PeriodDurationSource: effectiveSource(periodSrc),
	}

	reasons := ConfidenceReasons(st)
	out.DataQualityDetails = DataQualityDetails{
		Confidence:          enums.ConfidenceLevel(st.Confidence.String()),
		ConfidenceReasons:   reasons,
		RegularityStatus:    regularityStatus(st.Metrics.CycleRegularity),
		IsIrregularPossible: st.Metrics.CycleRegularity == engine.RegularityIrregular,
		MissingPeriodEnd: st.Warnings.Has(engine.WarnPeriodEndMissingWarningCapExceeded) ||
			st.Warnings.Has(engine.WarnPeriodEndMissingHardCapExceeded),
	}

	if card, ok := in.UI.PredictionCard.Get(); ok {
		p := &PredictionsAPI{
			NextPeriodStart:   card.Period.Start.Civil(),
			NextPeriodEnd:     card.Period.End.Civil(),
			Source:            effectiveSource(cycleSrc),
			Confidence:        enums.ConfidenceLevel(st.Confidence.String()),
			ConfidenceReasons: reasons,
		}
		if ov, ok := card.Ovulation.Get(); ok {
			p.EstimatedOvulationDate = ov.Date.Civil()
		}
		if f, ok := card.Fertile.Get(); ok {
			p.FertileWindowStart, p.FertileWindowEnd = f.Start.Civil(), f.End.Civil()
		}
		out.Predictions = p
	}

	if card := DailyCardV6(CardV6Input{
		Status: st, UI: in.UI, Today: in.Today, Locale: in.Locale,
		NoFertilityCopy: NoFertilityCopy(in.State.Mode()),
	}); card != nil {
		out.DailyCard = card
		ds := card.DataStatus
		out.DataStatus = &ds
	}
	return out
}

// ContentPhase is the I-11 map of a v6 subphase onto the legacy content phase (daily tips, `phase`): menstrual →
// menstruation; follicular and fertile_rising → follicular; high_fertility and ovulation_likely → ovulation; luteal and
// period_expected → luteal; unknown → none (neutral content only).
func ContentPhase(s engine.Subphase) (enums.CyclePhase, bool) {
	switch {
	case s == engine.SubphaseUnknown:
		return "", false
	case s.Main() == engine.MainPhaseMenstrual:
		return enums.CyclePhaseMenstruation, true
	case s.Main() == engine.MainPhaseFollicular || s == engine.SubphaseFertileRising:
		return enums.CyclePhaseFollicular, true
	case s == engine.SubphaseHighFertility || s == engine.SubphaseOvulationLikely:
		return enums.CyclePhaseOvulation, true
	default: // luteal subphases, period_expected
		return enums.CyclePhaseLuteal, true
	}
}

// NoFertilityCopy reports whether a life mode drops the fertility copy of the day (teen, menopause — B-N2-11b,
// CB-TEEN-04b). The engine computes those modes normally (E-29); only the wording changes.
func NoFertilityCopy(m engine.Mode) bool { return m == engine.ModeTeen || m == engine.ModeMenopause }

// ConfidenceReasons are the status's confidence_reasons as strings, [] when none.
func ConfidenceReasons(st *engine.Status) []string {
	out := make([]string, 0, st.ConfidenceReasons.Len())
	for r := range st.ConfidenceReasons.All() {
		out = append(out, r.String())
	}
	return out
}

func effectiveSource(s engine.LengthSource) enums.EffectiveSource {
	switch s {
	case engine.LengthHistory:
		return enums.EffectiveSourceRecentValidCycles
	case engine.LengthDeclared:
		return enums.EffectiveSourceProfile
	default:
		return enums.EffectiveSourceDefault
	}
}

func regularityStatus(r engine.CycleRegularity) enums.RegularityStatus {
	switch r {
	case engine.RegularityRegular:
		return enums.RegularityStatusRelativelyRegular
	case engine.RegularityIrregular:
		return enums.RegularityStatusIrregularPossible
	default:
		return enums.RegularityStatusNotEnoughData
	}
}

// lengthLayers fills the profile layer with the declared lengths (onboarding / settings, raw) and the calculated
// layer with the history medians the engine uses (based on the valid cycles in the window). The state exposes only
// the effective median, so a history median that a manual declared value overrides (E-20) stays null.
func lengthLayers(d engine.Declared, cycleLen int, cycleSrc engine.LengthSource, periodLen int, periodSrc engine.LengthSource, window int) (metrics.ProfileValues, metrics.CalculatedValues) {
	var pv metrics.ProfileValues
	var cv metrics.CalculatedValues
	if v, ok := d.CycleLength.Get(); ok {
		pv.CycleLength = &v
	}
	if v, ok := d.PeriodLength.Get(); ok {
		pv.PeriodDuration = &v
	}
	if cycleSrc == engine.LengthHistory {
		n, w := cycleLen, window
		cv.CycleLength, cv.BasedOnCycles = &n, &w
	}
	if periodSrc == engine.LengthHistory {
		n := periodLen
		cv.PeriodDuration = &n
	}
	return pv, cv
}

// CardV6Input is what DailyCardV6 reads.
type CardV6Input struct {
	Status *engine.Status
	UI     *engine.TodayUI
	Today  engine.Day
	Locale string
	// NoFertilityCopy (teen, menopause) keeps the card off the fertile window / ovulation copy (B-N2-11b).
	NoFertilityCopy bool
}

// DailyCardV6 is the daily card of a v6 status, with the v1.1 copy (DailyCardBuilder) chosen from v6 facts; nil when
// the day has no cycle day or an unknown phase without lateness.
//
//   - a day of a logged period: closed → «روز N پریود» (actual); open → the open-period card, with the end prompt
//     (log end / still bleeding) when today_ui.ring_action is `end` (§2.43);
//   - period_expected on or before today: due (days_late 0) or overdue (days_late > 0), needs_confirmation; an
//     unresolved day after the expected period (unknown, days_late > 0) keeps the overdue card;
//   - a predicted menstrual day on or before today: confirm the start;
//   - anything else (an unknown day under hormonal contraception included): the predicted-day copy (countdowns,
//     fertile-window copy unless suppressed or NoFertilityCopy).
//
// fertility_level is the v6 one (unknown on period_expected, under suppression and with NoFertilityCopy).
func DailyCardV6(in CardV6Input) *DailyCard {
	st := in.Status
	cd, ok := st.CycleDay.Get()
	lateUnknown := st.MainPhase == engine.MainPhaseUnknown && st.DaysLate.Or(0) > 0
	// E-13: under hormonal contraception every non-bleeding day is unknown but still a predicted cycle day.
	suppressedDay := st.MainPhase == engine.MainPhaseUnknown && st.Limits.FertilitySuppressed && !st.Limits.EngineSuspended
	if !ok || (st.MainPhase == engine.MainPhaseUnknown && !lateUnknown && !suppressedDay) {
		return nil
	}
	b := cardBuilder{locale: in.Locale}
	sel := st.Date
	isFuture, isToday := sel > in.Today, sel == in.Today
	a := &st.Anchors
	logged := a.CurrentPeriodStartSource == engine.StartUserLogged &&
		(st.ResolutionSource == engine.ResolutionUserLogged || st.ResolutionSource == engine.ResolutionUserLoggedWithAssumedEnd)

	var card DailyCard
	switch {
	case st.MainPhase == engine.MainPhaseMenstrual && logged && a.CurrentPeriodEndIsConfirmed:
		card = b.actualPeriodDay(cd, cd)
	case st.MainPhase == engine.MainPhaseMenstrual && logged:
		card = b.openPeriodDay(cd, cd, OpenPeriodState{EndOverdue: in.UI.RingAction == engine.RingEnd}, isToday)
	case lateUnknown: // unresolved after the expected period (§2.22 beyond max_late): still the overdue card
		card = b.periodOverdue(st.DaysLate.Or(0))
	case st.MainPhase == engine.MainPhasePeriodExpected && !isFuture:
		if late := st.DaysLate.Or(0); late > 0 {
			card = b.periodOverdue(late)
		} else {
			card = b.periodDue(cd)
		}
	case st.MainPhase == engine.MainPhaseMenstrual && !isFuture:
		card = b.predictedPeriodDay(cd, isToday)
	default:
		fertility := !in.NoFertilityCopy && !st.Limits.FertilitySuppressed
		var ov civildate.Date
		if d := a.EstimatedOvulationDate; d.IsSet() && fertility {
			ov = d.Civil()
		}
		card = b.predictedDay(cd, enums.CycleSubphase(st.Subphase.String()), st.DaysToPeriod.Or(0), isFuture, isToday,
			sel.Civil(), ov, fertility)
	}

	level := enums.FertilityLevel(st.FertilityLevel.String())
	if in.NoFertilityCopy || st.MainPhase == engine.MainPhasePeriodExpected {
		level = enums.FertilityLevelUnknown
	}
	card.FertilityLevel = level
	card.FertilityLabel = level.Label(in.Locale)
	return &card
}

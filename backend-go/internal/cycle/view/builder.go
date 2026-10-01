// Package view is the port of the cycle_view assembly
// (backend/app/Services/HealthEngine/CycleDayViewBuilder.php) and the services it stitches together:
// CyclePredictionService, OpenPeriodEvaluator, CycleConfidenceCalculator and DailyCardBuilder.
//
// Build merges resolver.Status.ToAPI() (the §35 fields) with the legacy render keys (phase,
// data_status, predictions, three-layer values, data_quality_details, daily_card), in PHP key order.
// The legacy HealthDataEngine result it needs (cycle_day, subphase) is injected through BaseCalc,
// implemented by the legacy engine port (T-M2-14). Everything takes "today" and the locale as
// arguments — no clock, no DB.
//
// # Test mapping
//
// backend/tests/Unit/CyclePredictionServiceTest.php → TestCyclePredictionService:
//
//	test_predicts_next_period_ovulation_and_fertile_window
//	test_projects_the_anchor_forward_when_several_cycles_have_passed
//	test_uses_the_period_duration_for_the_predicted_end
//
// backend/tests/Unit/CycleConfidenceCalculatorTest.php → TestCycleConfidenceCalculator:
//
//	test_three_regular_cycles_give_high_confidence
//	test_three_irregular_cycles_are_only_medium
//	test_one_or_two_cycles_are_medium
//	test_profile_only_is_low
//	test_a_long_missing_end_downgrades_and_annotates
//
// backend/tests/Unit/OpenPeriodEvaluatorTest.php → TestOpenPeriodEvaluator:
//
//	test_no_open_period_returns_the_empty_state
//	test_within_expected_length_is_shown_as_active
//	test_past_expected_length_flags_incomplete_and_asks_for_the_end
//	test_past_hard_cap_stops_presenting_an_active_period
//	test_max_expected_follows_the_profile_period_duration
//	test_a_future_start_is_not_an_ongoing_bleed
//	test_hard_cap_dominates_a_long_profile_period_duration
//
// backend/tests/Unit/DailyCardBuilderTest.php → TestDailyCardBuilder:
//
//	test_inside_a_closed_logged_period_is_actual
//	test_open_period_within_expected_length_is_incomplete
//	test_open_period_past_expected_length_asks_for_the_end_today
//	test_predicted_period_due_needs_confirmation
//	test_one_day_overdue_reports_the_gap
//	test_several_days_overdue_switches_message
//	test_countdown_to_the_next_period
//	test_fertile_window_uses_the_subphase_fertility
//	test_ovulation_day
//	test_plain_day_counts_down_to_the_fertile_window
//	test_plain_day_past_the_window_counts_down_to_the_period
//	test_predicted_period_day_offers_confirming_the_start
//	test_predicted_period_day_in_the_future_stays_a_prediction
//	test_future_day_never_says_not_started_and_offers_a_reminder
//	test_persian_locale_localises_title_and_digits
//
// backend/tests/Unit/ExampleTest.php (test_that_true_is_true) is the Laravel placeholder and is not
// ported. TestCycleViewGoldenSweep additionally replays Laravel's /cycle/date sweep goldens.
package view

import (
	"sort"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// BaseCalc is the part of the legacy HealthDataEngine::calculateForDate result the view builder
// reads ($baseCalc['cycle_day'], $baseCalc['subphase']). The legacy engine port (T-M2-14)
// implements it on its calculation result.
type BaseCalc interface {
	// CycleDay is the legacy cycle day; ok is false for PHP null (incomplete profile).
	CycleDay() (day int, ok bool)
	// Subphase is the legacy sub-phase; only read when CycleDay is ok.
	Subphase() enums.CycleSubphase
}

// PredictionsAPI is the `predictions` block: CyclePrediction::toApiArray() + CycleConfidence::toApiArray().
type PredictionsAPI struct {
	NextPeriodStart        civildate.Date        `json:"next_period_start"`
	NextPeriodEnd          civildate.Date        `json:"next_period_end"`
	EstimatedOvulationDate civildate.Date        `json:"estimated_ovulation_date"`
	FertileWindowStart     civildate.Date        `json:"fertile_window_start"`
	FertileWindowEnd       civildate.Date        `json:"fertile_window_end"`
	Source                 enums.EffectiveSource `json:"source"`
	Confidence             enums.ConfidenceLevel `json:"confidence"`
	ConfidenceReasons      []string              `json:"confidence_reasons"`
}

// DataQualityDetails is the `data_quality_details` block.
type DataQualityDetails struct {
	Confidence          enums.ConfidenceLevel  `json:"confidence"`
	ConfidenceReasons   []string               `json:"confidence_reasons"`
	RegularityStatus    enums.RegularityStatus `json:"regularity_status"`
	IsIrregularPossible bool                   `json:"is_irregular_possible"`
	MissingPeriodEnd    bool                   `json:"missing_period_end"`
}

// CycleView is the `cycle_view` payload: the §35 status fields first, then the legacy render keys
// (array_merge order of CycleDayViewBuilder::build / emptyView).
type CycleView struct {
	resolver.StatusAPI
	Phase              *enums.CyclePhase        `json:"phase"`
	DataStatus         *enums.DataStatus        `json:"data_status"`
	Predictions        *PredictionsAPI          `json:"predictions"`
	ProfileValues      metrics.ProfileValues    `json:"profile_values"`
	CalculatedValues   metrics.CalculatedValues `json:"calculated_values"`
	EffectiveValues    metrics.EffectiveValues  `json:"effective_values"`
	DataQualityDetails DataQualityDetails       `json:"data_quality_details"`
	DailyCard          *DailyCard               `json:"daily_card"`
}

// Build is CycleDayViewBuilder::build. histories are the user's full cycle_histories (any order;
// the PHP caller passes newest first), profile nil = no profile row, today = the Tehran calendar day.
func Build(histories []model.History, profile *model.Profile, selected, today civildate.Date, locale string, base BaseCalc) CycleView {
	m := metrics.Calculate(histories, profile)
	status := resolver.Resolve(histories, profile, selected, today, m)
	layers := m.Layers()

	cycleDay, ok := base.CycleDay()
	if !ok {
		return emptyView(m, layers, status)
	}

	lastStart := lastConfirmedStart(histories, profile, today)
	prediction := Predict(lastStart, m.EffectiveCycleLength, m.EffectivePeriodDuration, lastStart, m.CycleLengthSource)

	var profilePeriodDuration *int
	if profile != nil {
		profilePeriodDuration = profile.PeriodDuration
	}
	openState := EvaluateOpenPeriod(openPeriodStart(histories, today), m.EffectivePeriodDuration, profilePeriodDuration, today)

	loggedDay, loggedClosed := LoggedPeriodFor(histories, selected, today)

	card := BuildDailyCard(CardInput{
		Selected:           selected,
		Today:              today,
		CycleDay:           cycleDay,
		Subphase:           base.Subphase(),
		PredictedNextStart: prediction.NextPeriodStart,
		OpenPeriod:         openState,
		LoggedPeriodDay:    loggedDay,
		LoggedPeriodClosed: loggedClosed,
		EstimatedOvulation: status.EstimatedOvulationDate,
		Locale:             locale,
		NoFertilityCopy:    profile != nil && profile.NoFertilityCopy,
	})

	confidence := ConfidenceForPrediction(m, openState.PastHardCap)

	var phase *enums.CyclePhase
	if p, ok := status.MainPhase.LegacyPhase(); ok {
		phase = &p
	}
	dataStatus := card.DataStatus

	return CycleView{
		StatusAPI:  status.ToAPI(),
		Phase:      phase,
		DataStatus: &dataStatus,
		Predictions: &PredictionsAPI{
			NextPeriodStart:        prediction.NextPeriodStart,
			NextPeriodEnd:          prediction.NextPeriodEnd,
			EstimatedOvulationDate: prediction.EstimatedOvulationDate,
			FertileWindowStart:     prediction.FertileWindowStart,
			FertileWindowEnd:       prediction.FertileWindowEnd,
			Source:                 prediction.Source,
			Confidence:             confidence.Level,
			ConfidenceReasons:      confidence.Reasons,
		},
		ProfileValues:    layers.ProfileValues,
		CalculatedValues: layers.CalculatedValues,
		EffectiveValues:  layers.EffectiveValues,
		DataQualityDetails: DataQualityDetails{
			Confidence:          confidence.Level,
			ConfidenceReasons:   confidence.Reasons,
			RegularityStatus:    m.RegularityStatus,
			IsIrregularPossible: m.RegularityStatus == enums.RegularityStatusIrregularPossible,
			MissingPeriodEnd:    openState.EndOverdue,
		},
		DailyCard: &card,
	}
}

func emptyView(m metrics.Metrics, layers metrics.Layers, status resolver.Status) CycleView {
	return CycleView{
		StatusAPI:        status.ToAPI(),
		ProfileValues:    layers.ProfileValues,
		CalculatedValues: layers.CalculatedValues,
		EffectiveValues:  layers.EffectiveValues,
		DataQualityDetails: DataQualityDetails{
			Confidence:          enums.ConfidenceLevelLow,
			ConfidenceReasons:   []string{ReasonProfileOnly},
			RegularityStatus:    m.RegularityStatus,
			IsIrregularPossible: false,
			MissingPeriodEnd:    false,
		},
	}
}

// newestFirst returns histories sorted by start descending, stable (PHP 8 sortByDesc keeps the
// input order of equal starts).
func newestFirst(histories []model.History) []model.History {
	out := append([]model.History(nil), histories...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].PeriodStart.After(out[j].PeriodStart) })
	return out
}

// lastConfirmedStart: the latest confirmed start ≤ ref, else the profile LMP, else ref.
func lastConfirmedStart(histories []model.History, profile *model.Profile, ref civildate.Date) civildate.Date {
	for _, h := range newestFirst(histories) {
		if h.IsConfirmed && !h.PeriodStart.After(ref) {
			return h.PeriodStart
		}
	}
	if profile != nil && !profile.LastPeriodStart.IsZero() {
		return profile.LastPeriodStart
	}
	return ref
}

// openPeriodStart: the latest record of all histories, only if it is confirmed, unclosed and
// started on/before ref; zero Date otherwise.
func openPeriodStart(histories []model.History, ref civildate.Date) civildate.Date {
	sorted := newestFirst(histories)
	if len(sorted) == 0 {
		return civildate.Date{}
	}
	latest := sorted[0]
	if latest.IsConfirmed && !latest.HasEnd() && !latest.PeriodStart.After(ref) {
		return latest.PeriodStart
	}
	return civildate.Date{}
}

// LoggedPeriodFor is CycleDayViewBuilder::loggedPeriodFor: if selected falls inside a confirmed
// logged period, its 1-based day number and whether that period is closed. An open period covers
// up to ref but never into the next confirmed period.
func LoggedPeriodFor(histories []model.History, selected, ref civildate.Date) (*int, bool) {
	confirmed := metrics.ConfirmedOldestFirst(histories)
	for i, h := range confirmed {
		if selected.Before(h.PeriodStart) {
			continue
		}
		day := h.PeriodStart.DiffDays(selected) + 1

		if h.HasEnd() {
			if !selected.After(h.PeriodEnd) {
				return &day, true
			}
			continue
		}

		limit := ref
		if i+1 < len(confirmed) {
			if dayBeforeNext := confirmed[i+1].PeriodStart.AddDays(-1); dayBeforeNext.Before(limit) {
				limit = dayBeforeNext
			}
		}
		if !selected.After(limit) {
			return &day, false
		}
	}
	return nil, false
}

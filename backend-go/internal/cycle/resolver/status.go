package resolver

import (
	"slices"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Status is CycleStatus.php: the resolved v1.1 status of one target date (task.md §35).
// Nullable PHP ints are *int; nullable Carbon dates are the zero civildate.Date.
type Status struct {
	Date                        civildate.Date
	CycleDay                    *int
	MainPhase                   enums.MainPhase
	Subphase                    enums.CycleSubphase
	FertilityLevel              enums.FertilityLevel
	DaysToOvulation             *int
	DaysToPeriod                *int
	DaysLate                    *int
	CurrentPeriodStart          civildate.Date
	CurrentPeriodStartSource    enums.PeriodStartSource
	CurrentPeriodEnd            civildate.Date
	CurrentPeriodEndSource      enums.PeriodEndSource
	CurrentPeriodEndIsConfirmed bool
	PredictedNextPeriodStart    civildate.Date
	EstimatedOvulationDate      civildate.Date
	EffectiveCycleLength        int
	EffectivePeriodLength       int
	CycleVariability            *int
	Confidence                  enums.ConfidenceLevel
	ConfidenceReasons           []string
	DataQuality                 enums.DataQualityLevel
	ResolutionSource            enums.ResolutionSource
	RequiresUserInput           bool
	Warnings                    []string
}

// Anchors is the §35 `anchors` object.
type Anchors struct {
	CurrentPeriodStart          civildate.Date          `json:"current_period_start"`
	CurrentPeriodStartSource    enums.PeriodStartSource `json:"current_period_start_source"`
	CurrentPeriodEnd            civildate.Date          `json:"current_period_end"`
	CurrentPeriodEndSource      enums.PeriodEndSource   `json:"current_period_end_source"`
	CurrentPeriodEndIsConfirmed bool                    `json:"current_period_end_is_confirmed"`
	PredictedNextPeriodStart    civildate.Date          `json:"predicted_next_period_start"`
	EstimatedOvulationDate      civildate.Date          `json:"estimated_ovulation_date"`
}

// StatusMetrics is the §35 `metrics` object.
type StatusMetrics struct {
	EffectiveCycleLength  int  `json:"effective_cycle_length"`
	EffectivePeriodLength int  `json:"effective_period_length"`
	CycleVariability      *int `json:"cycle_variability"`
}

// StatusAPI is CycleStatus::toApiArray() — the exact §35 structure, fields in PHP key order.
// The zero civildate.Date marshals as null.
type StatusAPI struct {
	Date              civildate.Date         `json:"date"`
	CycleDay          *int                   `json:"cycle_day"`
	MainPhase         enums.MainPhase        `json:"main_phase"`
	Subphase          enums.CycleSubphase    `json:"subphase"`
	FertilityLevel    enums.FertilityLevel   `json:"fertility_level"`
	DaysToOvulation   *int                   `json:"days_to_ovulation"`
	DaysToPeriod      *int                   `json:"days_to_period"`
	DaysLate          *int                   `json:"days_late"`
	Anchors           Anchors                `json:"anchors"`
	Metrics           StatusMetrics          `json:"metrics"`
	Confidence        enums.ConfidenceLevel  `json:"confidence"`
	ConfidenceReasons []string               `json:"confidence_reasons"`
	DataQuality       enums.DataQualityLevel `json:"data_quality"`
	ResolutionSource  enums.ResolutionSource `json:"resolution_source"`
	IsPredicted       bool                   `json:"is_predicted"`
	RequiresUserInput bool                   `json:"requires_user_input"`
	Warnings          []string               `json:"warnings"`
}

// ToAPI is CycleStatus::toApiArray().
func (s Status) ToAPI() StatusAPI {
	return StatusAPI{
		Date:            s.Date,
		CycleDay:        s.CycleDay,
		MainPhase:       s.MainPhase,
		Subphase:        s.Subphase,
		FertilityLevel:  s.FertilityLevel,
		DaysToOvulation: s.DaysToOvulation,
		DaysToPeriod:    s.DaysToPeriod,
		DaysLate:        s.DaysLate,
		Anchors: Anchors{
			CurrentPeriodStart:          s.CurrentPeriodStart,
			CurrentPeriodStartSource:    s.CurrentPeriodStartSource,
			CurrentPeriodEnd:            s.CurrentPeriodEnd,
			CurrentPeriodEndSource:      s.CurrentPeriodEndSource,
			CurrentPeriodEndIsConfirmed: s.CurrentPeriodEndIsConfirmed,
			PredictedNextPeriodStart:    s.PredictedNextPeriodStart,
			EstimatedOvulationDate:      s.EstimatedOvulationDate,
		},
		Metrics: StatusMetrics{
			EffectiveCycleLength:  s.EffectiveCycleLength,
			EffectivePeriodLength: s.EffectivePeriodLength,
			CycleVariability:      s.CycleVariability,
		},
		Confidence:        s.Confidence,
		ConfidenceReasons: nonNil(s.ConfidenceReasons),
		DataQuality:       s.DataQuality,
		ResolutionSource:  s.ResolutionSource,
		IsPredicted:       s.ResolutionSource.IsPredicted(),
		RequiresUserInput: s.RequiresUserInput,
		Warnings:          nonNil(s.Warnings),
	}
}

// nonNil keeps an empty PHP list encoding as [] instead of null.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return slices.Clone(s)
}

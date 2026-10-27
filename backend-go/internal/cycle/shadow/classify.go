// Package shadow is the CYCLE_ENGINE=shadow rollout mode (T-M9-32, DECISIONS.md X-03, docs/engine-v6/README.md §7):
// the request is served by v1 unchanged; after the response a bounded pool computes the v6 status of the same day and
// records how it differs from the served v1.1 cycle_view, so stage can be reviewed before switching to v6.
//
//   - Classify compares the served v1 cycle_view JSON with the v6 status field by field and names the difference
//     classes (Class*). A day can have several.
//   - Pool runs the comparisons on at most PoolSize goroutines; a job that finds both busy is dropped (counted), so
//     shadow mode never queues work and never adds latency to a request.
//   - Recorder counts engine_shadow_diff_total{class}, a 7-day Redis hash of counts and a capped Redis list of the
//     last MaxSamples samples (ritme-go:eng:shadow:samples). A sample carries an HMAC of the user id (never the id),
//     the target's offset from today and, per class, only the fields that class is about.
//   - Report prints the counts and the samples (`api engine-shadow-report`).
package shadow

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/engine"
)

// Class is one difference class (engine_shadow_diff_total{class}).
type Class string

// The difference classes, in report order.
const (
	// ClassOPlusOne: v1.1 calls the day after ovulation fertile (window O−5..O+1), v6 does not (§2.18, D-30).
	ClassOPlusOne Class = "o_plus_one"
	// ClassLateness: days_late differs (v1.1 wraps the cycle, v6 counts the §2.22 lateness tiers).
	ClassLateness Class = "lateness"
	// ClassWarnings: the warning sets differ.
	ClassWarnings Class = "warnings"
	// ClassConfidence: the confidence differs.
	ClassConfidence Class = "confidence"
	// ClassPhase: main_phase or subphase differ (other than on O+1).
	ClassPhase Class = "phase"
	// ClassOnboardingAnchor: the current start source differs where one side is the onboarding anchor (§2.41, D-123).
	ClassOnboardingAnchor Class = "onboarding_anchor"
	// ClassSuppression: v6 suppresses fertility (E-24) and the fertility read-out differs.
	ClassSuppression Class = "suppression"
	// ClassNewFieldsOnly: every shared field agrees; v6 only adds fields.
	ClassNewFieldsOnly Class = "new_fields_only"
	// ClassOther: another shared field differs (cycle_day, days_to_period, anchors, metrics, data_quality, …).
	ClassOther Class = "other"
)

// Classes lists every class in report order.
func Classes() []Class {
	return []Class{ClassOPlusOne, ClassLateness, ClassWarnings, ClassConfidence, ClassPhase, ClassOnboardingAnchor,
		ClassSuppression, ClassNewFieldsOnly, ClassOther}
}

// v1View is the part of the v1.1 cycle_view (resolver.StatusAPI) the comparison reads.
type v1View struct {
	CycleDay        *int     `json:"cycle_day"`
	MainPhase       string   `json:"main_phase"`
	Subphase        string   `json:"subphase"`
	FertilityLevel  string   `json:"fertility_level"`
	DaysToOvulation *int     `json:"days_to_ovulation"`
	DaysToPeriod    *int     `json:"days_to_period"`
	DaysLate        *int     `json:"days_late"`
	Confidence      string   `json:"confidence"`
	Reasons         []string `json:"confidence_reasons"`
	DataQuality     string   `json:"data_quality"`
	Resolution      string   `json:"resolution_source"`
	RequiresInput   bool     `json:"requires_user_input"`
	Warnings        []string `json:"warnings"`
	Anchors         struct {
		Start          *string `json:"current_period_start"`
		StartSource    string  `json:"current_period_start_source"`
		End            *string `json:"current_period_end"`
		EndSource      string  `json:"current_period_end_source"`
		EndIsConfirmed bool    `json:"current_period_end_is_confirmed"`
		Next           *string `json:"predicted_next_period_start"`
		Ovulation      *string `json:"estimated_ovulation_date"`
	} `json:"anchors"`
	Metrics struct {
		CycleLength  int  `json:"effective_cycle_length"`
		PeriodLength int  `json:"effective_period_length"`
		Variability  *int `json:"cycle_variability"`
	} `json:"metrics"`
	Predictions *struct {
		FertileEnd *string `json:"fertile_window_end"`
	} `json:"predictions"`
}

// v1Data is the served v1 `data` object: the legacy engine's calculation (its cycle day wraps, so a late cycle
// restarts at day 1) and the v1.1 cycle_view.
type v1Data struct {
	Calculation struct {
		CycleDay *int `json:"cycle_day"`
		Fertile  bool `json:"is_fertile_window"`
	} `json:"calculation"`
	CycleView v1View `json:"cycle_view"`
}

// Diff is the outcome of one comparison: the classes and, per class, the compared values (only what the class is
// about; ClassOther lists field names, never values).
type Diff struct {
	Classes []Class        `json:"classes"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// pair is a [v1, v6] value pair in a sample.
func pair(v1, v6 any) [2]any { return [2]any{v1, v6} }

// Classify compares the served v1 `data` object ({calculation, cycle_view, …} as JSON) with the v6 status of the same
// day.
//
//   - o_plus_one: the target is the day after ovulation (v1.1 or v6) and v1 still calls it fertile — in the v1.1
//     phase or in its predicted window O−5..O+1 (`predictions.fertile_window_end` ≥ target) — while v6 does not.
//   - lateness: v6 counts the day late (days_late > 0) and v1 disagrees on days_late or on the cycle day (the legacy
//     calculation wraps into a new cycle).
func Classify(v1JSON []byte, v6 *engine.Status) (Diff, error) {
	var data v1Data
	if err := json.Unmarshal(v1JSON, &data); err != nil {
		return Diff{}, fmt.Errorf("shadow: decode v1 data: %w", err)
	}
	v1 := &data.CycleView
	d := Diff{Fields: map[string]any{}}
	add := func(c Class) { d.Classes = append(d.Classes, c) }

	main6, sub6, fert6 := v6.MainPhase.String(), v6.Subphase.String(), v6.FertilityLevel.String()
	dto6 := optPtr(v6.DaysToOvulation)

	isOPlusOne := eqInt(v1.DaysToOvulation, ptr(-1)) || eqInt(dto6, ptr(-1))
	v1Window := v1.Predictions != nil && v1.Predictions.FertileEnd != nil && *v1.Predictions.FertileEnd >= v6.Date.String()
	oPlusOne := isOPlusOne && main6 != "fertile" && (v1.MainPhase == "fertile" || data.Calculation.Fertile || v1Window)
	if oPlusOne {
		add(ClassOPlusOne)
		d.Fields["main_phase"] = pair(v1.MainPhase, main6)
		d.Fields["v1_window_covers_day"] = v1Window
	}
	late6 := optPtr(v6.DaysLate)
	cd6 := optPtr(v6.CycleDay)
	if v6.DaysLate.Or(0) > 0 && (!eqInt(v1.DaysLate, late6) || !eqInt(data.Calculation.CycleDay, cd6)) {
		add(ClassLateness)
		d.Fields["days_late"] = pair(v1.DaysLate, late6)
		d.Fields["cycle_day"] = pair(data.Calculation.CycleDay, cd6)
	}
	if w6 := warnings(v6.Warnings); !sameSet(v1.Warnings, w6) {
		add(ClassWarnings)
		d.Fields["warnings_added"] = minus(w6, v1.Warnings)
		d.Fields["warnings_removed"] = minus(v1.Warnings, w6)
	}
	if c6 := v6.Confidence.String(); v1.Confidence != c6 {
		add(ClassConfidence)
		d.Fields["confidence"] = pair(v1.Confidence, c6)
	}
	if !oPlusOne && (v1.MainPhase != main6 || v1.Subphase != sub6) {
		add(ClassPhase)
		d.Fields["main_phase"] = pair(v1.MainPhase, main6)
		d.Fields["subphase"] = pair(v1.Subphase, sub6)
	}
	src6 := v6.Anchors.CurrentPeriodStartSource.String()
	if v1.Anchors.StartSource != src6 && (v1.Anchors.StartSource == "onboarding_declared" || src6 == "onboarding_declared") {
		add(ClassOnboardingAnchor)
		d.Fields["current_period_start_source"] = pair(v1.Anchors.StartSource, src6)
	}
	suppression := v6.Limits.FertilitySuppressed && v1.FertilityLevel != fert6
	if suppression {
		add(ClassSuppression)
		d.Fields["fertility_level"] = pair(v1.FertilityLevel, fert6)
		d.Fields["suppression_reason"] = v6.Limits.SuppressionReason.String()
	}

	var other []string
	check := func(field string, differs bool) {
		if differs {
			other = append(other, field)
		}
	}
	check("cycle_day", !eqInt(v1.CycleDay, cd6))
	check("days_late", !eqInt(v1.DaysLate, late6) && !slices.Contains(d.Classes, ClassLateness))
	check("days_to_period", !eqInt(v1.DaysToPeriod, optPtr(v6.DaysToPeriod)))
	check("days_to_ovulation", !oPlusOne && !eqInt(v1.DaysToOvulation, dto6))
	check("fertility_level", !oPlusOne && !suppression && v1.FertilityLevel != fert6)
	a := &v6.Anchors
	check("anchors.current_period_start", !eqDay(v1.Anchors.Start, a.CurrentPeriodStart))
	check("anchors.current_period_start_source", v1.Anchors.StartSource != src6 && !slices.Contains(d.Classes, ClassOnboardingAnchor))
	check("anchors.current_period_end", !eqDay(v1.Anchors.End, a.CurrentPeriodEnd))
	check("anchors.current_period_end_source", v1.Anchors.EndSource != a.CurrentPeriodEndSource.String())
	check("anchors.current_period_end_is_confirmed", v1.Anchors.EndIsConfirmed != a.CurrentPeriodEndIsConfirmed)
	check("anchors.predicted_next_period_start", !eqDay(v1.Anchors.Next, a.PredictedNextPeriodStart))
	check("anchors.estimated_ovulation_date", !eqDay(v1.Anchors.Ovulation, a.EstimatedOvulationDate))
	m := &v6.Metrics
	check("metrics.effective_cycle_length", v1.Metrics.CycleLength != m.EffectiveCycleLength)
	check("metrics.effective_period_length", v1.Metrics.PeriodLength != m.EffectivePeriodLength)
	check("metrics.cycle_variability", !eqInt(v1.Metrics.Variability, optPtr(m.CycleVariability)))
	check("data_quality", v1.DataQuality != v6.DataQuality.String())
	check("resolution_source", v1.Resolution != v6.ResolutionSource.String())
	check("requires_user_input", v1.RequiresInput != v6.RequiresUserInput)
	check("confidence_reasons", !sameSet(v1.Reasons, reasons(v6.ConfidenceReasons)))
	if len(other) > 0 {
		add(ClassOther)
		d.Fields["other"] = other
	}
	if len(d.Classes) == 0 {
		add(ClassNewFieldsOnly)
	}
	if len(d.Fields) == 0 {
		d.Fields = nil
	}
	return d, nil
}

func ptr(n int) *int { return &n }

func optPtr(o engine.Opt[int]) *int {
	if v, ok := o.Get(); ok {
		return &v
	}
	return nil
}

func eqInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func eqDay(v1 *string, d engine.Day) bool {
	if v1 == nil || !d.IsSet() {
		return v1 == nil && !d.IsSet()
	}
	return *v1 == d.String()
}

func warnings(s engine.WarningSet) []string {
	out := make([]string, 0, s.Len())
	for w := range s.All() {
		out = append(out, w.String())
	}
	return out
}

func reasons(s engine.ReasonSet) []string {
	out := make([]string, 0, s.Len())
	for r := range s.All() {
		out = append(out, r.String())
	}
	return out
}

func sameSet(a, b []string) bool {
	return len(minus(a, b)) == 0 && len(minus(b, a)) == 0
}

// minus is a \ b (in a's order), never nil.
func minus(a, b []string) []string {
	out := []string{}
	for _, v := range a {
		if !slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	return out
}
